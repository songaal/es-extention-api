package extentions

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/danawalab/es-extention-api/src/model"
	"github.com/danawalab/es-extention-api/src/utils"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
	"github.com/olivere/elastic/v7"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

const (
	JoinField = "join"
	usage = "{\n\"usage\":  \nGET /parent-index/_left\n{\n  \"query\": {\n    \"bool\": {\n      \"must\": [\n        {\n          \"term\": {\n            \"pk\": {\n              \"value\": \"PK_00003\"\n            }\n          }\n        }\n      ]\n    }\n  },\n  \"join\": {\n    \"index\": \"child-index\",\n    \"parent\": \"parent-field\",\n    \"child\": \"child-field\",\n    \"query\": {\n      \"bool\": {\n        \"must\": [\n          {\n            \"term\": {\n              \"ref.keyword\": {\n                \"value\": \"REF_00003\"\n              }\n            }\n          }\n        ]\n      }\n    }\n  }\n}\n}"
)

func Left(res http.ResponseWriter, req *http.Request) {
	defer func() {
		v := recover()
		if v != nil {
			log.Println("error:", v)
			res.WriteHeader(400)
			res.Write([]byte("{\"error\": " + fmt.Sprintln(v) + "}"))
		}

	}()

	reqJson, _ := json.Marshal(req)
	log.Println("left : " + string(reqJson))


	// Parent 인덱스 조회
	vars := mux.Vars(req)
	indices := vars["indices"]

	var leftRequest map[string]interface{}

	// Request Body 파싱
	defer req.Body.Close()
	read, _ := ioutil.ReadAll(req.Body)
	err := json.Unmarshal(read, &leftRequest)
	if err != nil {
		res.WriteHeader(400)
		res.Write([]byte("{\"error\": \"" + err.Error() + "\"}"))
		return
	}

	// Join 필드 추출.
	var leftJoinList []model.LeftJoin
	if utils.TypeOf(leftRequest[JoinField]) == "list" {
		mapstructure.Decode(leftRequest[JoinField], &leftJoinList)
	} else if utils.TypeOf(leftRequest[JoinField]) == "object" {
		// Child Left Join 필드 추출
		leftJoin := model.LeftJoin{}
		mapstructure.Decode(leftRequest[JoinField], &leftJoin)
		leftJoinList = append(leftJoinList, leftJoin)
	}

	// child 쿼리는 제거.
	delete(leftRequest, JoinField)
	parentQuery := leftRequest

	// 조회 건 수 기본값 할당.
	if parentQuery["from"] == nil {
		parentQuery["from"] = 0
	}
	if parentQuery["size"] == nil {
		parentQuery["size"] = 20
	}

	// parent indices 존재 여부
	existsIndices(indices)

	// Parent 엘라스틱 서치 조회
	parentResult, err := EsClient.Search().
		Index(indices).
		Timeout("60s").
		Source(parentQuery).
		Do(context.TODO())
	if err != nil {
		log.Println(err)
		panic(err.Error())
		return
	}

	// parent 매핑 값 추출. (중복 제거)
	var list [][]string
	for _, parentElement := range parentResult.Hits.Hits {
		tmpSource := make(map[string]interface{}, 0)
		json.Unmarshal(parentElement.Source, &tmpSource)
		for i, childElement := range leftJoinList {
			parentKey := childElement.Parent
			childKey := childElement.Child
			val := fmt.Sprint(tmpSource[parentKey])

			if len(list) <= i {
				list = append(list, []string{})
			}

			// child index 존재 확인.
			existsIndices(childElement.Index)

			// 데이터 검증
			if len(parentKey) == 0 || len(childKey) == 0 {
				log.Println("invalid key. parentKey: ", parentKey, ", childKey: ", childKey)
				panic(usage)
				return
			}

			// list에 각각 관계 값 적재함.
			if tmpSource[parentKey] != nil && utils.Contains(list[i], val) == false {
				list[i] = append(list[i], val)
			}
		}
	}
	if len(list) > 0 {

		for index, childElement := range leftJoinList {
			// child 쿼리 ES 조회
			childQuery := make(map[string]interface{}, 1)
			boolQuery := make(map[string]interface{}, 1)
			mustQuery := make(map[string]interface{}, 1)
			var must []interface{}
			termsQuery := make(map[string]interface{}, 1)

			terms := make(map[string]interface{}, 1)
			terms[childElement.Child] = list[index]
			termsQuery["terms"] = terms
			must = append(must, termsQuery)

			if len(childElement.Query) > 0 {
				// 커스텀 쿼리가 있을 경우
				must = append(must, childElement.Query)
			}

			mustQuery["must"] = must
			boolQuery["bool"] = mustQuery
			childQuery["query"] = boolQuery

			printJson, _ := json.Marshal(childQuery)
			log.Println(string(printJson))

			childResult, err := EsClient.Search().
				Index(childElement.Index).
				Timeout("60s").
				Source(childQuery).
				From(0).
				Size(10000).
				Do(context.TODO())
			if err != nil {
				panic(err.Error())
				return
			}


			// 결과 조합
			for _, parent := range parentResult.Hits.Hits {
				parentSource := make(map[string]interface{}, 0)
				json.Unmarshal(parent.Source, &parentSource)
				var searchHitInnerHits elastic.SearchHitInnerHits

				var searchHits elastic.SearchHits
				var searchHit []*elastic.SearchHit
				var totalHits elastic.TotalHits
				var maxScore float64

				maxScore = 0.0
				totalHits.Value = 0
				totalHits.Relation = "eq"

				for _, child := range childResult.Hits.Hits {
					child.Parent = strconv.Itoa(index)
					childSource := make(map[string]interface{}, 0)
					json.Unmarshal(child.Source, &childSource)

					if parentSource[childElement.Parent] == nil || childSource[childElement.Child] == nil {
						continue
					}
					if parentSource[childElement.Parent] == childSource[childElement.Child] {
						// 스코어 갱신
						if maxScore < *child.Score {
							maxScore = *child.Score
						}
						// 동일한 연결 적재
						searchHit = append(searchHit, &*child)
						// 적재 갯수 증가
						totalHits.Value += 1
					}
				}

				parentSearchHitInnerHits := parent.InnerHits["_child"]
				if parentSearchHitInnerHits != nil {
					tmpParentHits := parentSearchHitInnerHits.Hits
					tmpParentHits.TotalHits.Value += totalHits.Value
					tmpParentMaxScore := tmpParentHits.MaxScore
					if maxScore > *tmpParentMaxScore {
						tmpParentHits.MaxScore = &maxScore
					}
					for _, h := range searchHit {
						tmpParentHits.Hits = append(tmpParentHits.Hits, h)
					}
				} else {
					searchHits.TotalHits = &totalHits
					searchHits.MaxScore = &maxScore
					searchHits.Hits = searchHit

					searchHitInnerHits.Hits = &searchHits
					if parent.InnerHits == nil {
						innerHits := make(map[string]*elastic.SearchHitInnerHits, 0)
						parent.InnerHits = innerHits
					}
					parent.InnerHits["_child"] = &searchHitInnerHits
				}
			}
		}
	}

	response, _ := json.MarshalIndent(parentResult, "", "  ")
	res.WriteHeader(200)
	res.Write(response)
	return
}

func existsIndices(indices string) {
	parentExists, _ := EsClient.Exists().Index(indices).Do(context.TODO())
	if parentExists {
		panic("{\n  \"error\" : {\n    \"root_cause\" : [\n      {\n        \"type\" : \"index_not_found_exception\",\n        \"reason\" : \"no such index [" + indices + "]\",\n        \"resource.type\" : \"index_or_alias\",\n        \"resource.id\" : \"" + indices + " \",\n        \"index_uuid\" : \"_na_\",\n        \"index\" : \"" + indices + "\"\n      }\n    ],\n    \"type\" : \"index_not_found_exception\",\n    \"reason\" : \"no such index [" + indices + "]\",\n    \"resource.type\" : \"index_or_alias\",\n    \"resource.id\" : \"" + indices + "\",\n    \"index_uuid\" : \"_na_\",\n    \"index\" : \"" + indices + "\"\n  },\n  \"status\" : 404\n}\n")
	}
}
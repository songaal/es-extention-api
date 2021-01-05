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
)

const (
	JoinField = "join"
)

func Left(res http.ResponseWriter, req *http.Request) {
	defer func() {
		v := recover()
		if v != nil {
			log.Println("error:", v)
		}

	}()

	// Parent 인덱스 조회
	vars := mux.Vars(req)
	indices := vars["indices"]
	log.Println("indices: ", indices)

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

	// Child Left Join 필드 추출
	leftJoin := model.LeftJoin{}
	mapstructure.Decode(leftRequest[JoinField], &leftJoin)
	delete(leftRequest, JoinField)

	// 데이터 검증
	if len(leftJoin.Parent) == 0 || len(leftJoin.Child) == 0 {
		res.WriteHeader(400)
		res.Write([]byte("{\"usage\": \" GET /parent-index/_left\n{\n  \"query\": {\n    \"bool\": {\n      \"must\": [\n        {\n          \"term\": {\n            \"pk.keyword\": {\n              \"value\": \"PK_00003\"\n            }\n          }\n        }\n      ]\n    }\n  },\n  \"join\": {\n    \"index\": \"child-index\",\n    \"parent\": \"parent-field\",\n    \"child\": \"child-field\",\n    \"query\": {\n      \"bool\": {\n        \"must\": [\n          {\n            \"term\": {\n              \"ref.keyword\": {\n                \"value\": \"REF_00003\"\n              }\n            }\n          }\n        ]\n      }\n    }\n  }\n}\"}"))
		return
	}

	// Parent 엘라스틱 서치 조회
	parentResult, err := EsClient.Search().
		Index(indices).
		Timeout("60s").
		Pretty(true).
		Source(leftRequest).
		Do(context.TODO())
	if err != nil {
		res.WriteHeader(400)
		res.Write([]byte("{\"error\": \"" + err.Error() + "\"}"))
		return
	}

	// 매핑 값 추출. (중복 제거)
	var list []string
	for _, element := range parentResult.Hits.Hits {
		tmpSource := make(map[string]interface{}, 0)
		json.Unmarshal(element.Source, &tmpSource)
		if tmpSource[leftJoin.Parent] != nil {
			parentKey := leftJoin.Parent
			val := fmt.Sprint(tmpSource[parentKey])
			if utils.Contains(list, val) == false {
				list = append(list, val)
			}
		}
	}

	if len(list) > 0 {
		// child 쿼리 ES 조회
		childQuery := make(map[string]interface{}, 1)
		boolQuery := make(map[string]interface{}, 1)
		mustQuery := make(map[string]interface{}, 1)

		terms := make(map[string]interface{}, 1)
		terms[leftJoin.Child] = list

		termsQuery := make(map[string]interface{}, 1)
		termsQuery["terms"] = terms

		var must []interface{}
		must = append(must, termsQuery)
		if len(leftJoin.Query) > 0 {
			// 커스텀 쿼리가 있을 경우
			must = append(must, leftJoin.Query)
		}
		mustQuery["must"] = must
		boolQuery["bool"] = mustQuery
		childQuery["query"] = boolQuery

		jsonString, _ := json.Marshal(&childQuery)
		fmt.Println(jsonString)

		childResult, err := EsClient.Search().
			Index(leftJoin.Index).
			Timeout("60s").
			Source(childQuery).
			Pretty(true).
			TrackTotalHits(true).
			Do(context.TODO())
		if err != nil {
			res.WriteHeader(400)
			res.Write([]byte("{\"error\": \"" + err.Error() + "\"}"))
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
				childSource := make(map[string]interface{}, 0)
				json.Unmarshal(child.Source, &childSource)

				log.Println(leftJoin.Parent, parentSource[leftJoin.Parent], childSource[leftJoin.Child], leftJoin.Child)
				if parentSource[leftJoin.Parent] == nil || childSource[leftJoin.Child] == nil {
					continue
				}
				if parentSource[leftJoin.Parent] == childSource[leftJoin.Child] {

					if maxScore < *child.Score {
						maxScore = *child.Score
					}
					searchHit = append(searchHit, &*child)
					totalHits.Value += 1
					totalHits.Relation = "eq"
				}
			}

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

	response, _ := json.MarshalIndent(parentResult, "", "  ")
	res.WriteHeader(200)
	res.Write(response)
	return
}

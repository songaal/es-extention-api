package extentions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/danawalab/es-extention-api/src/model"
	"github.com/danawalab/es-extention-api/src/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/olivere/elastic/v7"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func Left(indices string, leftRequest map[string]interface{}, res http.ResponseWriter, req *http.Request) {
	defer func() {
		v := recover()
		if v != nil {
			log.Println("error:", v)
			res.WriteHeader(400)
			_, _ = res.Write([]byte("{\"error\": " + fmt.Sprintln(v) + "}"))
		}
	}()

	//reqJson, _ := json.Marshal(req)
	//log.Println("left : " + string(reqJson))

	// Parent 인덱스 조회
	//vars := mux.Vars(req)
	//indices := vars["indices"]

	//var leftRequest map[string]interface{}

	// Request Body 파싱
	//defer req.Body.Close()
	//read, _ := ioutil.ReadAll(req.Body)
	//err := json.Unmarshal(read, &leftRequest)
	//if err != nil {
	//	res.WriteHeader(400)
	//	_, _ = res.Write([]byte("{\"error\": \"" + err.Error() + "\"}"))
	//	return
	//}

	// Join 필드 추출.
	originJoinList := make([]map[string]interface{}, 0)
	leftJoinList := make([]model.LeftJoin, 0)
	if utils.TypeOf(leftRequest["join"]) == "list" {
		// join 여러개
		tmpJoinList := make([]model.TmpLeftJoin, 0)
		_ = mapstructure.Decode(leftRequest["join"], &tmpJoinList)

		for _, val := range tmpJoinList {
			leftJoin := model.LeftJoin{}
			leftJoin.Index = val.Index
			leftJoin.Parent = strings.Split(val.Parent, ",")
			leftJoin.Child = strings.Split(val.Child, ",")
			leftJoin.Host = val.Host
			leftJoin.Username = val.Username
			leftJoin.Password = val.Password
			leftJoin.From = val.From
			leftJoin.Size = val.Size
			leftJoin.Query = val.Query
			if val.Source != nil {
				leftJoin.Source = val.Source
			}
			leftJoinList = append(leftJoinList, leftJoin)

			originJoin := make(map[string]interface{})
			originJoin["index"] = val.Index
			originJoin["parent"] = val.Parent
			originJoin["child"] = strings.Split(val.Parent, ",")
			originJoin["from"] = strings.Split(val.Child, ",")
			originJoin["size"] = val.Size
			originJoin["query"] = val.Query
			if val.Source != nil {
				originJoin["source"] = val.Source
			}
			originJoinList = append(originJoinList, originJoin)
		}
	} else if utils.TypeOf(leftRequest["join"]) == "object" {
		// parent, child 필드 추출
		tmpLeftJoinMap := make(map[string]interface{}, 0)
		_ = mapstructure.Decode(leftRequest["join"], &tmpLeftJoinMap)
		tmpParentList := make([]string, 0)
		tmpChildList := make([]string, 0)
		if utils.TypeOf(tmpLeftJoinMap["parent"]) == "list" && utils.TypeOf(tmpLeftJoinMap["child"]) == "list" {
			_ = mapstructure.Decode(tmpLeftJoinMap["parent"], &tmpParentList)
			_ = mapstructure.Decode(tmpLeftJoinMap["child"], &tmpChildList)
		} else if utils.TypeOf(tmpLeftJoinMap["parent"]) == "string" && utils.TypeOf(tmpLeftJoinMap["child"]) == "string"{
			tmpParentList = append(tmpParentList, fmt.Sprintf("%v", tmpLeftJoinMap["parent"]))
			tmpChildList = append(tmpChildList, fmt.Sprintf("%v", tmpLeftJoinMap["child"]))
		} else {
			panic("Invalid parent and child values. USAGE: " + usage)
			return
		}

		// Child Left Join 필드 추출
		leftJoin := model.LeftJoin{}
		_ = mapstructure.Decode(leftRequest["join"], &leftJoin)
		leftJoin.Parent = tmpParentList
		leftJoin.Child = tmpChildList
		leftJoinList = append(leftJoinList, leftJoin)
		originJoinList = append(originJoinList, tmpLeftJoinMap)
	}

	// 메인 쿼리에서 join 필드 제거.
	delete(leftRequest, "join")
	delete(leftRequest, TypeField)
	parentQuery := leftRequest

	// 조회 건 수 기본값 할당.
	if parentQuery["from"] == nil {
		parentQuery["from"] = 0
	}

	if parentQuery["size"] == nil {
		parentQuery["size"] = 20
	}

	parentSource := make(map[string][]string, 0)
	parentSource["includes"] = []string{}
	parentSource["excludes"] = []string{}

	for _, l := range leftJoinList {
		parentSource["includes"] = append(parentSource["includes"], l.Parent...)
	}

	isHideHits := false
	if utils.TypeOf(parentQuery["_source"]) == "list" {
		source := make([]string, 0)
		_ = mapstructure.Decode(parentQuery["_source"], &source)
		parentSource["includes"] = append(parentSource["includes"], source...)
		parentQuery["_source"] = parentSource
	} else if utils.TypeOf(parentQuery["_source"]) == "object" {
		includes := make([]string, 0)
		excludes := make([]string, 0)
		source := make(map[string]interface{}, 0)
		_ = mapstructure.Decode(parentQuery["_source"], &source)
		if source["includes"] != nil {
			_ = mapstructure.Decode(source["includes"], &includes)
		}
		if source["excludes"] != nil {
			_ = mapstructure.Decode(source["excludes"], &excludes)
		}
		parentSource["includes"] = append(parentSource["includes"], includes...)
		parentSource["excludes"] = append(parentSource["excludes"], excludes...)
		parentQuery["_source"] = parentSource
	} else if utils.TypeOf(parentQuery["_source"]) == "string" {
		parentSource["includes"] = append(parentSource["includes"], fmt.Sprintf("%v", parentQuery["_source"]))
		parentQuery["_source"] = parentSource
	} else if utils.TypeOf(parentQuery["_source"]) == "bool" && parentQuery["_source"] == false {
		parentQuery["_source"] = parentSource
		isHideHits = true
	} else {
		parentQuery["_source"] = true
	}

	// parent indices 존재 여부
	//ExistsIndices(parentClient, indices)

	// Parent 엘라스틱 서치 조회
	parentResult, err := DefaultClient.Search().
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
	// filed seq: [val1, val2, val3 ....]
	leftJoinOnWhere := make([]map[string][]string, len(leftJoinList))
	// Parent 조회
	for _, parentElement := range parentResult.Hits.Hits {
		tmpSource := make(map[string]interface{}, 0)
		_ = json.Unmarshal(parentElement.Source, &tmpSource)

		for index, childElement := range leftJoinList {
			// child index 존재 확인.
			//ExistsIndices(childClient, childElement.Index)

			parentKey := childElement.Parent
			childKey := childElement.Child

			if len(parentKey) == 0 || len(childKey) == 0 {
				panic("There is at least 1 parent key and child key.")
				return
			} else if len(parentKey) != len(childKey) {
				panic("The number of parent and child keys does not match.")
				return
			}

			for pi, k := range parentKey {
				strParentKey := fmt.Sprintf("%v", k)
				if len(strParentKey) == 0 {
					// invalid usage
					panic("The parent key cannot be empty.")
					return
				}
				// parent key 순서와 child key 순서에 맞게 매칭함.
				strChildKey := fmt.Sprintf("%v", childKey[pi])
				if len(strChildKey) == 0 {
					panic("The child key cannot be empty.")
					return
				}
				//parent 값을 적재. child 검색 쿼리로 조합 목적
				parentRefValue := fmt.Sprint(tmpSource[strParentKey])
				if leftJoinOnWhere[index] == nil {
					leftJoinOnWhere[index] = map[string][]string{}
				}

				if utils.Contains(leftJoinOnWhere[index][strChildKey], parentRefValue) == false {
					leftJoinOnWhere[index][strChildKey] = append(leftJoinOnWhere[index][strChildKey], parentRefValue)
				}
			}
		}
	}

	totalAggregations := map[string]json.RawMessage{}

	if parentResult.Aggregations != nil {
		for k, v := range parentResult.Aggregations {
			totalAggregations[k] = v
		}
	}

	for index, childElement := range leftJoinList {
		childFrom := 0
		childSize := parentQuery["size"]
		if childElement.From > 0 {
			childFrom = childElement.From
		}
		if childElement.Size > 0 {
			childSize = childElement.Size
		}

		childClient := DefaultClient
		if childElement.Host != "" {
			childClient, _ = GetClient(childElement.Host, childElement.Username, childElement.Password)
		}

		// child 쿼리 ES 조회
		boolQuery := make(map[string]interface{}, 1)
		mustQuery := make(map[string]interface{}, 1)
		var must []interface{}
		//var filter []interface{}

		for _, childKey := range childElement.Child {
			if len(leftJoinOnWhere[index][childKey]) > 0 {
				termsQuery := make(map[string]interface{}, 1)
				terms := make(map[string]interface{}, 1)
				terms[childKey] = leftJoinOnWhere[index][childKey]
				termsQuery["terms"] = terms
				must = append(must, termsQuery)
			}
		}

		if len(childElement.Query) > 0 {
			// 커스텀 쿼리가 있을 경우
			must = append(must, childElement.Query)
		}

		mustQuery["must"] = must
		boolQuery["bool"] = mustQuery

		delete(originJoinList[index], "index")
		delete(originJoinList[index], "parent")
		delete(originJoinList[index], "child")
		delete(originJoinList[index], "query")

		delete(originJoinList[index], "host")
		delete(originJoinList[index], "username")
		delete(originJoinList[index], "password")

		delete(originJoinList[index], "from")
		delete(originJoinList[index], "size")

		originJoinList[index]["query"] = boolQuery
		originJoinList[index]["from"] = childFrom
		originJoinList[index]["size"] = childSize


		childSource := make(map[string][]string, 0)
		childSource["includes"] = []string{}
		childSource["excludes"] = []string{}
		childSource["includes"] = append(childSource["includes"], childElement.Child...)
		if utils.TypeOf(originJoinList[index]["_source"]) == "list" {
			source := make([]string, 0)
			_ = mapstructure.Decode(originJoinList[index]["_source"], &source)
			childSource["includes"] = append(childSource["includes"], source...)
			originJoinList[index]["_source"] = childSource
		} else if utils.TypeOf(originJoinList[index]["_source"]) == "object" {
			includes := make([]string, 0)
			excludes := make([]string, 0)
			source := make(map[string]interface{}, 0)
			_ = mapstructure.Decode(originJoinList[index]["_source"], &source)
			if source["includes"] != nil {
				_ = mapstructure.Decode(source["includes"], &includes)
			}
			if source["excludes"] != nil {
				_ = mapstructure.Decode(source["excludes"], &excludes)
			}
			childSource["includes"] = append(childSource["includes"], includes...)
			childSource["excludes"] = append(childSource["excludes"], excludes...)
			originJoinList[index]["_source"] = childSource
		} else if utils.TypeOf(originJoinList[index]["_source"]) == "string" {
			childSource["includes"] = append(childSource["includes"], fmt.Sprintf("%v", originJoinList[index]["_source"]))
			originJoinList[index]["_source"] = childSource
		} else if utils.TypeOf(originJoinList[index]["_source"]) == "bool" && originJoinList[index]["_source"] == false {
			originJoinList[index]["_source"] = childSource
		} else {
			originJoinList[index]["_source"] = true
		}

		printJson, _ := json.Marshal(originJoinList)
		log.Println(string(printJson))

		childResult, err := childClient.Search().
			Index(childElement.Index).
			Timeout("60s").
			Source(originJoinList[index]).
			Do(context.TODO())
		if err != nil {
			fmt.Println(err)
			panic(err.Error())
			return
		}
		log.Println("child Count:", childResult.TotalHits())

		childResults := make(map[string][]*elastic.SearchHit, 0)
		maxScoreMap := make(map[string]float64)
		for _, child := range childResult.Hits.Hits {
			childSource := make(map[string]interface{}, 0)
			_ = json.Unmarshal(child.Source, &childSource)

			// 키 조합.
			var tmpKeyBuf bytes.Buffer
			tmpKeyBuf.WriteString("ref-")
			for _, childKey := range childElement.Child {
				// :: 구분기호로 키조합.
				tmpKeyBuf.WriteString( url.QueryEscape(fmt.Sprintf("%v", childSource[childKey])) + "::" )
			}
			refKey := tmpKeyBuf.String()
			childResults[refKey] = append(childResults[refKey], child)

			if maxScoreMap[refKey] < *child.Score {
				maxScoreMap[refKey] = *child.Score
			}
		}

		if childResult.Aggregations != nil {
			for k, v := range childResult.Aggregations {
				totalAggregations[k] = v
			}
		}

		log.Println("child key Map Length: ", len(childResults))

		for _, parent := range parentResult.Hits.Hits {
			parentSource := make(map[string]interface{}, 0)
			_ = json.Unmarshal(parent.Source, &parentSource)

			// 키 조합.
			var tmpKeyBuf bytes.Buffer
			tmpKeyBuf.WriteString("ref-")
			for _, parentKey := range childElement.Parent {
				// :: 구분기호로 키조합.
				tmpKeyBuf.WriteString(url.QueryEscape(fmt.Sprintf("%v", parentSource[parentKey])) + "::" )
			}
			refKey := tmpKeyBuf.String()
			
			if isHideHits {
				parent.Source = nil
			} else if childResults[refKey] != nil {
				var searchHitInnerHits elastic.SearchHitInnerHits
				var searchHits elastic.SearchHits
				var searchHit []*elastic.SearchHit
				var totalHits elastic.TotalHits
				var maxScore float64

				maxScore = maxScoreMap[refKey]
				totalHits.Value = int64(len(childResults[refKey]))
				totalHits.Relation = "eq"

				searchHit = childResults[refKey]
				searchHits.Hits = searchHit
				searchHits.MaxScore = &maxScore
				searchHits.TotalHits = &totalHits
				searchHitInnerHits.Hits = &searchHits

				// 키 존재 하면 parent innerHit 문서 등록
				if parent.InnerHits == nil {
					// 기존 innerHit 미존재
					innerHits := make(map[string]*elastic.SearchHitInnerHits, 0)
					parent.InnerHits = innerHits
					parent.InnerHits["_child"] = &searchHitInnerHits
				} else {
					// 기존 innerHit 존재
					searchHits := parent.InnerHits["_child"]
					if *searchHits.Hits.MaxScore < maxScoreMap[refKey] {
						*searchHits.Hits.MaxScore = maxScoreMap[refKey]
					}
					searchHits.Hits.TotalHits.Value = searchHits.Hits.TotalHits.Value + int64(len(childResults[refKey]))
					searchHits.Hits.Hits = append(searchHits.Hits.Hits, childResults[refKey]...)
				}
			}
		}
	}

	if totalAggregations != nil && len(totalAggregations) > 0 {
		parentResult.Aggregations = totalAggregations
	}

	response, _ := json.MarshalIndent(parentResult, "", "  ")
	res.WriteHeader(200)
	_, _ = res.Write(response)
	return
}



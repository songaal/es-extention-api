package extentions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/danawalab/es-extention-api/src/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/olivere/elastic/v7"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

/**
 * 엘라스틱서치 검색을 SQL의 inner 조인 함수
 * 검색방법: child 1만건 -> query 생성 -> parent
 */
func Inner(indices string, fullQueryEntity map[string]interface{}) (results elastic.SearchResult) {
	if utils.TypeOf(fullQueryEntity["join"]) != "object" {
		log.Println("child query only object type.")
		panic("child query only object type.")
	}

	// childEntity
	childQueryEntity := make(map[string]interface{})
	b, _ := json.Marshal(fullQueryEntity["join"])
	if e := json.Unmarshal(b, &childQueryEntity); e != nil {
		log.Println(e)
		panic(e)
	}

	// 조인 예약어
	host     := ""
	username := ""
	password := ""
	childIndices  := ""
	parentFields := ""
	childFields :=  ""
	parentHost := ""
	parentUsername := ""
	parentPassword := ""
	if childQueryEntity[HostField]     != nil { host = fmt.Sprintf("%v", childQueryEntity[HostField]) }
	if childQueryEntity[UsernameField] != nil { username = fmt.Sprintf("%v", childQueryEntity[UsernameField]) }
	if childQueryEntity[PasswordField] != nil { password = fmt.Sprintf("%v", childQueryEntity[PasswordField]) }
	if childQueryEntity[IndicesField]  != nil { childIndices = fmt.Sprintf("%v", childQueryEntity[IndicesField]) }
	if childQueryEntity[ParentFields]  != nil { parentFields = fmt.Sprintf("%v", childQueryEntity[ParentFields]) }
	if childQueryEntity[ChildFields]   != nil { childFields = fmt.Sprintf("%v", childQueryEntity[ChildFields]) }

	// parent host
	if fullQueryEntity[HostField]      != nil { parentHost = fmt.Sprintf("%v", fullQueryEntity[HostField]) }
	if fullQueryEntity[UsernameField]  != nil { parentUsername = fmt.Sprintf("%v", fullQueryEntity[UsernameField]) }
	if fullQueryEntity[PasswordField]  != nil { parentPassword = fmt.Sprintf("%v", fullQueryEntity[PasswordField]) }

	// 필수값 체크
	if childIndices == "" {
		log.Println("indices 는 필수 입니다.")
		panic("indices 는 필수값입니다.")
	} else if len(strings.Split(parentFields, ",")) == 0 {
		log.Println("parent, child 는 필수값입니다.")
		panic("parent, child 갯수는 동일해야합니다.")
	} else if len(strings.Split(parentFields, ",")) != len(strings.Split(childFields, ",")) {
		log.Println("parent, child 갯수는 동일해야합니다.")
		panic("parent, child 갯수는 동일해야합니다.")
	}

	// parent, child ES 클라이언트 생성
	pClient := DefaultClient
	if parentHost != "" {
		log.Println("parent ES >>", parentHost, parentUsername, parentPassword)
		pClient, _ = GetClient(parentHost, parentUsername, parentPassword)
	}
	cClient := DefaultClient
	if host != "" {
		log.Println("child ES >>", host, username, password)
		cClient, _ = GetClient(host, username, password)
	}

	// 원본 쿼리에서 검색쿼리 분리
	parentQuery := make(map[string]interface{})
	childQuery := make(map[string]interface{})
	for k, v := range fullQueryEntity {
		if k != "join" &&
			k != TypeField &&
			k != HostField &&
			k != UsernameField &&
			k != PasswordField {
			parentQuery[k] = v
		}
	}
	for k, v := range childQueryEntity {
		if k != ParentFields &&
			k != ChildFields &&
			k != IndicesField &&
			k != HostField &&
			k != UsernameField &&
			k != PasswordField &&
			k != TypeField {
			childQuery[k] = v
		}
	}


	// child 조건절 검색
	if childQuery["size"] == nil {
		childQuery["size"] = 10000
	}
	childQuery["_source"] = true
	//st1 := time.Now().Unix()
	cResp, e := conditionSearchAll(&cClient, childIndices, "hits.hits", "120s", true, childQuery)
	//cResp, e := cClient.Search().
	//	Index(childIndices).
	//	FilterPath("hits.hits").
	//	Timeout("120s").
	//	Source(childQuery).
	//	Do(context.TODO())
	if e != nil {
		log.Println(e, cResp)
		panic(e)
	}
	//nt1 := time.Now().Unix()
	//log.Println("조회 소요시간 " + strconv.Itoa(int(nt1-st1)) + "s")

	if len(cResp.Hits.Hits) == 0 {
		zero := `{"took":1,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}`
		_ = json.Unmarshal([]byte(zero), &results)
		return
	}else {
		log.Println("키 추출 조회 결과 갯수: " + childIndices + ", " + strconv.Itoa(len(cResp.Hits.Hits)))
	}

	// 쿼리 생성
	parentKeyList := strings.Split(parentFields, ",")
	childKeyList := strings.Split(childFields, ",")
	parentQueryJson, _ := json.Marshal(parentQuery["query"])
	termsQueryJson, _ := json.Marshal(getTermsQuery(*cResp.Hits, parentKeyList, childKeyList))
	tempQuery := getTempQuery(string(parentQueryJson), string(termsQueryJson))
	searchQuery := make(map[string]interface{})
	_ = json.Unmarshal([]byte(tempQuery), &searchQuery)
	for k, v := range parentQuery {
		if k != "query" {
			searchQuery[k] = v
		}
	}

	pSource := make(map[string][]string, 0)
	pSource["includes"] = []string{}
	pSource["excludes"] = []string{}
	pSource["includes"] = parentKeyList

	if utils.TypeOf(searchQuery["_source"]) == "list" {
		source := make([]string, 0)
		_ = mapstructure.Decode(searchQuery["_source"], &source)
		pSource["includes"] = append(pSource["includes"], source...)
		searchQuery["_source"] = pSource
	} else if utils.TypeOf(searchQuery["_source"]) == "object" {
		includes := make([]string, 0)
		excludes := make([]string, 0)
		source := make(map[string]interface{}, 0)
		_ = mapstructure.Decode(searchQuery["_source"], &source)
		if source["includes"] != nil {
			_ = mapstructure.Decode(source["includes"], &includes)
		}
		if source["excludes"] != nil {
			_ = mapstructure.Decode(source["excludes"], &excludes)
		}
		pSource["includes"] = append(pSource["includes"], includes...)
		pSource["excludes"] = append(pSource["excludes"], excludes...)
		searchQuery["_source"] = pSource
	} else if utils.TypeOf(searchQuery["_source"]) == "string" {
		pSource["includes"] = append(pSource["includes"], fmt.Sprintf("%v", searchQuery["_source"]))
		searchQuery["_source"] = pSource
	} else if utils.TypeOf(searchQuery["_source"]) == "bool" && searchQuery["_source"] == false {
		searchQuery["_source"] = pSource
	} else {
		searchQuery["_source"] = true
	}

	// parent 조회
	st2 := time.Now().Unix()
	pResp, e := pClient.Search().
		Index(indices).
		Timeout("120s").
		Source(searchQuery).
		Do(context.TODO())
	if e != nil {
		log.Println(e, pResp)
		panic(e)
	}
	nt2 := time.Now().Unix()
	log.Println("메인 쿼리 조회 소요시간 " + strconv.Itoa(int(nt2-st2)) + "s")

	if len(pResp.Hits.Hits) == 0 {
		zero := `{"took":1,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}`
		_ = json.Unmarshal([]byte(zero), &results)
	} else {
		log.Println("메인 쿼리 조회 결과 갯수: " + indices + ", " + strconv.Itoa(len(pResp.Hits.Hits)))
	}

	// inner_hits 추가할 child 조회
	termsQueryJson, _ = json.Marshal(getTermsQuery(*pResp.Hits, childKeyList, parentKeyList))
	childQueryJson, _ := json.Marshal(childQuery["query"])
	tempQuery = getTempQuery(string(childQueryJson), string(termsQueryJson))
	// parent, child 타입이 text 일 경우 안나오는현상 발생
	//tempQuery = getTempQuery(string(childQueryJson), "null")
	searchQuery = make(map[string]interface{})
	_ = json.Unmarshal([]byte(tempQuery), &searchQuery)
	for k, v := range childQuery {
		if k != "query" {
			searchQuery[k] = v
		}
	}

	cSource := make(map[string][]string, 0)
	cSource["includes"] = []string{}
	cSource["excludes"] = []string{}
	cSource["includes"] = childKeyList

	if utils.TypeOf(searchQuery["_source"]) == "list" {
		source := make([]string, 0)
		_ = mapstructure.Decode(searchQuery["_source"], &source)
		cSource["includes"] = append(cSource["includes"], source...)
		searchQuery["_source"] = cSource
	} else if utils.TypeOf(searchQuery["_source"]) == "object" {
		includes := make([]string, 0)
		excludes := make([]string, 0)
		source := make(map[string]interface{}, 0)
		_ = mapstructure.Decode(searchQuery["_source"], &source)
		if source["includes"] != nil {
			_ = mapstructure.Decode(source["includes"], &includes)
		}
		if source["excludes"] != nil {
			_ = mapstructure.Decode(source["excludes"], &excludes)
		}
		cSource["includes"] = append(cSource["includes"], includes...)
		cSource["excludes"] = append(cSource["excludes"], excludes...)
		searchQuery["_source"] = cSource
	} else if utils.TypeOf(searchQuery["_source"]) == "string" {
		cSource["includes"] = append(cSource["includes"], fmt.Sprintf("%v", searchQuery["_source"]))
		searchQuery["_source"] = cSource
	} else if utils.TypeOf(searchQuery["_source"]) == "bool" && searchQuery["_source"] == false {
		searchQuery["_source"] = cSource
	} else {
		searchQuery["_source"] = true
	}

	// child 조회
	st3 := time.Now().Unix()
	cResp, e = conditionSearchAll(&cClient, childIndices, "", "120s", false, searchQuery)
	//cResp, e = cClient.Search().
	//	Index(childIndices).
	//	Timeout("120s").
	//	Source(searchQuery).
	//	Do(context.TODO())
	if e != nil {
		log.Println(e, cResp)
		panic(e)
	}
	nt3 := time.Now().Unix()
	log.Println("메인 쿼리 조회 소요시간 " + strconv.Itoa(int(nt3-st3)) + "s")

	log.Println("서브 쿼리 조회 결과 갯수: " + childIndices + ", " +strconv.Itoa(len(cResp.Hits.Hits)))

	refHits, maxScoreMap := getRefSet(*cResp.Hits, childKeyList)

	// parent 결과에 child 결과 inner_hits 조합
	for _, hit := range pResp.Hits.Hits {
		source := make(map[string]interface{}, 0)
		_ = json.Unmarshal(hit.Source, &source)
		key := convertSrcToKey(source, parentKeyList)

		var searchHitInnerHits elastic.SearchHitInnerHits
		var searchHits elastic.SearchHits
		var searchHit []*elastic.SearchHit
		var totalHits elastic.TotalHits
		var maxScore float64

		if refHits[key] != nil && len(refHits[key]) > 0 {
			totalHits.Value = int64(len(refHits[key]))
			totalHits.Relation = "eq"
			maxScore = maxScoreMap[key]
			searchHit = refHits[key]
			searchHits.Hits = searchHit
			searchHits.MaxScore = &maxScore
			searchHits.TotalHits = &totalHits
			searchHitInnerHits.Hits = &searchHits
		}

		// 키 존재 하면 parent innerHit 문서 등록
		if hit.InnerHits == nil {
			// 기존 innerHit 미존재
			hit.InnerHits = make(map[string]*elastic.SearchHitInnerHits)
			hit.InnerHits["_child"] = &searchHitInnerHits
		} else {
			// 기존 innerHit 존재
			hit.InnerHits["_child"] = &searchHitInnerHits
		}
	}

	results = *pResp

	totalAggregations := map[string]json.RawMessage{}
	if pResp.Aggregations != nil {
		for k, v := range pResp.Aggregations {
			totalAggregations[k] = v
		}
	}
	if cResp.Aggregations != nil {
		for k, v := range cResp.Aggregations {
			totalAggregations[k] = v
		}
	}
	if len(totalAggregations) > 0 {
		results.Aggregations = totalAggregations
	}

	return
}

func getTempQuery(mustQuery, filterQuery string) (tempQuery string){
	if mustQuery == "<nil>" || mustQuery == "null" {
		mustQuery = "[]"
	}
	if filterQuery == "<nil>" || mustQuery == "null" {
		filterQuery = "[]"
	}
	tempQuery = fmt.Sprintf(`
		{ 
			"query": { 
 				"bool": {
					"must": %s,
					"filter": %s
            	}
			}
		}`, mustQuery, filterQuery)
	return
}

func getTermsQuery(hits elastic.SearchHits, keyFields, valueFields []string) (termsWarpQuery []map[string]map[string][]string) {
	tmpFields := make(map[string][]string)
	for _, hit := range hits.Hits {
		source := make(map[string]interface{}, 0)
		_ = json.Unmarshal(hit.Source, &source)
		for i, pKey := range keyFields {
			val := fmt.Sprintf("%v", source[valueFields[i]])
			if source[valueFields[i]] == nil || val == "" {
				continue
			} else if utils.Contains(tmpFields[pKey], val) == false {
				// 새로운 값을 추가한다.
				tmpFields[pKey] = append(tmpFields[pKey], val)
				if len(tmpFields[pKey]) > TermsMaxCount {
					panic(fmt.Sprintf("terms 최대 갯수를 초과함. key: %s  value: %s", pKey, tmpFields[pKey]))
				}
			}
		}
	}
	termsWarpQuery = make([]map[string]map[string][]string, 0)
	for k, v := range tmpFields {
		terms := make(map[string]map[string][]string)
		terms["terms"] = make(map[string][]string)
		terms["terms"][k] = v
		termsWarpQuery = append(termsWarpQuery, terms)
	}
	return
}

func getRefSet(hits elastic.SearchHits, keyFields []string) (refSet map[string][]*elastic.SearchHit, maxScore map[string]float64) {
	refSet = make(map[string][]*elastic.SearchHit)
	maxScore = make(map[string]float64)
	for _, hit := range hits.Hits {
		source := make(map[string]interface{}, 0)
		_ = json.Unmarshal(hit.Source, &source)
		key := convertSrcToKey(source, keyFields)
		if refSet[key] == nil {
			refSet[key] = make([]*elastic.SearchHit, 0)
		}
		if maxScore[key] < *hit.Score {
			maxScore[key] = *hit.Score
		}
		refSet[key] = append(refSet[key], hit)
	}
	return
}

func convertSrcToKey(source map[string]interface{}, fields []string) (key string) {
	// 키 조합.
	var tmpKeyBuf bytes.Buffer
	tmpKeyBuf.WriteString("ref-")
	for _, k := range fields {
		// :: 구분기호로 키조합.
		tmpKeyBuf.WriteString(url.QueryEscape(fmt.Sprintf("%v", source[k])) + "::" )
	}
	key = tmpKeyBuf.String()
	return
}

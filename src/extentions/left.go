package extentions

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/danawalab/es-extention-api/src/model"
	"github.com/danawalab/es-extention-api/src/utils"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
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

	// Parent 엘라스틱 서치 조회
	parentResult, err := EsClient.Search().
		Index(indices).
		Timeout("60s").
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
		val := fmt.Sprint(tmpSource[leftJoin.Parent])

		if len(val) > 0 && utils.Contains(list, val) {
			list = append(list, val)
		}
	}

	if len(list) == 0 {
		// 매핑 값으면 로직 완료.
		pb, _ := json.Marshal(parentResult)
		res.Write(pb)
		return
	} else {

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

		childResult, err := EsClient.Search().
			Index(leftJoin.Index).
			Timeout("60s").
			Source(childQuery).
			TrackTotalHits(true).
			Do(context.TODO())
		if err != nil {
			res.WriteHeader(400)
			res.Write([]byte("{\"error\": \"" + err.Error() + "\"}"))
			return
		}

		//SearchHit := parentResult.Hits.Hits


		log.Println(parentResult, childResult)
		//totalHits := int(parentResult.Hits.TotalHits.Value)


	}

}



func printQueryDsl(src interface{}) {
	data, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}

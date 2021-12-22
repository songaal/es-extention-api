package extentions

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/danawalab/es-extention-api/src/utils"
	"github.com/gorilla/mux"
	"github.com/olivere/elastic/v7"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
)

const (
	TypeField = "type"

	IndicesField = "index"
	ParentFields = "parent"
	ChildFields = "child"

	HostField     = "host"
	UsernameField = "username"
	PasswordField = "password"
	usage         = "{}"
)

func Join(res http.ResponseWriter, req *http.Request) {
	defer func() {
		v := recover()
		if v != nil {
			log.Println("error:", v)
			res.WriteHeader(400)
			_, _ = res.Write([]byte("{\"error\": \"" + fmt.Sprint(v) + "\"}"))
		}
	}()

	// [개발모드] 요청된 로그 출력.
	if os.Getenv("go.env") == "development" {
		reqJson, _ := json.Marshal(req)
		log.Println("Join Query : " + string(reqJson))
	}

	// Request Body 파싱
	fullQuery, e := ParseBody(req.Body)
	if e != nil {
		panic(e)
	}
	// parent 인덱스 조회
	vars := mux.Vars(req)
	indices := vars["indices"]

	// 조인타입에 따라 분기처리.
	if fullQuery[TypeField] != nil && fullQuery[TypeField] == "inner" {
		results := elastic.SearchResult{}
		results = Inner(indices, fullQuery)
		res.WriteHeader(200)
		response := make([]byte, 0)
		if len(results.Hits.Hits) == 0 {
			//zero := `{"took":1,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}`
			zero := `{
  "took" : 1,
  "timed_out" : false,
  "_shards" : {
    "total" : 1,
    "successful" : 1,
    "skipped" : 0,
    "failed" : 0
  },
  "hits" : {
    "total" : {
      "value" : 0,
      "relation" : "eq"
    },
    "max_score" : null,
    "hits" : [ ]
  }
}
`
			response = []byte(zero)
		} else {
			response, _ = json.MarshalIndent(results, "", "  ")
		}
		_, _ = res.Write(response)
	} else {
		Left(indices, fullQuery, res, req)
	}

	return
}

/**
 * 요청받은 데이터를 객체로 변환하는 함수
 */
func ParseBody(body io.ReadCloser) (query map[string]interface{}, err error) {
	read, _ := ioutil.ReadAll(body)
	err = json.Unmarshal(read, &query)
	return
}


func conditionSearchAll(client *elastic.Client, indices, filterPath, timeout string, tracTotalHits bool, query map[string]interface{}) (response *elastic.SearchResult, err error) {
	if utils.Contains(scrollSearchIndices, indices) {
		// scroll search
		jsonString, _ := json.Marshal(query)
		queryObj := elastic.NewRawStringQuery(string(jsonString))
		svc := client.Scroll(indices).
			Scroll(scrollSearchTimeout).
			TrackTotalHits(true).
			Size(10000).
			Query(queryObj)
		if filterPath != "" {
			svc = svc.FilterPath(filterPath)
		}
		if timeout != "" {
			svc = svc.SearchSource(elastic.NewSearchSource().Timeout(timeout))
		}
		response, err = svc.Do(context.TODO())
		if err != nil {
			log.Println("search error.", err)
			return
		}
		// scrollids search...
		var scrollIds []string
		scrollIds = append(scrollIds, response.ScrollId)
		for {
			lastScrollId := scrollIds[len(scrollIds) - 1]
			if lastScrollId == "" || lastScrollId == "nil" {
				break
			}
			tmpResp, e := client.Scroll(indices).
				ScrollId(lastScrollId).
				Do(context.TODO())
			if e != nil {
				fmt.Println("scroll search 중 에러 발생.", e)
				log.Println("scroll search 중 에러 발생.", e)
				break
			}
			if tmpResp.ScrollId != "" {
				scrollIds = append(scrollIds, tmpResp.ScrollId)
			}

			for _, hit := range tmpResp.Hits.Hits {
				// hits 안에 hit 추가.
				response.Hits.Hits = append(response.Hits.Hits, hit)
			}
			if response.Hits.MaxScore != nil && tmpResp.Hits.MaxScore != nil {
				maxScore := math.Max(*response.Hits.MaxScore, *tmpResp.Hits.MaxScore)
				response.Hits.MaxScore = &maxScore
			}
			//if !tracTotalHits {
			//	response.Hits.TotalHits.Value += tmpResp.Hits.TotalHits.Value
			//}
		}
		fmt.Println("scroll search 완료. id 갯수: ", len(scrollIds), ", 문서 갯수: ", len(response.Hits.Hits), ", totalHits 수:", response.Hits.TotalHits.Value)
		log.Println("scroll search 완료. id 갯수: ", len(scrollIds), ", 문서 갯수: ", len(response.Hits.Hits), ", totalHits 수:", response.Hits.TotalHits.Value)

		client.ClearScroll(scrollIds...)

		log.Println("scroll searching..", indices, ", size: ", response.Hits.TotalHits.Value)
	} else {
		// search only
		svc := client.Search().
			Index(indices).
			Source(query).
			TrackTotalHits(tracTotalHits)
		if timeout != "" {
			svc = svc.Timeout(timeout)
		}
		if filterPath != "" {
			svc = svc.FilterPath(filterPath)
		}
		response, err = svc.Do(context.TODO())
	}

	return
}
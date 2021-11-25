package extentions

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/olivere/elastic/v7"
	"io"
	"io/ioutil"
	"log"
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
			zero := `{"took":1,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}`
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

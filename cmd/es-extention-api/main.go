package main

import (
	"bytes"
	"fmt"
	"github.com/danawalab/es-extention-api/src/extentions"
	"github.com/danawalab/es-extention-api/src/utils"
	"github.com/gorilla/mux"
	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	serverAddr   = utils.GetArg("address", "0.0.0.0", os.Args)
	serverPort   = utils.GetArg("port", "9000", os.Args)
	goEnv        = utils.GetArg("go.env", "production", os.Args)
	accessLogger = log.Logger{}
	debugLogger  = log.Logger{}
)


func main() {
	// 로그 설정
	// 기본 7일 삭제
	rl, _ := rotatelogs.New("./logs/application-%Y%m%d.log")
	log.SetOutput(rl)

	accessLogger = *log.Default()
	arl, _ := rotatelogs.New("./logs/access-%Y%m%d.log")
	accessLogger.SetOutput(arl)

	debugLogger = *log.Default()
	drl, _ := rotatelogs.New("./logs/debug-%Y%m%d.log")
	debugLogger.SetOutput(drl)

	// 확장 초기화
	extentions.Initialize()

	// 라우팅
	router := mux.NewRouter()

	router.HandleFunc("/{indices:[a-zA-Z0-9-_.,*]+}/_join", extentions.Join)
	router.HandleFunc("/{uri:.*}", extentions.Default)

	// listen..
	listen := fmt.Sprintf("%s:%s", serverAddr, serverPort)
	log.Println("API server Listening at : " + listen)
	fmt.Println("API server Listening at : " + listen)
	//log.Fatal(http.ListenAndServe(listen, router))
	log.Fatal(http.ListenAndServe(listen, logRequest(router)))
}



func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(strings.ToLower(r.URL.String()), "debug=true") {
			bodyBytes,err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Fatal(err)
			}
			r2 := r.Clone(r.Context())
			// clone body
			r.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
			r2.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
			tmpLogLine := fmt.Sprintf("%s %s %s\n%s", r.RemoteAddr, r.Method, r.URL, string(bodyBytes))
			fmt.Println(tmpLogLine)
			debugLogger.Println(tmpLogLine)
		} else {
			accessLogger.Println(fmt.Sprintf("%s %s %s", r.RemoteAddr, r.Method, r.URL))
		}
		handler.ServeHTTP(w, r)
	})
}
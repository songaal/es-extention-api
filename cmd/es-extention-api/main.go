package main

import (
	"fmt"
	"github.com/danawalab/es-extention-api/src/extentions"
	"github.com/danawalab/es-extention-api/src/utils"
	"github.com/gorilla/mux"
	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"log"
	"net/http"
	"os"
)

var (
	serverAddr   = utils.GetArg("address", "0.0.0.0", os.Args)
	serverPort   = utils.GetArg("port", "9000", os.Args)
	goEnv        = utils.GetArg("go.env", "production", os.Args)
)

func main() {
	// 로그 설정
	if goEnv == "production" {
		// 기본 7일 삭제
		rl, _ := rotatelogs.New("./application-%Y%m%d.log")
		log.SetOutput(rl)
	}

	// 확장 초기화
	extentions.Initialize()

	// 라우팅
	router := mux.NewRouter()
	router.HandleFunc("/{indices:[a-zA-Z0-9-_.,*]+}/_left", extentions.Left)
	router.HandleFunc("/{uri:.*}", extentions.Default)

	// listen..
	listen := fmt.Sprintf("%s:%s", serverAddr, serverPort)
	log.Println("API server Listening at : " + listen)
	log.Fatal(http.ListenAndServe(listen, router))
}
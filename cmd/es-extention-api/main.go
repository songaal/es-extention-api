package main

import (
	"github.com/danawalab/es-extention-api/src/extentions"
	"github.com/danawalab/es-extention-api/src/utils"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

var (
	ServerPort   = utils.GetArg("port", "9000", os.Args)
	EsTarget, _  = url.Parse(utils.GetArg("es.url", "http://elasticsearch:9200", os.Args))
	EsUser       = utils.GetArg("es.user", "", os.Args)
	EsPassword   = utils.GetArg("es.password", "", os.Args)
)


func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	proxy := httputil.NewSingleHostReverseProxy(EsTarget)

	proxy.ServeHTTP(res, req)
}


func main() {
	http.HandleFunc(".*/_left", extentions.LeftJoin)
	http.HandleFunc("/", handleRequestAndRedirect)
	log.Println("API server Listening at : 0.0.0.0:"+ServerPort)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+ServerPort, nil))
}
package extentions

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var defaultRoundRobinSequence = 0
func Default(res http.ResponseWriter, req *http.Request) {
	if GoEnv != "production" {
		log.Println("proxy : ", req.RequestURI)
	}
	esTarget, _ := url.Parse(EsTargetList[defaultRoundRobinSequence])
	if defaultRoundRobinSequence < len(EsTargetList) - 1 {
		defaultRoundRobinSequence += 1
	} else {
		defaultRoundRobinSequence = 0
	}
	proxy := httputil.NewSingleHostReverseProxy(esTarget)
	proxy.ServeHTTP(res, req)
}



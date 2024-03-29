package extentions

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

var defaultRoundRobinSequence = 0
func Default(res http.ResponseWriter, req *http.Request) {
	target := strings.Split(esUrl, ",")
	esTarget, _ := url.Parse(target[defaultRoundRobinSequence])
	if defaultRoundRobinSequence < len(target) - 1 {
		defaultRoundRobinSequence += 1
	} else {
		defaultRoundRobinSequence = 0
	}
	proxy := httputil.NewSingleHostReverseProxy(esTarget)
	proxy.ServeHTTP(res, req)
}



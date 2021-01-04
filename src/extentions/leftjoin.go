package extentions

import (
	"fmt"
	"github.com/danawalab/es-extention-api/src/utils"
	"net/http"
	"net/url"
	"os"
)

var (
	EsTarget, _  = url.Parse(utils.GetArg("es.url", "http://elasticsearch:9200", os.Args))
	EsUser       = utils.GetArg("es.user", "", os.Args)
	EsPassword   = utils.GetArg("es.password", "", os.Args)
)

func LeftJoin(res http.ResponseWriter, req *http.Request) {


	fmt.Println(">>>>>>>>>>")


}
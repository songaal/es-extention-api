package extentions

import (
	"github.com/danawalab/es-extention-api/src/utils"
	"github.com/olivere/elastic/v7"
	"log"
	"os"
	"strings"
	"time"
)

var (
	EsTargetList = strings.Split(utils.GetArg("es.urls", "http://elasticsearch:9200", os.Args), ",")
	EsUser       = utils.GetArg("es.user", "", os.Args)
	EsPassword   = utils.GetArg("es.password", "", os.Args)
	EsClient     = elastic.Client{}
)

func Initialize() {
	log.Println("init.")
	tmpEsClient, err := elastic.NewClient(
		elastic.SetBasicAuth(EsUser, EsPassword),
		elastic.SetURL(EsTargetList...),
		elastic.SetGzip(false),
		elastic.SetSniff(false),
		elastic.SetHealthcheckInterval(10*time.Second),
		elastic.SetMaxRetries(3),
		elastic.SetTraceLog(log.New(os.Stdout, "TRACE ", log.Ltime|log.Lshortfile)))
	if err != nil {
		log.Println("ES Connection ERROR", err)
	} else {
		EsClient = *tmpEsClient
	}
}

package extentions

import (
	"fmt"
	"github.com/danawalab/es-extention-api/src/utils"
	"github.com/olivere/elastic/v7"
	"log"
	"os"
	"strings"
	"time"
)

var (
	GoEnv          = utils.GetArg("go.env", "development", os.Args)
	EsTargetList = strings.Split(utils.GetArg("es.urls", "http://elasticsearch:9200", os.Args), ",")
	EsUser       = utils.GetArg("es.user", "", os.Args)
	EsPassword   = utils.GetArg("es.password", "", os.Args)
	EsClient     = elastic.Client{}
)

func Initialize() {
	log.Println("init.")
	tmpEsClient, err := elastic.NewClient(
		elastic.SetURL(EsTargetList...),
		elastic.SetBasicAuth(EsUser, EsPassword),
		elastic.SetHealthcheckInterval(10*time.Second),
		elastic.SetMaxRetries(3),
		elastic.SetGzip(true),
		elastic.SetSniff(false),
		elastic.SetTraceLog(log.New(os.Stdout, "", log.Ltime|log.Lshortfile)),
		)

	if err != nil {
		log.Println("ES Connection ERROR", err)
		panic("ES Connection ERROR" + fmt.Sprintln(err))
	} else {
		EsClient = *tmpEsClient
	}
}

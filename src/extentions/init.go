package extentions

import (
	"fmt"
	"github.com/danawalab/es-extention-api/src/utils"
	"github.com/olivere/elastic/v7"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	GoEnv         = utils.GetArg("go.env", "development", os.Args)
	esUrl         = utils.GetArg("es.urls", "http://elasticsearch:9200", os.Args)
	esUser        = utils.GetArg("es.user", "", os.Args)
	esPass        = utils.GetArg("es.password", "", os.Args)
	DefaultClient = elastic.Client{}
	termsMaxCountStr = utils.GetArg("termsMaxCount", "9999999", os.Args)
	TermsMaxCount = 9999999
)

func Initialize() {
	log.Println("init.")
	log.Println("es list:", esUrl)
	log.Println("es user:", esUser)
	log.Println("es pass:", esPass)
	if tmpEsClient, err := GetClient(esUrl, esUser, esPass); err != nil {
		log.Println("ES Connection ERROR", err)
		panic("ES Connection ERROR" + fmt.Sprintln(err))
	} else {
		DefaultClient = tmpEsClient
	}
	TermsMaxCount, _ = strconv.Atoi(termsMaxCountStr)
}

func GetClient(host, user, password string) (client elastic.Client, err error) {
	if tmpEsClient, tmpErr := elastic.NewClient(
		elastic.SetURL(strings.Split(host, ",")...),
		elastic.SetBasicAuth(user, password),
		elastic.SetHealthcheckInterval(10*time.Second),
		elastic.SetMaxRetries(3),
		elastic.SetGzip(true),
		elastic.SetSniff(false),
		//elastic.SetTraceLog(log.New(os.Stdout, "", log.Ltime|log.Lshortfile)),
	); tmpErr != nil {
		err = tmpErr
	} else {
		client = *tmpEsClient
	}
	return
}

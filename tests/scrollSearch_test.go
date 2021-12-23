package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"strings"
	"testing"
	"time"
)

func TestScrollSearch(t *testing.T) {

	client := elastic.Client{}
	if tmpEsClient, tmpErr := elastic.NewClient(
		elastic.SetURL(strings.Split("http://es2.danawa.io:9200", ",")...),
		elastic.SetBasicAuth("elastic", "ekskdhk1!"),
		elastic.SetHealthcheckInterval(10*time.Second),
		elastic.SetMaxRetries(3),
		elastic.SetGzip(true),
		elastic.SetSniff(false),
		//elastic.SetTraceLog(log.New(os.Stdout, "", log.Ltime|log.Lshortfile)),
	); tmpErr != nil {
		fmt.Println(tmpErr)
	} else {
		client = *tmpEsClient
	}



	q := `
{
	"match_all": {}
}
`

	childQuery := make(map[string]interface{})
	_ = json.Unmarshal([]byte(q), &childQuery)


	//query := make(map[string]interface{})

	//esClient.Search().
	//	Index("childIndices").
	//	FilterPath("hits.hits").
	//	Timeout("120s").
	//	Source(childQuery).
	//	Do(context.TODO())

	jsonString, err := json.Marshal(childQuery)
	fmt.Println(err)
	query := elastic.NewRawStringQuery(string(jsonString))
	fmt.Println(query)

	do, err := client.Scroll("tcmpny_link-a").
		FilterPath("hits.hits").
		Query(query).
		Scroll("1m").
		TrackTotalHits(true).
		Do(context.TODO())
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("ok", do)
	}



}
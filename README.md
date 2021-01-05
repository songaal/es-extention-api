# 엘라스틱서치 확장 API 

## 소개

엘라스틱서치의 Left Join 기능을 제공합니다.

es-extention-api를 통해 엘라스틱서치 연결을 하면 /_left 엔드포인트를 사용할 수 있게 됩니다.    

_search 유사하게 조인 QueryDSL를 작성하게 되면 innerHits에 _child를 포함한 결과를 받을 수 있습니다.

```
GET /parent-index/_left
{
  "query": {
    "match": {
      "productName": "상품 조회"
    }
  },
  "from": 0,
  "size": 10000,
  "join": {
    "index": "child-index",
    "parent": "bundleKey",
    "child": "bundleKey",
    "query": {
       "match_all": {}
     }
  }
}
```

OUTPUT
```
{
  "took": 210,
  "hits": {
    "total": {
      "value": 10000,
      "relation": "gte"
    },
    "max_score": 1,
    "hits": [
      {
        "_score": 1,
        "_index": "parent-index",
        "_type": "_doc",
        "_id": "3664488",
        "_seq_no": null,
        "_primary_term": null,
        "_source": {
          "priceType": "",
          "discontinued": "N",
          ...(중략)
        },
        "inner_hits": {
          "_child": {
            "hits": {
              "total": {
                "value": 1,
                "relation": "eq"
              },
              "max_score": 1,
              "hits": [
                {
                  "_score": 1,
                  "_index": "child-index",
                  "_type": "_doc",
                  "_id": "3664488",
                  "_seq_no": null,
                  "_primary_term": null,
                  "_source": {
                    "popularityScore": "0",
                    "shareCate3": "0"
                    ...(중략)
                  }
                }
              ]
            }
          }
        }
      },
      ... (이하 생략)
```

### JOIN 필드 설명

| 필드명 | 설명 |
| --- | --- |
|index | child 인덱스명 |
|parent | Parent 필드명 |
|child | Child 필드명 |
|query | Child 검색 쿼리 |


## es-extention-api 실행방법

### 파라미터
|옵션|기본값|설명|
|---|---|---|
|address|0.0.0.0|Listen Address|
|port|9000|Listen Port|
|es.urls|http://elasticsearch:9200|엘라스틱서치 URL ,(콤마) 구분하여 입력|
|es.user|""|엘라스틱서치 사용자명|
|es.password|""|엘라스틱서치 비밀번호|


### 실행 명령어
아래 명령어로 서버를 실행하 후 curl localhost:9000/_cat/nodes 요청시 정상적으로 ES 결과를 받는지 확인합니다. 
 ```
./application port=9000 es.urls=http://elasticsearch:9200 es.user=elastic es.password=z1p87tXaR8gggSqPh8x 
```




 
 
 
 
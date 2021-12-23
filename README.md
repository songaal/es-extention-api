# 엘라스틱서치 확장 API 

## 소개

엘라스틱서치에서 조인기능을 제공합니다.

엘라스틱서치 앞에 ex-extention-api를 사용할 경우, /<index>/_join 엔드포인트를 사용할 수 있습니다.

엘라스틱 문서 검색과 동일하게 QueryDSL 통해 검색하면되며, 조인을 위해선 join 과 type 필드를 작성하면됩니다. 

조인의 추가된 결과는 innerHits 하위에 _child 영역에 포함되게 됩니다.

---
새로운 inner 조인기능이 추가되었습니다.

inner 조인은 parent, child 조건에 동일한 결과의 집합합니다.  

left 조인과 inner 조인은 type 필드로 구분됩니다. 

조인 구조
```text
{
    query: {
        ... (생략)
    }
    type: "inner" or "left" 
    join: { 
      index: "child-index",
      host: "http://dev-elasticsearch:9200",
      username: "elastic",
      password: "secret",
      parent: "parent-key",
      child:  "child-key",
      ... (생략)
     }
    ... (생략)
}
```
* inner 조인은 여러개의 join 을 미지원.

---

자세한 내용은 다나와 기술블로그를 확인해주세요.
https://danawalab.github.io/elastic/2021/01/06/elasticsearch-left-join-proxy.html

```
GET /parent-index/_join
{
  "query": {
    "match": {
      "productName": "상품 조회"
    }
  },
  "from": 0,
  "size": 10000,
  "type": "left",
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
|type | "left", "inner"를 지원합니다. |
|index | child 인덱스명 |
|parent | Parent 필드명 |
|child | Child 필드명 |
|query | Child 검색 쿼리 |
|host | Child 검색할 엘라스틱 호스트정보 (ex: http://elasticsearch:9200,...) |
|username | Child 검색할 엘라스틱 사용자명 |
|password | Child 검색할 엘라스틱 비밀번호 |
|scroll.indices | Child 조회시 scroll 방식 조회할 인덱스를 콤마구분하여 입력 (ex: product, goods) |
|scroll.keepalive | scroll keepalive (기본값: 5m) |

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




 
 
 
 
# Datastore demo
## MongoDB
database: `vc`

collection: `datastore`

indexs: `document_id: 1`

### Query
`use vc`

`db.datastore.find({"meta.document_id": "<uuid>"}).pretty()`

## Demo environment
### Run
`make start`

### Stop
`make stop`

### Restart
`make restart`

### Clean
cleans docker volumes etc..

`make clean`

## Produce mock data to datastore
### Upload one document
This call will send a complete mock version of a document to the datastore, the datastore will then save it in mongoDB.

```curl --request POST \
  --url http://172.16.80.7:8080/api/v1/mock/next \
  --header 'Content-Type: application/json' \
  --data '{
	"document_type": "EHIC",
	"authentic_source": "SUNET"
    }'
```

### Upload many documents
This call will send `n` complete mocks. Return a list of `document_id`.

```curl --request POST \
  --url 'http://172.16.80.7:8080/api/v1/mock/bulk?n=1000' \
  --header 'Content-Type: application/json' \
  --data '{
	"document_type": "PDA1",
	"authentic_source": "SUNET"
    }'
```

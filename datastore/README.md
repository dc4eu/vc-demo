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

## mockas
Produce mock data to datastore

### Upload one document
This call will send a complete mock version of a document to the datastore, the datastore will then save it in mongoDB. Return complete payload for inspection.

```bash
curl --request POST \
  --url http://172.16.80.7:8080/api/v1/mock/next \
  --header 'Content-Type: application/json' \
  --data '{
	"document_type": "EHIC",
	"authentic_source": "SUNET"
    }'
```

### Upload many documents
This call will send `n` complete mocks. Return a list of `document_id`.

```bash
curl --request POST \
  --url 'http://172.16.80.7:8080/api/v1/mock/bulk?n=1000' \
  --header 'Content-Type: application/json' \
  --data '{
	"document_type": "PDA1",
	"authentic_source": "SUNET"
    }'
```

## APIGW
### /api/v1/id/mapping
Map known attributes to document in datastore. Attributes needs to be change to match previous reply from `mockas`

```bash
curl --request POST \
  --url http://172.16.80.2:8080/api/v1/id/mapping \
  --header 'Content-Type: application/json' \
  --data '{
	"authentic_source": "SUNET",
	"identity": {
		"version": "1",
			"family_name": "Schimmel",
			"given_name": "Tremayne",
			"birth_date": "1913-12-04 01:35:53.232126111 +0000 UTC",
			"uid": "6e006f0d-dea0-477e-af9f-3856d910327c",
			"family_name_at_birth": "Quitzon",
			"given_name_at_birth": "Dayana",
			"birth_place": "Jacksonville",
			"gender": "M"
	}
}'
```
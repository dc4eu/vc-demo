docker-build-ui:
	$(info Docker Building web1 with tag: $(VERSION))
	docker build --tag docker.sunet.se/dc4eu/ui:demo-v3 .

docker-push-ui:
	$(info Pushing docker images)
	docker push docker.sunet.se/dc4eu/ui:demo-v3

start:
	$(info Run!)
	docker-compose -f docker-compose.yaml up -d --remove-orphans

stop:
	$(info stopping VC)
	docker-compose -f docker-compose.yaml rm -s -f

restart: stop start


### Reminder/frequently used commands: ###

# Build all containers:
# make docker-build-web1

# Start all containers:
# make start

# Stop all containers:
# make stop

# Restart all containers:
# make restart

# Logs:
# docker logs vc_web1
# docker logs vc_apigw

# Inspect container:
# docker inspect vc_web1
# docker inspect vc_apigw

# Browse admin web
# http://http://172.16.50.12:8080/

# Access running container
# docker exec -ti vc_web1 bash

# Curl example:

#curl --request POST \
#  --url http://172.16.50.13:8080/api/v1/mock/next \
#  --header 'Content-Type: application/json' \
#  --data '{
#	"document_type": "EHIC",
#	"authentic_source": "SUNET"
#    }' | jq .




# curl --request POST   --url http://172.16.50.13:8080/api/v1/mock/next   --header 'Content-Type: application/json'   --header 'User-Agent: insomnia/2023.5.8'   --data '{
#        "document_type": "PDA1",
#        "authentic_source": "SUNET"
#
#}' | jq .
#
# Note: "| jq ." pipes out formatted json 
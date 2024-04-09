docker-build-web1:
	$(info Docker Building web1 with tag: $(VERSION))
	docker build --tag docker.sunet.se/dc4eu/web1:latest .

docker-push-web1:
	$(info Pushing docker images)
	docker push docker.sunet.se/dc4eu/web1:latest

start:
	$(info Run!)
	docker-compose -f docker-compose.yaml up -d --remove-orphans

stop:
	$(info stopping VC)
	docker-compose -f docker-compose.yaml rm -s -f

restart: stop start
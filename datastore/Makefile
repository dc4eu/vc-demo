start:
	$(info Run!)
	docker-compose -f docker-compose.yaml up -d --remove-orphans

stop:
	$(info stopping VC)
	docker-compose -f docker-compose.yaml rm -s -f

restart: stop start

clean: stop
	docker volume rm datastore_redis_data
	docker volume rm datastore_mongo_data
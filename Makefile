MIGRATE_DB=postgres://notifier_user:postgres@localhost:5435/notifier_db?sslmode=disable
KAFKA_CONTAINER=delayed-notifier-kafka
TOPIC_NAME=notify-topic
BROKER=localhost:9092
PARTITIONS=3
REPLICATION=1

create-topic:
	docker exec $(KAFKA_CONTAINER) kafka-topics.sh \
		--create \
		--if-not-exists \
		--topic $(TOPIC_NAME) \
		--bootstrap-server $(BROKER) \
		--partitions $(PARTITIONS) \
		--replication-factor $(REPLICATION)

migrate-up:
	goose -dir migrations postgres "$(MIGRATE_DB)" up

migrate-down:
	goose -dir migrations postgres "$(MIGRATE_DB)" down

lint:
	golangci-lint run --config=.golangci.yml ./...

fmt:
	gci write -s standard -s default -s "prefix(delayed-notifier)" .

generate-mocks:
	mockery --name=NotifyDBRepository --dir=internal/service --output=internal/repository/postgres/mocks --with-expecter
	mockery --name=NotifyCacheRepository --dir=internal/service --output=internal/repository/redis/mocks --with-expecter
	mockery --name=NotifyProducer --dir=internal/service --output=internal/repository/producer/mocks --with-expecter
	mockery --name=Notifier --dir=internal/service --output=internal/repository/email/mocks --with-expecter

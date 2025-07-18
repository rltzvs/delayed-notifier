MIGRATE_DB=postgres://notifier_user:postgres@localhost:5435/notifier_db?sslmode=disable

migrate-up:
	goose -dir migrations postgres "$(MIGRATE_DB)" up

migrate-down:
	goose -dir migrations postgres "$(MIGRATE_DB)" down

lint:
	golangci-lint run ./...

fmt:
	gci write -s standard -s default -s "prefix(delayed-notifier)" .

generate-mocks:
	mockery --name=NotifyDBRepository --dir=internal/service --output=internal/repository/postgres/mocks --with-expecter
	mockery --name=NotifyCacheRepository --dir=internal/service --output=internal/repository/redis/mocks --with-expecter
	mockery --name=NotifyProducer --dir=internal/service --output=internal/repository/producer/mocks --with-expecter

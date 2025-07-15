MIGRATE_DB=postgres://notifier_user:postgres@localhost:5435/notifier_db?sslmode=disable

migrate-up:
	goose -dir migrations postgres "$(MIGRATE_DB)" up

migrate-down:
	goose -dir migrations postgres "$(MIGRATE_DB)" down
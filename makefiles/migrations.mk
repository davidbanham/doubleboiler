.PHONY: migrate
migrate:
	migrate -database $(DEV_DB_URI) -path ./migrations up

.PHONY: migrate_redo
migrate_redo:
	migrate -database $(DEV_DB_URI) -path ./migrations down 1
	migrate -database $(DEV_DB_URI) -path ./migrations up 1

.PHONY: migrate_undo
migrate_undo:
	migrate -database $(DEV_DB_URI) -path ./migrations migrate -1

.PHONY: test_migrate
test_migrate:
	migrate -database $(TESTING_DB_URI) -path ./migrations up

.PHONY: migration
migname ?= $(shell bash -c 'read -p "Name: " name; echo $$name')
migration:
	migrate -database $(DEV_DB_URI) create -dir ./migrations -ext sql $(migname)

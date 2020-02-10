.PHONY: migrate
migrate:
	migrate -url $(DEV_DB_URI) -path ./migrations up

.PHONY: migrate_redo
migrate_redo:
	migrate -url $(DEV_DB_URI) -path ./migrations redo

.PHONY: migrate_undo
migrate_undo:
	migrate -url $(DEV_DB_URI) -path ./migrations migrate -1

.PHONY: test_migrate
test_migrate:
	migrate -url $(TESTING_DB_URI) -path ./migrations up

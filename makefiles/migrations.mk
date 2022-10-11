.PHONY: migrate
migrate:
	DB_URI=$(DEV_DB_URI) go run ./migrations/up

.PHONY: prodmigrate
prodmigrate:
	DB_URI=$(PROD_DB_URI) go run ./migrations/up

.PHONY: migrate_redo
migrate_redo:
	DB_URI=$(DEV_DB_URI) go run ./migrations/redo

.PHONY: migrate_undo
migrate_undo:
	DB_URI=$(DEV_DB_URI) go run ./migrations/undo

.PHONY: test_migrate
test_migrate:
	DB_URI=$(TESTING_DB_URI) go run ./migrations/up

.PHONY: migration
migname ?= $(shell bash -c 'read -p "Name: " name; echo $$name')
migration:
	go run ./migrations/migration $(migname)

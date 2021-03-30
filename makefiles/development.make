.PHONY: live_reload
live_reload: export TEST_MOCKS_ON := false
live_reload: export DB_URI := $(DEV_DB_URI)
live_reload:
	DB_URI=$(DEV_DB_URI) ENVIRONMENT=development ./local_dev/CompileDaemon -command="./doubleboiler" -include="*html" -include="*.js"

.PHONY: rummage
rummage:
	psql --user doubleboiler

.PHONY: change
change:
	cat changelog/template.go | sed 's/{{now}}/$(now)/' > "changelog/$(now_no_colons).go"
	vim "changelog/$(now_no_colons).go"


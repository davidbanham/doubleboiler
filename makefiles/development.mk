.PHONY: live_reload
live_reload: export TEST_MOCKS_ON := false
live_reload: export DB_URI := $(DEV_DB_URI)
live_reload: assets/css/main.css
live_reload: logs
	DB_URI=$(DEV_DB_URI) ENVIRONMENT=development ./local_dev/CompileDaemon -command="./doubleboiler" -include="*html" -include="*.js" 2>&1 | tee logs

.PHONY: rummage
rummage:
	psql --user doubleboiler

.PHONY: change
change:
	cat changelog/template.go | sed 's/{{now}}/$(now)/' > "changelog/$(now_no_colons).go"
	vim "changelog/$(now_no_colons).go"

logs:
	mkfifo logs

.PHONY: devlogger
devlogger: logs
	go run ./devlogs/main.go < logs

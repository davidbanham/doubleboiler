DEV_DB_URI = postgres://doubleboiler:wut@localhost:5432/doubleboiler?sslmode=disable
TESTING_DB_URI = postgres://doubleboiler:wut@localhost:5432/doubleboiler_test?sslmode=disable

include ./makefiles/development.mk
include ./makefiles/go.mk
include ./makefiles/migrations.mk
include ./makefiles/standup.mk
include ./makefiles/tailwind.mk

name = app
brand = doubleboiler
prefix = $(brand)-
project = speedtest-186210
keybase_team = notbad.software
forbidden_untracked_extensions = '\.go|\.js'
now = $(shell date -u --rfc-3339 seconds | sed 's/ /T/')
now_no_colons = $(shell echo $(now) | sed 's/:/_/g')

titleCaseBrand := $(shell awk 'BEGIN{print toupper(substr("$(brand)",1,1)) substr("$(brand)", 2, length("$(brand)"))}')
upperCaseBrand := $(shell awk 'BEGIN{print toupper("$(brand)")}')

.db_init:
	psql postgres postgres < ./config/database/initialize_db.sql
	touch .db_init

stage = dummy
pull_policy = Always
uniq:=$(shell uuidgen)
tag = $(shell whoami)-dev-$(uniq)

region = australia-southeast1
hostnames = $(stage).$(domain),*.$(stage).$(domain)
servicename = $(brand)-$(name)-$(stage)

.PHONY: production
production: areyousure test demand_clean build stage_production ca-certificates.crt docker_image_build registry_push cloud_run_deploy

.PHONY: staging
staging: test demand_clean build stage_staging docker_image_build registry_push cloud_run_deploy

.PHONY: development
development: test build stage_development docker_image_build registry_push cloud_run_deploy

.PHONY: demand_clean
demand_clean:
	@# Check there are no forbidden extensions not tracked by git
	echo git ls-files --others --exclude-standard | grep -E $(forbidden_untracked_extensions) | xargs -n 1 test -z
	@# Check that there are no local modifications
	echo git diff-index --quiet HEAD -- && test -z "$(git ls-files --exclude-standard --others)"
	@# Check that we are up to date with remotes
	echo ./kube_maker/makefiles/gitup.sh
	$(eval pull_policy=IfNotPresent)
	$(eval tag=$(shell git rev-parse HEAD))

.PHONY: stage_production
stage_production:
	$(eval hostnames=$(domain),*.$(domain))
	$(eval stage=production)

.PHONY: stage_staging
stage_staging:
	$(eval stage=staging)

.PHONY: stage_development
stage_development:
	$(eval stage=development)

ca-certificates.crt: /etc/ssl/certs/ca-certificates.crt
	cp /etc/ssl/certs/ca-certificates.crt .

.PHONY: docker_image_build
docker_image_build:
	docker build --tag $(name) .
	docker tag $(name):latest gcr.io/$(project)/$(prefix)$(name):$(tag)

.PHONY: registry_push
registry_push:
	gcloud docker -- push gcr.io/$(project)/$(prefix)$(name):$(tag)

.PHONY: cloud_run_deploy
cloud_run_deploy: service_route
	echo gcloud run deploy $(servicename) --image gcr.io/$(project)/$(prefix)$(name):$(tag) --platform managed --region $(region) \
		--set-env-vars=$(shell keybase decrypt < $(stage).env.encrypted | tr '\n' ',') \
	  --add-cloudsql-instances $(project):$(region):$(cloudsql_instance_name)

.PHONY: service_route
service_route:
	-gcloud beta compute network-endpoint-groups create $(servicename)-serverless-neg \
    --region=$(region) \
    --network-endpoint-type=SERVERLESS  \
    --cloud-run-service=$(servicename)
	-gcloud compute backend-services create $(servicename)-backend-service \
    --global
	-gcloud beta compute backend-services add-backend $(servicename)-backend-service \
    --global \
    --network-endpoint-group=$(servicename)-serverless-neg \
    --network-endpoint-group-region=$(region)
	-gcloud compute url-maps add-path-matcher $(parentname)-url-map \
   --default-service $(servicename)-backend-service \
   --path-matcher-name $(servicename)-path-matcher \
	 --new-hosts $(hostnames)

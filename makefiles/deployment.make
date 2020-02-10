.PHONY: deploy
deploy: test build
  gcloud builds submit [CONFIG_FILE_PATH] [SOURCE_DIRECTORY]

.PHONY: docker_image_build
docker_image_build:
	docker build --tag $(name) .

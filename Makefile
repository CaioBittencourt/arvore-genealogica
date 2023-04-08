APP_NAME?=$(shell pwd | xargs basename)
APP_DIR=/go/src/github.com/CaioBittencourt/${APP_NAME}
INTERACTIVE:=$(shell [ -t 0 ] && echo i || echo d)
PWD=$(shell pwd)

generate-mongodb-keyfile:
ifeq ("$(wildcard ./keyfile)","")
	@sudo apt-get install libssl-dev
	openssl rand -base64 756 > keyfile
	chmod 600 keyfile
	sudo chown 999 keyfile
	sudo chgrp 999 keyfile
endif

mod:
	@go mod vendor

setup: mod generate-mongodb-keyfile

test: setup
	@go clean --testcache

	@docker compose up --remove-orphans -d mongo
	TARGET=development docker compose build server

	@docker compose run \
		-t${INTERACTIVE} --rm \
		-e MONGO_URI=mongodb://mongo:27017 -e MONGO_DATABASE=familyTreeTest \
		-v ${PWD}:${APP_DIR}:delegated \
		-w ${APP_DIR} \
		--name ${APP_NAME}-test \
		server \
		go test ./... -race

server-prod: setup
	TARGET=release docker-compose build server
	@docker-compose up --no-attach mongo

server-dev: setup
	@docker compose up --remove-orphans -d mongo
	TARGET=development docker compose build server

	@docker compose run \
		-t${INTERACTIVE} --rm \
		-v ${PWD}:${APP_DIR}:delegated \
		-w ${APP_DIR} \
		-p 8080:80 \
		--name ${APP_NAME}-dev \
		server \
		modd -f ./modd.conf

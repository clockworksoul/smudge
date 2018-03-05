.DEFAULT_GOAL := help

############################
## Project Info
############################
PROJECT = smudge

############################
## Docker Registry Info
############################
REGISTRY_URL = clockworksoul
IMAGE_NAME = $(REGISTRY_URL)/$(PROJECT)
IMAGE_TAG = latest

help:
	# Commands:
	# make help           - Show this message
	#
	# Dev commands:
	# make clean          - Remove generated files
	# make install        - Install requirements (none atm)
	# make test           - Run Go tests
	# make build          - Build go binary
	#
	# Docker commands:
	# make image          - Build Docker image with current version tag
	# make run            - Run Docker image with current version tag
	#
	# Deployment commands:
	# make push           - Push current version tag to registry

clean:
	if [ -d "bin" ]; then rm -R bin; fi

install:
	echo

test: 
	@docker build --target test -t foo_smudge_foo .
	@docker rmi foo_smudge_foo

build: clean
	mkdir -p bin
	@go build -a -installsuffix cgo -o bin/smudge github.com/clockworksoul/smudge/smudge

image:
	@docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

push:
	@docker push $(IMAGE_NAME):$(IMAGE_TAG)

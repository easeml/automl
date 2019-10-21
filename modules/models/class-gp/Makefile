# Makefile for the Go implementation of the ease.ml schema subproject.


# Summary and context path of this makefile.
SUMMARY := This Makefile is intended for managing model images.
CONTEXT_PATH := schema/go


# Paths to the parent directory of this makefile and the repo root directory.
MY_DIR_PATH := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR_PATH := $(realpath $(MY_DIR_PATH)../../..)


# Importable config variables.
DIST_PATH ?= dist
EASEML_ADDRESS ?= localhost:8080


# Other config variables.
PROJECT_NAME := mdl-class-gp
MODULE_TYPE := model
DOCKER := $(shell which docker)
EASEML := $(shell which easeml)
GO := $(shell which go)
GOLINT := $(shell which golint)
GOFMT := $(shell which gofmt)
VERSION := $(shell cat $(ROOT_DIR_PATH)/VERSION)
RELEASE_TAR_NAME := $(PROJECT_NAME)-$(VERSION).tgz


# The show-help target which prints all targets in this file that have a doc help comment.
include $(ROOT_DIR_PATH)/dev/makefiles/show-help.mk
include $(ROOT_DIR_PATH)/dev/makefiles/show-prompt.mk


.PHONY: clean
## Clean all the files resulting from building and testing.
clean:
	$(call show-prompt,Cleaning the build files)
	$(GO) clean
	-rm -rf $(DIST_PATH)/$(RELEASE_TAR_NAME)
	-rm -rf dist


.PHONY: build
## Build the docker image for this module.
build:
	$(call show-prompt,Building the Docker image)
	$(DOCKER) build -t $(PROJECT_NAME) .


.PHONY: push
## Push the module to the ease.ml service specified by EASEML_ADDRESS.
push: build
	$(call show-prompt,Pushing the module to the ease.ml service)
	@echo Target service: $(EASEML_ADDRESS)
	$(EASEML) create module --type $(MODULE_TYPE) --source upload --source-address $(PROJECT_NAME) --id $(PROJECT_NAME) --name $(PROJECT_NAME)


.PHONY: release
## Release the module to Docker hub.
serve:
	# TODO: Add code for performing the actual release.
	$(call show-prompt,Releasing to Docker hub)
	$(GO) run .

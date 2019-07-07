# Makefile for the Go implementation of the ease.ml client subproject.


# Summary and context path of this makefile.
SUMMARY := This Makefile corresponds to the Go implementation of the ease.ml client.
CONTEXT_PATH := client/go
FOOTER := To specify the target directory for make package use the DIST_PATH environment variable \
		  \(default: DIST_PATH=./dist\).


# Paths to the parent directory of this makefile and the repo root directory.
MY_DIR_PATH := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR_PATH := $(realpath $(MY_DIR_PATH)../../..)


# Importable config variables.
ifeq ($(strip $(DIST_PATH)),)
	DIST_PATH := ./dist
endif


# Other config variables.
PROJECT_NAME := easemlclient
GO := $(shell which go)
GOLINT := $(shell which golint)
GOFMT := $(shell which gofmt)
VERSION := $(shell cat $(ROOT_DIR_PATH)/VERSION)
RELEASE_TAR_NAME := $(PROJECT_NAME)-$(VERSION).tgz


# Include common make functions.
include $(ROOT_DIR_PATH)/build/makefiles/show-help.mk
include $(ROOT_DIR_PATH)/build/makefiles/show-prompt.mk


.PHONY: dist
## Initialize the development environment to enable running of build and test.
dist:
	@echo $(DIST_PATH)

.PHONY: init
## Initialize the development environment to enable running of build and test.
init:
	$(call show-prompt,Installing development dependencies)
	$(GO) mod download
	$(GO) get github.com/robertkrimen/godocdown/godocdown


.PHONY: clean
## Clean all the files resulting from building and testing.
clean:
	$(call show-prompt,Cleaning the build files)
	$(GO) clean
	-rm -rf $(DIST_PATH)/$(RELEASE_TAR_NAME)
	-rm -rf ./dist


.PHONY: build
## Compile the project code.
build:
	$(call show-prompt,Compiling project code)
	$(GO) build


.PHONY: package
## Build the project and assemble a deployable package.
package: build
	$(call show-prompt,Building the deployment package)
	mkdir -p $(DIST_PATH)
	tar -zcvf $(DIST_PATH)/$(RELEASE_TAR_NAME) types/*.go *.go go.*


.PHONY: test
## Run all tests.
test:
	$(call show-prompt,Running all tests)
	$(GO) test ./...


.PHONY: lint
## Run the linting checks.
lint:
	$(call show-prompt,Running all linting checks)
	$(GOLINT) ./...


.PHONY: lint-fix
## Automatically fix style errors where possible.
lint-fix:
	$(call show-prompt,Fixing coding style violations)
	$(GOFMT) ./...


.PHONY: version
## Set the version of this package according to version file found in the repo root.
version:
	@echo

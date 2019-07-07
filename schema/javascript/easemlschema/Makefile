# Makefile for the JavaScript implementation of the ease.ml schema subproject.


# Summary and context path of this makefile.
SUMMARY := This Makefile corresponds to the JavaScript implementation of the ease.ml schema.
CONTEXT_PATH := schema/javascript
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
PROJECT_NAME := easemlschema
NPM := $(shell which npm)
NPM_RUN := $(NPM) run
VERSION := $(shell cat $(ROOT_DIR_PATH)/VERSION)


# Include common make functions.
include $(ROOT_DIR_PATH)/build/makefiles/show-help.mk
include $(ROOT_DIR_PATH)/build/makefiles/show-prompt.mk


.PHONY: init
## Initialize the development environment to enable running of build and test.
init:
	$(call show-prompt,Installing development dependencies)
	$(NPM) install


.PHONY: clean
## Clean all the files resulting from building and testing.
clean:
	$(call show-prompt,Cleaning the build files)
	$(NPM_RUN) clean
	-rm -f $(DIST_PATH)/$(PROJECT_NAME)-$(VERSION).tgz


.PHONY: build
## Compile the project code.
build:
	$(call show-prompt,Compiling project code)
	$(NPM_RUN) build


.PHONY: package
## Build the project and assemble a deployable package.
package: build
	$(call show-prompt,Building the deployment package)
	$(NPM) pack
ifneq ($(abspath $(DIST_PATH)), $(CURDIR))
	mkdir -p $(DIST_PATH)
	mv -f $(PROJECT_NAME)-$(VERSION).tgz $(DIST_PATH)/
endif


.PHONY: test
## Run all tests.
test:
	$(call show-prompt,Running all tests)
	$(NPM_RUN) test


.PHONY: lint
## Run the linting checks.
lint:
	$(call show-prompt,Running all linting checks)
	$(NPM_RUN) lint


.PHONY: lint-fix
## Automatically fix style errors where possible.
lint-fix:
	$(call show-prompt,Fixing coding style violations)
	$(NPM_RUN) lint fix


.PHONY: version
## Set the version of this package according to version file found in the repo root.
version:
	$(call show-prompt,Updating package version)
	$(NPM) version $(VERSION) --no-git-tag-version
	@echo New version: $(VERSION)

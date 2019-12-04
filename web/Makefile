# Makefile for the implementation of the ease.ml web UI.


# Summary and context path of this makefile.
SUMMARY := This Makefile corresponds to the implementation of the ease.ml web UI.
CONTEXT_PATH := web


# Paths to the parent directory of this makefile and the repo root directory.
MY_DIR_PATH := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR_PATH := $(realpath $(MY_DIR_PATH)..)


# This variable is ignored here.
ifeq ($(strip $(DIST_PATH)),)
	DIST_PATH := ./dist
endif


# Other config variables.
PROJECT_NAME := web
NPM := $(shell which npm)
NPM_RUN := $(NPM) run
VERSION := $(shell cat $(ROOT_DIR_PATH)/VERSION)


# The show-help target which prints all targets in this file that have a doc help comment.
include $(ROOT_DIR_PATH)/dev/makefiles/show-help.mk
include $(ROOT_DIR_PATH)/dev/makefiles/show-prompt.mk


.PHONY: init
## Initialize the environment to enable running of build and test.
init:
	$(call show-prompt,Installing module dependencies)
	$(NPM) install


.PHONY: init-dev
## Same as init but also instal all development dependencies.
init-dev: init
	$(call show-prompt,Installing development dependencies)
	#


.PHONY: clean
## Clean all the files resulting from building and testing.
clean:
	$(call show-prompt,Cleaning the build files)
	$(NPM_RUN) clean


.PHONY: build
## Build the web UI and place it in the "dist" directory. Does a clean build each time.
build: init
	$(call show-prompt,Compiling project code)
	$(NPM_RUN) build


.PHONY: package
## Build the project and assemble a deployable package.
package: build
	@echo


.PHONY: serve
## Serves the web UI in a standalone server. Triggers build on each source file update. Used for rapid development.
serve:
	$(call show-prompt,Serving the web UI)
	$(NPM_RUN) serve


.PHONY: test
## Run all tests. Currently does nothing as there are no UI tests at the moment.
test:
	@echo


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

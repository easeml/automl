# Makefile for the Go implementation of the ease.ml engine.


# Summary and context path of this makefile.
SUMMARY := This Makefile corresponds to ease.ml engine code.
CONTEXT_PATH := engine/easeml
FOOTER := To specify the target directory for make package use the DIST_PATH environment variable \
		  \(default: DIST_PATH=./dist\). \
		  \\n\\nUse OS to specify the target operating system \(options: linux, darwin, windows\; default: all three\). \
		  \\n\\nUse ARCH to specify target architecture \(default: amd64\).

# Paths to the parent directory of this makefile and the repo root directory.
MY_DIR_PATH := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR_PATH := $(realpath $(MY_DIR_PATH)..)

# Paths to the web UI directory and the dist directory where the built web UI is expected to be.
WEB_UI_PATH := $(ROOT_DIR_PATH)/web
WEB_UI_DIST_PATH := $(WEB_UI_PATH)/dist

# Importable config variables.
ifeq ($(strip $(DIST_PATH)),)
	DIST_PATH := ./dist
endif

# All platforms for which to generate a package. Can be overridden.
ifeq ($(strip $(OS)),)
	OS := linux darwin windows
endif
ifeq ($(strip $(ARCH)),)
	ARCH := amd64
endif


# Other config variables.
PROJECT_NAME := easeml
GO := $(shell which go)
PACKR := $(shell which packr2)
GOLINT := $(shell which golint)
GOFMT := $(shell which gofmt)
VERSION := $(shell cat $(ROOT_DIR_PATH)/VERSION)
RELEASE_TAR_NAME := $(PROJECT_NAME)-$(VERSION).tgz
ifdef GOPATH
INSTALL_PATH?=$(GOPATH)/bin
endif

# The show-help target which prints all targets in this file that have a doc help comment.
include $(ROOT_DIR_PATH)/dev/makefiles/show-help.mk
include $(ROOT_DIR_PATH)/dev/makefiles/show-prompt.mk


# This rule helps us ensure that the web UI is built before we build the engine.
$(WEB_UI_DIST_PATH):
	$(MAKE) -C $(WEB_UI_PATH) build


.PHONY: init
## Initialize the development environment to enable running of build and test.
init:
	$(call show-prompt,Installing module dependencies)
	$(GO) mod download
	echo "Installing PACKR2 binary"
	$(GO) go get -v -u github.com/gobuffalo/packr/v2/packr2


.PHONY: clean
## Clean all the files resulting from building and testing.
clean:
	$(call show-prompt,Cleaning the build files)
	echo $(GO) clean
	$(GO) clean
	-for arch in $(ARCH) ; do \
		for os in $(OS) ; do \
			rm -rf $(DIST_PATH)/$$os-$$arch; \
		done; \
	done
	-rm -rf dist


.PHONY: build
## Compile the project code.
build: $(WEB_UI_DIST_PATH)
	$(call show-prompt,Compiling project code)
	$(PACKR)
	$(GO) build -o $(PROJECT_NAME)
	$(PACKR) clean

.PHONY: install
## Call build then move binary to INSTALL_PATH
install: build
	@if [ -n "$(INSTALL_PATH)" ]; then \
		echo 'Installing to '$(INSTALL_PATH); \
		mv $(PROJECT_NAME) $(PROJECT_NAME)_temp;\
		mv $(PROJECT_NAME)_temp $(INSTALL_PATH)/$(PROJECT_NAME);\
	else\
		echo "Empty INSTALL_PATH and GOPATH. Please set the env variable INSTALL_PATH or GOPATH to a valid directory so that the output binary can be placed there. Preferably this directory is included in your \$PATH environmental variable";\
		exit 1;\
	fi && \
	echo '$@: done'

.PHONY: package
## Build the project and assemble a deployable package.
package: $(WEB_UI_DIST_PATH)
	$(call show-prompt,Building the deployment package)
	mkdir -p $(DIST_PATH)
	for arch in $(ARCH) ; do \
		for os in $(OS) ; do \
			[ $$os = windows ] && name=$(PROJECT_NAME).exe || name=$(PROJECT_NAME); \
			$(PACKR); \
			GOOS=$$os $(GO) build -o $(DIST_PATH)/$$os-$$arch/$$name; \
			$(PACKR) clean; \
		done; \
	done


.PHONY: serve
## Start the easeml service.
serve:
	$(call show-prompt,Starting the easeml service)
	$(GO) run . start


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


.PHONY: docs
## Generate the documentation Markdown files.
docs:
	$(call show-prompt,Generating documentation)
	$(GO) run docs/cli/build.go

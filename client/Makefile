# Makefile for all implementations of the ease.ml client.


# Summary and context path of this makefile.
SUMMARY := This Makefile corresponds the ease.ml client implementations in all languages.
CONTEXT_PATH := client


# Paths to the parent directory of this makefile and the repo root directory.
MY_DIR_PATH := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR_PATH := $(realpath $(MY_DIR_PATH)..)


# Include common make functions.
include $(ROOT_DIR_PATH)/build/makefiles/show-help.mk
include $(ROOT_DIR_PATH)/build/makefiles/show-prompt.mk


# All available language versions.
LANGUAGES := go javascript python


# Other config variables.
PROJECT_NAME := easemlclient


# Function that repeats the same action for all languages.
define repeat-for-all
	for language in $(LANGUAGES) ; do \
        $(MAKE) -C $$language/$(PROJECT_NAME) $(1) $(if $(DIST_PATH),DIST_PATH=$(abspath $(DIST_PATH)/$$language),); \
    done
endef


# Handle export rules -- keep local variables private, pass everything else.
unexport SUMMARY CONTEXT_PATH MY_DIR_PATH LANGUAGES


.PHONY: clean
## Clean all the files resulting from building and testing.
clean:
	$(call show-prompt,Cleaning the build files)
	$(call repeat-for-all,$@)
ifneq ($(DIST_PATH),)
	# Clean the language directories if they were created.
	for language in $(LANGUAGES) ; do \
        rm -rf $(DIST_PATH)/$$language; \
    done
endif


.PHONY: build
## Build all project
build:
	$(call show-prompt,Compiling project code)
	$(call repeat-for-all,$@)


.PHONY: package
## Build the project and assemble a deployable package.
package:
	$(call show-prompt,Building the deployment package)
	$(call repeat-for-all,$@)


.PHONY: test
## Run all tests.
test:
	$(call show-prompt,Running all tests)
	$(call repeat-for-all,$@)


.PHONY: lint
## Run the linting checks.
lint:
	$(call show-prompt,Running all linting checks)
	$(call repeat-for-all,$@)


.PHONY: version
## Set the version of this package according to version file found in the repo root.
version:
	$(call show-prompt,Updating package version)
	$(call repeat-for-all,$@)

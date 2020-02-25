# Makefile the whole ease.ml project.

# Summary and context path of this makefile.
SUMMARY := This the main Makefile of the ease.ml project. All commands in this \
	       makefile will be performed on every single component of the project. \
		   To build individual components such as the javascript client or the engine, \
		   go to the corresponding component subdirectory.
CONTEXT_PATH := .
FOOTER := To specify the target directory for make package use the DIST_PATH environment variable \
		  \(default: DIST_PATH=./dist\).


# Paths to the parent directory of this makefile and the repo root directory.
MY_DIR_PATH := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR_PATH := $(realpath $(MY_DIR_PATH))


# Include common make functions.
include $(ROOT_DIR_PATH)/dev/makefiles/show-help.mk
include $(ROOT_DIR_PATH)/dev/makefiles/show-prompt.mk


# All available components. To include a component subdirectory in the build-all command, simply
# ensure it implements all commands (at least with an empty recipe) and add its directory name here.
COMPONENTS := client schema web engine


# Other config variables.
PROJECT_NAME := easeml


# Importable config variables.
ifeq ($(strip $(DIST_PATH)),)
	DIST_PATH := ./dist
endif


# Function that repeats the same action for all components.
define repeat-for-all
	for component in $(COMPONENTS) ; do \
        $(MAKE) -C $$component $(1) $(if $(DIST_PATH),DIST_PATH=$(abspath $(DIST_PATH)/$$component),); \
    done
endef


# Handle export rules -- keep local variables private, pass everything else.
unexport SUMMARY CONTEXT_PATH MY_DIR_PATH COMPONENTS


.PHONY: clean
## Clean all the files resulting from building and testing.
clean:
	$(call show-prompt,Cleaning the build files)
	$(call repeat-for-all,$@)
ifneq ($(DIST_PATH),)
	# Clean the component directories if they were created.
	for component in $(COMPONENTS) ; do \
        rm -rf $(DIST_PATH)/$$component; \
    done
	if [ -d $(DIST_PATH) ]; then \
        rmdir $(DIST_PATH) --ignore-fail-on-non-empty; \
    fi
endif


.PHONY: build
## Build all components.
build:
	$(call show-prompt,Compiling component code)
	$(call repeat-for-all,$@)


.PHONY: package
## Build all the components and assemble deployable packages and place them under the dist directory.
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

.PHONY: init
## Run initialization script in all Makefiles
init:
	$(call show-prompt,Running all linting checks)
	$(call repeat-for-all,$@)

.PHONY: version
## Set the version of all components according to version file found in the repo root. To update the version,
## make sure to first update the VERSION file.
version:
	$(call show-prompt,Updating package version)
	$(call repeat-for-all,$@)

# Makefile for the Python implementation of the ease.ml schema subproject.


# Summary and context path of this makefile.
SUMMARY := This Makefile corresponds to the Python implementation of the ease.ml client.
CONTEXT_PATH := client/python
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
PYTHON := $(shell which python)
PIP := $(shell which pip)
PIPENV := $(shell which pipenv)
PIPENV_RUN := $(PIPENV) run
VERSION := $(shell cat $(ROOT_DIR_PATH)/VERSION)


# Include common make functions.
include $(ROOT_DIR_PATH)/dev/makefiles/show-help.mk
include $(ROOT_DIR_PATH)/dev/makefiles/show-prompt.mk


.PHONY: init
## Initialize the development environment to enable running of build and test.
init:
	$(call show-prompt,Installing development dependencies)
ifeq ($(strip $(DIST_PATH)),)
	pip install pipenv
	PIPENV := $(shell which pipenv)
	PIPENV_RUN := $(PIPENV) run
endif
	$(PIPENV) install -e . --dev


.PHONY: clean
## Clean all the files resulting from building and testing.
clean:
	$(call show-prompt,Cleaning the build files)
	-$(PIPENV_RUN) python setup.py clean
	-rm -f ./$(DIST_PATH)/$(PROJECT_NAME)-$(VERSION)*.whl
	-rm -f ./$(DIST_PATH)/$(PROJECT_NAME)-$(VERSION)*.tar.gz
	-rm -rf ./dist
	-rm -rf ./build
	-find . -name '*.py[co]' -exec rm {} \;
	-find . -name '__pycache__' -exec rm -r {} \;
	-find . -name '*.egg-info' -exec rm -r {} \;


.PHONY: build
## Compiles the project and procudes PYC files.
build:
	$(call show-prompt,Compiling project code)
	$(PIPENV_RUN) python -m compileall .


.PHONY: package
## Build the project and assemble a deployable package.
package:
	$(call show-prompt,Building the deployment package)
	$(PIPENV_RUN) python setup.py sdist -d $(DIST_PATH)
	$(PIPENV_RUN) python setup.py bdist_wheel -d $(DIST_PATH)


.PHONY: test
## Run all tests.
test:
	$(call show-prompt,Running all tests)
	$(PIPENV_RUN) python -m unittest discover ./test


.PHONY: lint
## Run the linting checks.
lint:
	$(call show-prompt,Running all linting checks)
	$(PIPENV_RUN) pylint --rcfile=$(ROOT_DIR_PATH)/.pylintrc $(PROJECT_NAME) -f parseable -r n && \
    $(PIPENV_RUN) mypy --silent-imports $(PROJECT_NAME) && \
    $(PIPENV_RUN) pycodestyle $(PROJECT_NAME) --max-line-length=120


.PHONY: lint-fix
## Automatically fix style errors where possible.
lint-fix:
	$(call show-prompt,Fixing coding style violations)
	$(PIPENV_RUN) pipenv run autopep8 --in-place --aggressive --aggressive --aggressive \
				  --max-line-length 120 --recursive $(PROJECT_NAME) test


.PHONY: version
## Set the version of this package according to version file found in the repo root.
version:
	$(call show-prompt,Updating package version)
	echo __version__ = \"$(VERSION)\" > easemlschema/__init__.py
	@echo New version: $(VERSION)

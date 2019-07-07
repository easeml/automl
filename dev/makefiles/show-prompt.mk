# This string will either be completely empty or will be of the form @(CONTEXT_PATH).
CONTEXT_PATH_STRING = $(if $(CONTEXT_PATH:-=),@($(CONTEXT_PATH)),)

# A makefile function used to pretty print a prompt message reporting a section of progress.
define show-prompt
	@echo
	@echo "$$(tput bold) $$(tput  setaf 6) -> $$(tput  sgr0)$$(tput bold) $(1) $$(tput sgr0)   $(CONTEXT_PATH_STRING)"
	@echo
endef

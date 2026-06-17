APP_NAME := myapp
BIN_DIR  := bin

# Common development tasks (build, run, dev, test, fmt, vet, tidy, swag, migrate)
# are provided by the `crank` CLI, so they are intentionally NOT duplicated here.
# Run `crank tools` to list them. This Makefile holds only targets that crank does
# not provide natively — add your own project-specific targets below. They can be
# run with either `make <target>` or `crank <target>` (crank transparently
# delegates unknown commands to this Makefile).

.PHONY: help clean

help:
	@echo "Common tasks are provided by the crank CLI:"
	@echo "  crank build        - compile the binary into $(BIN_DIR)/$(APP_NAME)"
	@echo "  crank run          - run the application"
	@echo "  crank dev          - run with live reload (air)"
	@echo "  crank test         - run all tests"
	@echo "  crank swag         - generate Swagger docs"
	@echo "  crank migrate up   - apply database migrations"
	@echo "  crank make migration <name> - create an up/down migration pair"
	@echo ""
	@echo "Project-specific Makefile targets:"
	@echo "  make clean       - remove build artifacts ($(BIN_DIR)/)"

clean:
	rm -rf $(BIN_DIR)

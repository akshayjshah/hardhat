.PHONY: lint
lint:
	@rm -rf lint.log
	@gofmt -d -s . 2>&1 | tee lint.log
	@go vet ./... 2>&1 | tee -a lint.log
	@golint $(shell go list ./...) 2>&1 | tee -a lint.log
	@git grep -i fixme | grep -v -e vendor -e Makefile | tee -a lint.log
	@[ ! -s lint.log ]

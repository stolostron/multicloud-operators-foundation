.PHONY: go/gosec-install
## Installs latest release of Gosec
go/gosec-install:
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $(GOPATH)/bin


.PHONY: go/gosec
## Runs gosec in quiet mode (meaning output only if issues found). Any findings will be printed to stdout.
go/gosec: go/gosec-install
	gosec --quiet ./...
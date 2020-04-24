.PHONY: sonar/go
## Run SonarCloud analysis for Go. This will generally be run as part of a Travis build, not during local development.
sonar/go: go/gosec-install
	go test -coverprofile=coverage.out -json ./... > report.json
	gosec --quiet -fmt sonarqube -out gosec.json -no-fail ./...
	unset SONARQUBE_SCANNER_PARAMS
	sonar-scanner --debug

# This expects that your code uses Jest to execute tests.
# Add this field to your jest.config.js file to generate the report:
#     testResultProcessor: 'jest-sonar-reporter',
# It must be run before make component/test/unit.
.PHONY: sonar/js/jest-init
## Installs npm module to generate Sonar test report output from Jest. This will generally be run as part of a Travis build, not during local development.
sonar/js/jest-init:
	npm install -D jest-sonar-reporter

# Test reports and code coverage must be generated before running the scanner.
# It must be run after make component/test/unit.
.PHONY: sonar/js
## Runs the SonarCloud analysis for JavaScript. This will generally be run as part of a Travis build, not during local development.
sonar/js:
	unset SONARQUBE_SCANNER_PARAMS
	sonar-scanner --debug

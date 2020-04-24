## Extensions to `build-harness`

This repo is structured just like `build-harness`, and is pulled in via:

```BUILD_HARNESS_EXTENSIONS_PATH```

In order to use the build harness and extensions, make yourself a [token](https://github.com/settings/tokens) that at least has `repo` access add the following to your `Makefile`:

```
# GITHUB_USER containing '@' char must be escaped with '%40'
GITHUB_USER := $(shell echo $(GITHUB_USER) | sed 's/@/%40/g')
GITHUB_TOKEN ?=

-include $(shell [ -f ".build-harness-bootstrap" ] || curl -sSL -o .build-harness-bootstrap -H "Authorization: token $(GITHUB_TOKEN)" -H "Accept: application/vnd.github.v3.raw" "https://raw.github.com/open-cluster-management/build-harness-extensions/master/templates/Makefile.build-harness-bootstrap"; echo .build-harness-bootstrap)
```

Some OSes seem to have trouble with the V3 API for github - here is an alternate invocation that uses the V4 API:
```
-include $(shell [ -f ".build-harness-bootstrap" ] || curl -H 'Authorization: token ${GITHUB_TOKEN}' -H 'Accept: application/vnd.github.v4.raw' -L https://api.github.com/repos/open-cluster-management/build-harness-extensions-test/contents/templates/Makefile.build-harness-bootstrap -o .build-harness-bootstrap; echo .build-harness-bootstrap)

```

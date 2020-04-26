VENDORIZATION_DIRECTORY ?= vbh
VENDORIZATION_TEMPLATE_FILE ?= $(BUILD_HARNESS_EXTENSIONS_PATH)/templates/Makefile.build-harness-vendorized

.PHONY : vendorize
## vendorize the build harness
vendorize::
	@mkdir -p vbh
	@cp -r $(BUILD_HARNESS_PATH) $(VENDORIZATION_DIRECTORY)/$(BUILD_HARNESS_PROJECT)
	@rm -rf $(VENDORIZATION_DIRECTORY)/$(BUILD_HARNESS_PROJECT)/.git
	@cp -r $(BUILD_HARNESS_EXTENSIONS_PATH) $(VENDORIZATION_DIRECTORY)/$(BUILD_HARNESS_EXTENSIONS_PROJECT)
	@rm -rf $(VENDORIZATION_DIRECTORY)/$(BUILD_HARNESS_EXTENSIONS_PROJECT)/.git
	@cp $(VENDORIZATION_TEMPLATE_FILE) $(VENDORIZATION_DIRECTORY)/.build-harness-vendorized


# todo -- auto-modify the Makefile to reference vendorization -- .gitignore/Makefile changes
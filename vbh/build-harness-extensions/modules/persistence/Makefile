# Simple persistence (fancy word for saving, loading, and deleting) via S3

# The following environment variables must be defined so the persistence script can see them:
# S3KEY - The S3 key
# S3SECRET - The S3 secret
# S3BUCKET - The top-level S3 bucket to read/write
# S3STORAGETYPE - Storage type like STANDARD, REDUCED_REDUNDANCY, etc.
# AWS_SERVICE - Service type, generally 'S3'
# AWS_REGION - Your region, eg. us-east-1
# PERSIST_FILE - either a single file name or a full, ABSOLUTE path to a file name
# PERSIST_PATH - An additional path on S3 to insert between the bucket and destination file; use '/' for the root of the bucket

.PHONY: persistence/get-file
## Retrieve a file
persistence/get-file: %get-file:
	@echo "Retrieving file: $(PERSIST_FILE)"
	$(shell $(BUILD_HARNESS_PATH)/../build-harness-extensions/modules/persistence/bin/s3_persist.sh get $(PERSIST_PATH) $(PERSIST_FILE))

.PHONY: persistence/put-file
## Save a file
persistence/put-file: %put-file:
	@echo "Saving file: $(PERSIST_FILE)"
	$(shell $(BUILD_HARNESS_PATH)/../build-harness-extensions/modules/persistence/bin/s3_persist.sh put $(PERSIST_PATH) $(PERSIST_FILE))

.PHONY: persistence/delete-file
## Delete a file
persistence/delete-file: %delete-file:
	@echo "Deleting file: $(PERSIST_FILE)"
	$(shell $(BUILD_HARNESS_PATH)/../build-harness-extensions/modules/persistence/bin/s3_persist.sh delete $(PERSIST_PATH) $(PERSIST_FILE))

#!/bin/bash
# gets/puts file to S3
# s3_persist [put|get] <s3_folder_path> <absolute_path_to_file>

# put file into S3
function putS3
{
  s3_folder_path=$2
  file_path=$3
  date=$(date -R)
  acl="x-amz-acl:private"
  content_type="application/x-compressed-tar"
  storage_type="x-amz-storage-class:${S3STORAGETYPE}"
  string="PUT\n\n$content_type\n$date\n$acl\n$storage_type\n/$S3BUCKET$s3_folder_path${file_path##/*/}"
  signature=$(echo -en "${string}" | openssl sha1 -hmac "${S3SECRET}" -binary | base64)

  curl -L --retry 3 --retry-delay 10 -X PUT -T "$file_path" \
       -H "Host: $S3BUCKET.${AWS_SERVICE}.amazonaws.com" \
       -H "Date: $date" \
       -H "Content-Type: $content_type" \
       -H "$storage_type" \
       -H "$acl" \
       -H "Authorization: AWS ${S3KEY}:$signature" \
       "https://$S3BUCKET.${AWS_SERVICE}.amazonaws.com$s3_folder_path${file_path##/*/}"

  return $?
}

# delete a file from S3
function deleteS3
{
  s3_folder_path=$2
  file_path=$3
  date=$(date -R)
  acl="x-amz-acl:private"
  content_type="application/x-compressed-tar"
  string="DELETE\n\n$content_type\n$date\n$acl\n/$S3BUCKET$s3_folder_path${file_path##/*/}"
  signature=$(echo -en "${string}" | openssl sha1 -hmac "${S3SECRET}" -binary | base64)

  curl -L --retry 3 --retry-delay 10 -X DELETE -T "$file_path" \
       -H "Host: $S3BUCKET.${AWS_SERVICE}.amazonaws.com" \
       -H "Date: $date" \
       -H "Content-Type: $content_type" \
       -H "$acl" \
       -H "Authorization: AWS ${S3KEY}:$signature" \
       "https://$S3BUCKET.${AWS_SERVICE}.amazonaws.com$s3_folder_path${file_path##/*/}"

  return $?
}

# get file from S3
function getS3
{
  file_path=$3
  bucket="${S3BUCKET}"

  AWS_SERVICE_ENDPOINT_URL="${S3BUCKET}.${AWS_SERVICE}.amazonaws.com"

  CURRENT_DATE_DAY="$(date -u '+%Y%m%d')"
  CURRENT_DATE_TIME="$(date -u '+%H%M%S')"
  CURRENT_DATE_ISO8601="${CURRENT_DATE_DAY}T${CURRENT_DATE_TIME}Z"

  HTTP_REQUEST_METHOD='GET'
  HTTP_REQUEST_PAYLOAD=''
  HTTP_REQUEST_PAYLOAD_HASH="$(printf "${HTTP_REQUEST_PAYLOAD}" | \
    openssl dgst -sha256 | sed 's/^.* //')"
  HTTP_CANONICAL_REQUEST_URI=${file_path##/*/}
  HTTP_CANONICAL_REQUEST_QUERY_STRING=''
  HTTP_REQUEST_CONTENT_TYPE='application/x-www-form-urlencoded'

  HTTP_CANONICAL_REQUEST_HEADERS="\
content-type:${HTTP_REQUEST_CONTENT_TYPE}
host:${AWS_SERVICE_ENDPOINT_URL}
x-amz-content-sha256:${HTTP_REQUEST_PAYLOAD_HASH}
x-amz-date:${CURRENT_DATE_ISO8601}"
  # Note: The signed headers must match the canonical request headers.
HTTP_REQUEST_SIGNED_HEADERS="\
content-type;host;x-amz-content-sha256;x-amz-date"

  HTTP_CANONICAL_REQUEST="\
${HTTP_REQUEST_METHOD}
/${HTTP_CANONICAL_REQUEST_URI##/*/}
${HTTP_CANONICAL_REQUEST_QUERY_STRING}
${HTTP_CANONICAL_REQUEST_HEADERS}\n
${HTTP_REQUEST_SIGNED_HEADERS}
${HTTP_REQUEST_PAYLOAD_HASH}"

  SIGNATURE="$(create_signature)"

  HTTP_REQUEST_AUTHORIZATION_HEADER="\
AWS4-HMAC-SHA256 Credential=${S3KEY}/${CURRENT_DATE_DAY}/\
${AWS_REGION}/${AWS_SERVICE}/aws4_request, \
SignedHeaders=${HTTP_REQUEST_SIGNED_HEADERS};x-amz-date, Signature=${SIGNATURE}"

  curl -X "${HTTP_REQUEST_METHOD}" \
      "https://${AWS_SERVICE_ENDPOINT_URL}/${HTTP_CANONICAL_REQUEST_URI}" \
      -H "Authorization: ${HTTP_REQUEST_AUTHORIZATION_HEADER}" \
      -H "content-type: ${HTTP_REQUEST_CONTENT_TYPE}" \
      -H "x-amz-content-sha256: ${HTTP_REQUEST_PAYLOAD_HASH}" \
      -H "x-amz-date: ${CURRENT_DATE_ISO8601}" -o ${file_path}

  return $?
}

# Create an SHA-256 hash in hexadecimal.
# Usage:
#   hash_sha256 <string>
function hash_sha256 {
  printf "${1}" | openssl dgst -sha256 | sed 's/^.* //'
}

# Create an SHA-256 hmac in hexadecimal.
# Usage:
#   hmac_sha256 <key> <data>
function hmac_sha256 {
  key="$1"
  data="$2"
  printf "${data}" | openssl dgst -sha256 -mac HMAC -macopt "${key}" | \
      sed 's/^.* //'
}

# Create the signature.
# Usage:
#   create_signature
function create_signature {
  stringToSign="AWS4-HMAC-SHA256
${CURRENT_DATE_ISO8601}
${CURRENT_DATE_DAY}/${AWS_REGION}/${AWS_SERVICE}/aws4_request
$(hash_sha256 "${HTTP_CANONICAL_REQUEST}")"

  dateKey=$(hmac_sha256 key:"AWS4${S3SECRET}" \
      "${CURRENT_DATE_DAY}")
  regionKey=$(hmac_sha256 hexkey:"${dateKey}" "${AWS_REGION}")
  serviceKey=$(hmac_sha256 hexkey:"${regionKey}" "${AWS_SERVICE}")
  signingKey=$(hmac_sha256 hexkey:"${serviceKey}" "aws4_request")

  if [ "$BUILD_HARNESS_OS" == "darwin" ]; then
  printf "${stringToSign}" | openssl dgst -sha256 -mac HMAC -macopt \
      hexkey:"${signingKey}" | awk '{print $0}'; else
  printf "${stringToSign}" | openssl dgst -sha256 -mac HMAC -macopt \
      hexkey:"${signingKey}" | awk '{print $2}'; fi
}

function usage
{
  echo "Usage: $0 [put|get] <s3_folder_path> <absolute_path_to_file>"
  echo "    $0 '/tmp/storage_backup/storage-backup_07192017_112945.tar.gz' '/'"
}

#validate some positional parameters are present
if [ "$1" = "get" ]; then
  getS3 $1 $2 $3
elif [ "$1" = "put" ]; then
  putS3 $1 $2 $3
elif [ "$1" = "delete" ]; then
  deleteS3 $1 $2 $3
else
  usage
  exit 2
fi

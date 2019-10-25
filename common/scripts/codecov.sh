#!/bin/bash
#
# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e
set -u
set -o pipefail

# turn off GOSUMDB check
export GOSUMDB=off

# add $GOPATH/bin to $PATH
# TODO: add $GOPATH/bin to $PATH in build-tools image
export PATH=${PATH}:${GOPATH}/bin

ROOTDIR="$(cd "$(dirname "$0")"/../.. ; pwd -P)"
REPORT_PATH=${REPORT_PATH:-"${GOPATH}/out/codecov"}
#CODECOV_SKIP=${GOPATH}/out/codecov/codecov.skip
MAXPROCS="${MAXPROCS:-}"
mkdir -p "${GOPATH}"/out/codecov

DIR="./..."
#SKIPPED_TESTS_GREP_ARGS=
TEST_RETRY_COUNT=3

if [ "${1:-}" != "" ]; then
    DIR="./$1/..."
fi

COVERAGEDIR="$(mktemp -d /tmp/test_coverage.XXXXXXXXXX)"
mkdir -p "$COVERAGEDIR"

# half the number of cpus seem to saturate
if [[ -z ${MAXPROCS:-} ]]; then
    MAXPROCS=$(($(getconf _NPROCESSORS_ONLN)/2))
fi

function code_coverage() {
  local filename
  local count=${2:-0}
  filename="$(echo "${1}" | tr '/' '-')"
  go test \
    -coverprofile="${COVERAGEDIR}/${filename}.cov" \
    -covermode=atomic "${1}" \
    | tee "${COVERAGEDIR}/${filename}.report" \
    | tee >(go-junit-report > "${COVERAGEDIR}/${filename}-junit.xml") \
    && RC=$? || RC=$?

  if [[ ${RC} != 0 ]]; then
    if (( count < TEST_RETRY_COUNT )); then
      code_coverage "${1}" $((count+1))
    else
      echo "${1}" | tee "${COVERAGEDIR}/${filename}.err"
    fi
  fi

  #remove skipped tests from .cov file
#   remove_skipped_tests_from_cov "${COVERAGEDIR}/${filename}.cov"
}

function wait_for_proc() {
  local num
  num=$(jobs -p | wc -l)
  while [ "${num}" -gt ${MAXPROCS} ]; do
    sleep 2
    num=$(jobs -p|wc -l)
  done
}

# function parse_skipped_tests() {
#   while read -r entry; do
#     if [[ "${SKIPPED_TESTS_GREP_ARGS}" != '' ]]; then
#       SKIPPED_TESTS_GREP_ARGS+='\|'
#     fi
#     if [[ "${entry}" != "#"* ]]; then
#       SKIPPED_TESTS_GREP_ARGS+="\\(${entry}\\)"
#     fi
#   done < "${CODECOV_SKIP}"
# }

# function remove_skipped_tests_from_cov() {
#   while read -r entry; do
#     entry="$(echo "${entry}" | sed 's/\//\\\//g')"
#     sed -i "/${entry}/d" "$1"
#   done < "${CODECOV_SKIP}"
# }

cd "${ROOTDIR}"

# parse_skipped_tests

# For generating junit.xml files
go get github.com/jstemmer/go-junit-report

echo "Code coverage test (concurrency ${MAXPROCS})"
for P in $(go list "${DIR}" | grep -v vendor); do
#   if echo "${P}" | grep -q "${SKIPPED_TESTS_GREP_ARGS}"; then
#     echo "Skipped ${P}"
#     continue
#   fi
  code_coverage "${P}" &
  wait_for_proc
done

wait

######################################################
# start generating report
######################################################
touch "${COVERAGEDIR}/empty"
mkdir -p "${REPORT_PATH}"
pushd "${REPORT_PATH}"

# Build the combined coverage files
go get github.com/wadey/gocovmerge
gocovmerge "${COVERAGEDIR}"/*.cov > coverage.cov
cat "${COVERAGEDIR}"/*.report > report.out

# Build the combined junit.xml
go get github.com/imsky/junit-merger/src/junit-merger
junit-merger "${COVERAGEDIR}"/*-junit.xml > junit.xml

popd

echo "Intermediate files were written to ${COVERAGEDIR}"
echo "Final reports are stored in ${REPORT_PATH}"

if ls "${COVERAGEDIR}"/*.err 1> /dev/null 2>&1; then
  echo "The following tests had failed:"
  cat "${COVERAGEDIR}"/*.err
  exit 1
fi
######################################################
# end generating report
######################################################

# Upload to codecov.io in post submit only for visualization
bash <(curl -s https://codecov.io/bash) -t "${CODECOV_TOKEN}" -f "${REPORT_PATH}/coverage.cov"

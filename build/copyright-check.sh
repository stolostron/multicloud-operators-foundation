#!/bin/bash

#Project start year
origin_year=2016
#Back up year if system time is null or incorrect
back_up_year=2019
#Currrent year
current_year=$(date +"%Y")

TRAVIS_BRANCH=$1
TRAVIS_PULL_REQUEST_BRANCH=$2
export TRAVIS_PULL_REQUEST_BRANCH

CHANGED_FILES=$(git diff --name-only --diff-filter=AM "${TRAVIS_BRANCH}")
echo "CHANGED_FILES=" "$CHANGED_FILES"

if [ -z "$current_year" ] || [ "$current_year" -lt "$origin_year" ]; then
  echo "Can't get correct system time"
  current_year=$back_up_year
fi

#lic_year to scan for year formart single line's correctness
lic_year=()
#All possible combination within [origin_year, current_year] range is valid format
#seq isn't recommanded after bash version 3.0
for ((start_year=origin_year;start_year<=current_year;start_year++));
do
  lic_year+=(" (C) Copyright IBM Corporation ${start_year}. All Rights Reserved")
  for ((end_year=start_year+1;end_year<=current_year;end_year++));
  do
    lic_year+=(" (C) Copyright IBM Corporation ${start_year}, ${end_year} All Rights Reserved")
  done
done
lic_year_size=${#lic_year[@]}

#lic_rest to scan for rest copyright format's correctness
lic_rest=()
lic_rest+=(" licensed Materials - Property of IBM")
lic_rest+=(" US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.")
lic_rest+=(" Copyright (c) 2020 Red Hat, Inc.")
lic_rest_size=${#lic_rest[@]}

#Used to signal an exit
ERROR=0


echo "##### Copyright check #####"
#Loop through all files. Ignore .FILENAME types
#for f in `find .. -type f ! -path "../.eslintrc.js" ! -path "../build-harness/*" ! -path "../auth-setup/*" ! -path "../sslcert/*" ! -path "../node_modules/*" ! -path "../coverage/*" ! -path "../test-output/*" ! -path "../build/*" ! -path "../nls/*" ! -path "../public/*"`; do
for f in $CHANGED_FILES; do
  FILETYPE=$(basename "${f##*.}")
  case "${FILETYPE}" in
  	js | go | scss | java | rb)
  		;;
  	*)
      continue
  esac

  #Read the first 15 lines, most Copyright headers use the first 10 lines.
  header=$(head -15 "$f")
  printf " Scanning  . .%s . " "$f"

  #Check for year copyright single line
  year_line_count=0
  for ((i=0;i<lic_year_size;i++));
  do
    #Validate year formart within [origin_year, current_year] range
    if [[ "$header" == *"${lic_year[$i]}"* ]]; then
      year_line_count=$((year_line_count + 1))
    fi
  done

  #Must find and only find one line valid year, otherwise invalid copyright formart
  if [[ $year_line_count != 1 ]]; then
    printf "Missing copyright\n  >>Could not find correct copyright year in the file %s\n" "$f"
    ERROR=1
    break
  fi

  #Check for rest copyright lines
  for ((i=0;i<lic_rest_size;i++));
  do
    #Validate the copyright line being checked is present
    if [[ "$header" != *"${lic_rest[$i]}"* ]]; then
      printf "Missing copyright\n  >>Could not find [${lic_rest[$i]}] in the file %s\n" "$f"
      ERROR=1
      break 2
    fi
  done

  #Add a status message of OK, if all copyright lines are found
  if [[ "$ERROR" == 0 ]]; then
    printf "OK\n"
  fi
done

echo "##### Copyright check ##### ReturnCode: ${ERROR}"
exit $ERROR

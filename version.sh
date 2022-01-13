#!/bin/bash
# This script generates a "full" version number:
# Pattern: <major>.<minor>.<bugfix>r<revision>_<whatsmeow date>
# Example: 1.0.0r42_20220101010101
# The actual whatsmeow date is only known for sure if build has happened before
# so whatsmeow is available in GOPATH.
# The version string must not end in a new-line else it will mess up build scripts.
cd "$(dirname "$0")"
version="$(cat VERSION)"
revision="$(git rev-list --count HEAD)"

if [[ -n "${GOPATH}" ]]
then
  wmdate="_$(cd src/go && go list -m -f '{{.Dir}}' go.mau.fi/whatsmeow | rev | cut -d'-' -f 2 | rev)"
else
  wmdate="_$(curl --silent -H 'Accept: application/vnd.github.v3+json' https://api.github.com/repos/tulir/whatsmeow/commits?per_page=1 | jq -r '.[].commit.committer.date' | tr -d 'TZ:-')"
  echo "Warning: whatsmeow date has been guessed. Get version after build to be sure." >2
fi

printf "%sr%s%s" "${version}" "${revision}" "${wmdate}"

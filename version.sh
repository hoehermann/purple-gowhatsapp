#!/bin/bash
# This script generates a "full" version number:
# Pattern: <major>.<minor>.<bugfix>r<revision>_<whatsmeow date>
# Example: 1.0.0r42_20220101010101
# The whatsmeow date is only available if build has happened before
# in build/ with standard cmake configuration.
# The version string must not end in a new-line else it will mess up build scripts.
BUILDDIR="build"
version="$(cat VERSION)"
revision="$(git rev-list --count HEAD)"
export GOPATH=$(realpath "${BUILDDIR}/src/go/go")
(
  cd "src/go"
  if [[ -n "${GOPATH}" ]]
  then
    go mod tidy
    wmdate="_$(go list -m -f '{{.Dir}}' go.mau.fi/whatsmeow | rev | cut -d'-' -f 2 | rev)"
  else
    echo "Warning: whatsmeow date is known after build only." >2
  fi
  printf "%sr%s%s" "${version}" "${revision}" "${wmdate}"
)

#!/bin/bash

set -e
PROCS="$(nproc)"
((PROCS=PROCS-1))
JOBS=$PROCS

echo $JOBS
# It appears that there may be read-only files as part of golang's default behavior.
# We will set these all RW so they can be removed
if [ -d "./build" ]
then
  chmod -R 755 build
  rm -rf build
fi

mkdir build
cd build

cmake ..
cmake --build "." -j$JOBS

echo "Now calling sudo make install"
sudo make install/strip DESTDIR=destdir

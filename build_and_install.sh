#!/bin/bash

set -e
PROCS="$(nproc)"
((PROCS=PROCS-1))
JOBS=$PROCS

# It appears that there may be read-only files as part of golang's default behavior.
# We will set these all RW so they can be removed
if [ -d "./build" ]
then
  chmod -R 755 build
  rm -rf build
fi

mkdir build
cd build
cmake -DCMAKE_BUILD_TYPE=Release ..
cmake -j "${JOBS}" --build .

echo "Now calling sudo make install"
sudo make install/strip DESTDIR=destdir

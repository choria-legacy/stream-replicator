#!/bin/bash

set -e
set -x

if [ -z $NAME ]
then
  NAME="stream-replicator"
fi

if [ -z $VERSION ]
then
  echo "VERSION has not been set, cannot build"
  exit 1
fi

if [ ! -d /build ]
then
  echo "/build is not mounted, cannot build"
  exit 1
fi

TARBALL="${NAME}-${VERSION}.tgz"

chown root:root "/build/${TARBALL}"

rpmbuild --target ${ARCH} -ta "/build/${TARBALL}"

cp -v /usr/src/redhat/RPMS/*/* /build
cp -v /usr/src/redhat/SRPMS/* /build
#!/bin/bash

set -e
set -x

if [ -z $NAME ]
then
  NAME="stream-replicator"
fi

if [ -z $BINDIR ]
then
  BINDIR="/usr/sbin"
fi

if [ -z $ETCDIR ]
then
  ETCDIR="/etc/${NAME}"
fi

if [ -z $ARTIFACTS ]
then
  ARTIFACTS="store:/build"
fi

if [ -z $ARCH ]
then
  echo "ARCH has not been set, cannot build"
  exit 1
fi

if [ -z $DIST ]
then
  echo "DIST has not been set, cannot build"
  exit 1
fi

if [ -z $VERSION ]
then
  echo "VERSION has not been set, cannot build"
  exit 1
fi

if [ -z $RELEASE ]
then
  RELEASE="1"
fi

if [ -z $MANAGE_CONF ]
then
  MANAGE_CONF=1
fi

WORKDIR="/tmp/build/${ARCH}/${NAME}-${VERSION}"
BINARY="stream-replicator-${VERSION}-linux-${ARCH}"
TARBALL="${NAME}-${VERSION}.tgz"

if [ ! -f "build/${BINARY}" ]
then
  echo "build/${BINARY} does not exist, cannot build"
  exit 1
fi

mkdir -p ${WORKDIR}/dist

/usr/bin/find build/dist -maxdepth 1 -type f | xargs -I {} -n 1 cp -v {} ${WORKDIR}/dist
cp -v build/dist/${DIST}/* ${WORKDIR}/dist

for i in $(find ${WORKDIR}/dist -type f); do
  sed -i "s!{{pkgname}}!${NAME}!g" ${i}
  sed -i "s!{{bindir}}!${BINDIR}!g" ${i}
  sed -i "s!{{etcdir}}!${ETCDIR}!g" ${i}
  sed -i "s!{{version}}!${VERSION}!g" ${i}
  sed -i "s!{{iteration}}!${RELEASE}!g" ${i}
  sed -i "s!{{dist}}!${DIST}!g" ${i}
  sed -i "s!{{manage_conf}}!${MANAGE_CONF}!g" ${i}

  sed -i "s!{{go_arch}}!${ARCH}!g" ${i}
  
  if [ $ARCH = "amd64" ]; then
    sed -i "s!{{arch}}!x86_64!g" ${i}
  else
    sed -i "s!{{arch}}!i386!g" ${i}
  fi


done

cp "build/${BINARY}" ${WORKDIR}

cd $(dirname ${WORKDIR})

tar -cvzf ${TARBALL} $(basename ${WORKDIR})

docker cp ${TARBALL} ${ARTIFACTS}/

cd -

docker cp build/build.sh ${ARTIFACTS}/
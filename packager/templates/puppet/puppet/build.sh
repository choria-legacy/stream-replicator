#!/bin/bash

set -x

cd {{cpkg_source_dir}}/puppet

sed -i.bak -re "s/(.+\"version\": \").+/\1{{cpkg_version}}\",/" metadata.json
sed -i.bak -re "s/(.+version = \").+/\1{{cpkg_version}}\",/" manifests/init.pp

find . -name \*.bak -delete

/opt/puppetlabs/bin/puppet module build .

cp pkg/*.tar.gz /tmp

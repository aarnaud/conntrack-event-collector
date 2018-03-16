#!/bin/bash

set -e
set -x

SCRIPT_DIR="$(dirname "${PWD}/$0")"


function build() {
    # Build program
    cd "${SCRIPT_DIR}/../"
    if [[ -z ${GOARCH} ]]
    then
        export GOARCH=amd64 # OR GOARCH=mipsle
    fi
    CGO_ENABLED=0 go build -a -o openwrt/data/usr/sbin/owp-conntrack-event-collector
}

build

# Copy config and init file
cd "${SCRIPT_DIR}"
mkdir -p data/etc/init.d/ data/etc/config/owp/
cp owp-conntrack-event-collector.initd data/etc/init.d/owp-conntrack-event-collector
cp conntrack-event-collector.yml data/etc/config/owp/

# Make package
mkdir -p build
cd control
tar -czpvf ../build/control.tar.gz ./
cd -
cd data
tar -czpvf ../build/data.tar.gz ./
cd -
cd build/
tar -czpvf owp-conntrack-event-collector_${GOARCH}.ipk control.tar.gz data.tar.gz
cd -

# Cleaning
rm -rf build/control.tar.gz build/data.tar.gz data
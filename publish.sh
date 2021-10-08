#!/bin/bash

set -x
set -e

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

pushd $SCRIPT_DIR/frontend
yarn build --mode=production
cd ../standin
yarn build --mode=production
cd ..
go build

mkdir -p $SCRIPT_DIR/release/frontend/dist
rsync -av $SCRIPT_DIR/frontend/dist/ $SCRIPT_DIR/release/frontend/dist/

mkdir -p $SCRIPT_DIR/release/standin/dist
rsync -av $SCRIPT_DIR/standin/dist/ $SCRIPT_DIR/release/standin/dist/

sudo cp $SCRIPT_DIR/conf/logrotate.d/crutch /etc/logrotate.d/
sudo cp $SCRIPT_DIR/conf/rsyslog.d/01-crutch.conf /etc/rsyslog.d

sudo cp $SCRIPT_DIR/conf/system/crutch.service /etc/systemd/system/
sudo systemctl daemon-reload

sudo systemctl stop crutch; 
cp $SCRIPT_DIR/crutch $SCRIPT_DIR/release/; 
sudo systemctl start crutch

cd frontend
yarn build --mode=development
cd ../standin
yarn build --mode=development

popd

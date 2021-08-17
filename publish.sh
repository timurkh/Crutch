#!/bin/bash

set -x
set -e

pushd /home/severstal/s/crutch/frontend
yarn build --mode=production
cd ..
go build

rsync -av frontend/dist/ release/frontend/dist/

sudo cp crutch.service /etc/systemd/system/
sudo systemctl daemon-reload

sudo systemctl stop crutch; 
cp /home/severstal/s/crutch/crutch /home/severstal/s/crutch/release/; 
sudo systemctl start crutch

cd frontend
yarn build --mode=development

popd

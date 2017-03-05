#!/bin/bash

source .env

export GOPATH=$APP_ENGINE_LOCATION/gopath/

$APP_ENGINE_LOCATION/goapp serve

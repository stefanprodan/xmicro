#!/bin/bash
set -e

image="xmicro"
network="xmicro"

# build image
if [ ! "$(docker images -q  $image)" ];then
    docker build -t $image .
fi

# create network
if [ ! "$(docker network ls --filter name=$network -q)" ];then
    docker network create $network
fi

hostIP="$(ifconfig eth0 | sed -En 's/.*inet (addr:)?(([0-9]*\.){3}[0-9]*).*/\2/p')"

# start proxy
docker run -d -p 8000:8000 \
--name "${image}-proxy" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="${image}-proxy" \
-e SERVICE_TAGS="xmicro,proxy" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
-env=DEBUG \
-port=8000 \
-role=proxy 

# start frontend
docker run -d -p 8010:8000 \
--name "${image}-frontend" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="${image}-frontend" \
-e SERVICE_TAGS="xmicro,frontend" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
-env=DEBUG \
-port=8000 \
-role=frontend 

# start backend
docker run -d -p 8020:8000 \
--name "${image}-backend" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="${image}-backend" \
-e SERVICE_TAGS="xmicro,backend" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
-env=DEBUG \
-port=8000 \
-role=backend 

# start storage
docker run -d -p 8030:8000 \
--name "${image}-storage" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="${image}-storage" \
-e SERVICE_TAGS="xmicro,storage" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
-env=DEBUG \
-port=8000 \
-role=storage 
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

hostIP="$(hostname -I|awk '{print $1}')"

# proxy
node="${image}-proxy"
role="proxy"
docker run -d -p 8000:8000 \
-h "$node" \
--name "$node" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="$node" \
-e SERVICE_TAGS="$role" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
xmicro -env=DEBUG \
-port=8000 \
-role=$role 

# shard1 primary
node="${image}-node1"
role="shard1"
docker run -d -p 8001:8000 \
-h "$node" \
--name "$node" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="$node" \
-e SERVICE_TAGS="le,$role" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
xmicro -env=DEBUG \
-port=8000 \
-role=$role 

# shard2 primary
node="${image}-node2"
role="shard2"
docker run -d -p 8002:8000 \
-h "$node" \
--name "$node" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="$node" \
-e SERVICE_TAGS="le,$role" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
xmicro -env=DEBUG \
-port=8000 \
-role=$role 

# shard1 standby
node="${image}-node1-standby"
role="shard1"
docker run -d -p 8003:8000 \
-h "$node" \
--name "$node" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="$node" \
-e SERVICE_TAGS="le,$role" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
xmicro -env=DEBUG \
-port=8000 \
-role=$role 

# shard2 standby
node="${image}-node2-standby"
role="shard2"
docker run -d -p 8004:8000 \
-h "$node" \
--name "$node" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="$node" \
-e SERVICE_TAGS="le,$role" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
xmicro -env=DEBUG \
-port=8000 \
-role=$role
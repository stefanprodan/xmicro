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

# start node1
node="${image}-node1"
docker run -d -p 8001:8000 \
-h "$node" \
--name "$node" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="$node" \
-e SERVICE_TAGS="xmicro,frontend" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
xmicro -env=DEBUG \
-port=8000 \
-role=shard1 

# start node2
node="${image}-node2"
docker run -d -p 8002:8000 \
-h "$node" \
--name "$node" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="$node" \
-e SERVICE_TAGS="xmicro,frontend" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
xmicro -env=DEBUG \
-port=8000 \
-role=shard2 

# start node3
node="${image}-node1-replica"
docker run -d -p 8003:8000 \
-h "$node" \
--name "$node" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="$node" \
-e SERVICE_TAGS="xmicro,frontend" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
xmicro -env=DEBUG \
-port=8000 \
-role=shard1 

# start node4
node="${image}-node2-replica"
docker run -d -p 8004:8000 \
-h "$node" \
--name "$node" \
--network "$network" \
--restart unless-stopped \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_NAME="$node" \
-e SERVICE_TAGS="xmicro,frontend" \
-e SERVICE_CHECK_HTTP="/ping" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image \
xmicro -env=DEBUG \
-port=8000 \
-role=shard2
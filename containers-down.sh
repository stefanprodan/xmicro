#!/bin/bash
set -e

image="xmicro"

docker rm -f $(docker ps -a -q -f "ancestor=$image")

docker rmi -f $image

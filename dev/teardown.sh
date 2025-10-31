#!/bin/bash -ex

kind delete cluster
docker stop registry
docker rm registry

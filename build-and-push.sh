#!/bin/bash

docker build -t gitlab-bcds.udg.edu:5050/sergivb01/skeduler/server -f Dockerfile.server .
docker build -t gitlab-bcds.udg.edu:5050/sergivb01/skeduler/worker -f Dockerfile.worker .
docker build -t gitlab-bcds.udg.edu:5050/sergivb01/skeduler/database -f Dockerfile.database .

docker push gitlab-bcds.udg.edu:5050/sergivb01/skeduler/server
docker push gitlab-bcds.udg.edu:5050/sergivb01/skeduler/worker
docker push gitlab-bcds.udg.edu:5050/sergivb01/skeduler/database

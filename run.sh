#!/bin/bash +x
docker build . -t spotifysync:latest
docker run -p 8999:8999 -v "$(PWD)/config.json":/app/config.json spotifysync:latest

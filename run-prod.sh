#!/bin/bash +x
docker build . -t spotifysync:latest
docker stop spotifysync || true && docker rm spotifysync || true # stop the old container
docker run -p 80:80 -p 443:443 -v "$(pwd)/config.json":/app/config.json -v "$(pwd)/certs/":/app/certs/ -d --name spotifysync spotifysync:latest

#!/bin/sh

buildImage() {
    docker build -f build/Dockerfile -t fluxracine/webchat:$1 .
}

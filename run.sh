#!/bin/sh

set -exu pipefail

sudo docker build . -t localhost/ldap-proxy

sudo docker run -d \
    --restart=always \
    -p"10.83.1.1:38980:38980" \
    -e"LISTEN_ADDRESS=0.0.0.0:38980" -e"LDAP_URL=ldap://ldap.kth.se:389" \
    --name ldap-proxy \
    localhost/ldap-proxy

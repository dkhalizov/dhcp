#!/bin/bash

# Set up variables
NETWORK_NAME="dhcp-test-network"
SERVER_NAME="dhcp-server"
CLIENT_NAME="dhcp-client"
SERVER_IMAGE="dhcp"

# Create Docker network
echo "Creating Docker network..."
docker network create --subnet=172.20.0.0/16 $NETWORK_NAME

# Run DHCP server
echo "Starting DHCP server..."
docker run -d --name $SERVER_NAME --network $NETWORK_NAME --ip 172.20.0.2 --cap-add=NET_ADMIN -p 67:67/udp $SERVER_IMAGE

# Wait for server to start
echo "Waiting for server to start..."
sleep 5

# Run DHCP client
echo "Starting DHCP client..."
docker run -it --name $CLIENT_NAME --network $NETWORK_NAME \
  --cap-add=NET_ADMIN alpine /bin/sh -c '
    apk add --no-cache dhclient
    echo "Running DHCP client..."
    dhclient -v eth0
    echo "IP Address assigned:"
    ip addr show eth0
    echo "Testing connectivity..."
    ping -c 4 172.20.0.2
  '

# Clean up
echo "Cleaning up..."
docker stop $SERVER_NAME $CLIENT_NAME
docker rm $SERVER_NAME $CLIENT_NAME
docker network rm $NETWORK_NAME

echo "Test completed."
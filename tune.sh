#!/bin/bash

# allow testing with buffers up to 64MB 
sudo sysctl -w net.core.rmem_max=67108864 
sudo sysctl -w net.core.wmem_max=67108864 
# increase Linux autotuning TCP buffer limit to 32MB
sudo sysctl -w net.ipv4.tcp_rmem=33554432
sudo sysctl -w net.ipv4.tcp_wmem=33554432
# recommended default congestion control is htcp 
sudo sysctl -w net.ipv4.tcp_congestion_control=htcp

sudo tcset veth0 --delay 60ms --rate 100mbps --change
sudo tcset veth2 --delay 60ms --rate 100mbps --change
sudo ip netns exec ns2 tcset veth1 --delay 60ms --rate 100mbps --change
sudo ip netns exec ns2 tcset veth3 --delay 60ms --rate 100mbps --change
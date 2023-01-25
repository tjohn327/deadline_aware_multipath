#!/bin/bash

sudo ip netns add ns2

sudo sysctl -w net.mptcp.enabled=1
sudo ip netns exec ns2 sysctl -w net.mptcp.enabled=1

sudo ip link add veth0 type veth peer name veth1 netns ns2
sudo ip link add veth2 type veth peer name veth3 netns ns2
sudo ip addr add 10.1.0.1/24 dev veth0
sudo ip -n ns2 addr add 10.1.0.2/24 dev veth1
sudo ip addr add 10.3.0.1/24 dev veth2
sudo ip -n ns2 addr add 10.3.0.2/24 dev veth3

sudo ip link set veth0 up
sudo ip netns exec ns2 ip link set veth1 up
sudo ip link set veth2 up
sudo ip netns exec ns2 ip link set veth3 up

sudo ip rule add from 10.1.0.1 table 1
sudo ip rule add from 10.3.0.1 table 2
sudo ip route add 10.1.0.0/24 dev veth0 scope link table 1
sudo ip route add 10.3.0.0/24 dev veth2 scope link table 2

sudo ip netns exec ns2 ip rule add from 10.1.0.2 table 1
sudo ip netns exec ns2 ip rule add from 10.3.0.2 table 2
sudo ip netns exec ns2 ip route add 10.1.0.0/24 dev veth1 scope link table 1
sudo ip netns exec ns2 ip route add 10.3.0.0/24 dev veth3 scope link table 2



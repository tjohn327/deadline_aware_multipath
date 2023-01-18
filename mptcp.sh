#!/bin/bash

sudo ip rule add from 10.1.0.1 table 1
sudo ip rule add from 10.3.0.1 table 2
sudo ip route add 10.1.0.0/24 dev veth0 scope link table 1
sudo ip route add 10.3.0.0/24 dev veth2 scope link table 2

sudo ip netns exec ns2 ip rule add from 10.1.0.2 table 1
sudo ip netns exec ns2 ip rule add from 10.3.0.2 table 2
sudo ip netns exec ns2 ip route add 10.1.0.0/24 dev veth1 scope link table 1
sudo ip netns exec ns2 ip route add 10.3.0.0/24 dev veth3 scope link table 2

sudo ip mptcp limits set subflow 2
sudo ip mptcp limits set add_addr_accepted 2
sudo ip netns exec ns2 ip mptcp limits set subflow 2
# sudo ip netns exec ns2 ip mptcp limits set add_addr_accepted 2
sudo ip netns exec ns2 ip mptcp endpoint add 10.3.0.2 dev veth1 signal

# sudo ip mptcp limits set subflow 2 add_addr_accepted 2
sudo ip mptcp endpoint add 10.3.0.1 dev veth0 signal


# sudo ip netns exec ns2 ip mptcp limits set subflow 2 add_addr_accepted 2
# sudo ip netns exec ns2 ip mptcp endpoint add 10.3.0.2 dev veth1 subflow
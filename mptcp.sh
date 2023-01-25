#!/bin/bash



sudo ip mptcp endpoint flush
sudo ip netns exec ns2 ip mptcp endpoint flush

sudo ip mptcp limits set subflow 2 add_addr_accepted 2
sudo ip netns exec ns2 ip mptcp limits set subflow 2 add_addr_accepted 2

sudo ip mptcp endpoint add 10.3.0.1 dev veth0 signal
sudo ip netns exec ns2 ip mptcp endpoint add 10.3.0.2 dev veth1 signal

# sudo ip mptcp limits set subflow 2
# sudo ip mptcp limits set add_addr_accepted 2
# sudo ip mptcp endpoint add 10.3.0.2 dev veth2 id 1 subflow
# sudo ip netns exec ns2 ip mptcp limits set subflow 2
# sudo ip netns exec ns2 ip mptcp limits set add_addr_accepted 2


# sudo ip mptcp limits set subflow 2 add_addr_accepted 2
# sudo ip mptcp endpoint add 10.3.0.1 dev veth0 signal
# sudo ip netns exec ns2 ip mptcp endpoint add 10.3.0.2 dev veth1 signal


# sudo ip netns exec ns2 ip mptcp limits set subflow 2 add_addr_accepted 2
# sudo ip netns exec ns2 ip mptcp endpoint add 10.3.0.2 dev veth1 subflow
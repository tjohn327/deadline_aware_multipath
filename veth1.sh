
sudo ip netns add ns2
sudo ip link add veth0 type veth peer name veth1 netns ns2
sudo ip link set veth0 up
sudo ip netns exec ns2 ip link set veth1 up
sudo ip addr add 10.1.0.1/24 dev veth0
sudo ip netns exec ns2 ip addr add 10.1.0.2/24 dev veth1
sudo ip link add veth2 type veth peer name veth3 netns ns2
sudo ip link set veth2 up
sudo ip netns exec ns2 ip link set veth3 up
sudo ip addr add 10.3.0.1/24 dev veth2
sudo ip netns exec ns2 ip addr add 10.3.0.2/24 dev veth3

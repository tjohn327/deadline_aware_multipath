#!/bin/bash

go build

# sudo tcset veth0 --delay 60ms --rate 100mbps --change
# sudo tcset veth2 --delay 60ms --rate 100mbps --change
# sudo ip netns exec ns2 tcset veth1 --delay 60ms --rate 100mbps --change
# sudo ip netns exec ns2 tcset veth3 --delay 60ms --rate 100mbps --change

# cd /home/ubuntu/code/DARM/intial_eval/deadline_aware_multipath
# sudo mptcpize run ./deadline_aware_multipath -f /media/sf_SharedUbuntu/combined4.csv -t 50 -m 3 -d 20 &

sudo ./deadline_aware_multipath -f /home/ubuntu/code/shared/combined6 -t 50 -m 3 -d 20 
# sudo ./deadline_aware_multipath -f /code/shared/mptcp -t 50 -m 4 -d 20 
# sudo mptcpize run ./deadline_aware_multipath -f /home/scion/code/shared/mptcp -t 50 -m 4 -d 20 
# sudo mptcpize run ./deadline_aware_multipath -f /home/ubuntu/code/shared/mptcp -t 50 -m 4 -d 20 




# cd ~



# declare -a loss=("0.01%" "0.1%" "0.5%" "1%" "2%" "5%" "7%" "10%" "0%")
# length=${#loss[@]}
# for (( j=0; j<${length}; j++ ));
# do
#   printf "Current index %d with value %s\n" $j "${loss[$j]}" 
#   sudo tcset veth0 --delay 60ms --rate 100mbps --change --loss "${loss[$j]}"
#   sleep 5
# done
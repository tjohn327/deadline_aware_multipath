#!/bin/bash
# cd /home/ubuntu/code/DARM/intial_eval/deadline_aware_multipath
sudo ./deadline_aware_multipath -f /media/sf_SharedUbuntu/combined3.csv -t 50 -m 3 &
# cd ~

declare -a loss=("0.01%" "0.1%" "0.5%" "1%" "2%" "5%" "7%" "10%" "0%")
length=${#loss[@]}
for (( j=0; j<${length}; j++ ));
do
  printf "Current index %d with value %s\n" $j "${loss[$j]}"
  sudo tcset veth2 --delay 100ms --rate 100mbps --change --loss "${loss[$j]}"
  sleep 5
done
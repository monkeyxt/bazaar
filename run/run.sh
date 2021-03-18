#!/bin/bash

# Steps to test this script with your own AWS instances:
# ------------------------------------------------------
#
# First, comment out the line HOSTS="...
# Next, add your own ec2 public IPs.
# Example:
#   HOSTS=("54.162.114.73" "52.87.70.136")
#
# Comment out the KEYS="..." line right under it, and set that to the key file
# Example:
#   KEY=~/.aws/677kp.pem
#
# Now you'll need to modify the username.
# Example:
#   USERNAME=ec2-user
#
# Make sure that your inbound rules on AWS are set to all traffic (-1) for all
# IPs (0.0.0.0/0), or at least the default security group. This just opens up
# every port.
# Finally, run this script and see what happens!
# Remember to pass the config file you wish to use as a parameter to this script
# Example: 'bash run.sh config.yml'

# read a list of hosts
USERNAME=ec2-user
HOSTS=("54.172.102.164" "54.144.240.179" "34.202.157.206" "54.197.206.230" "100.24.244.33" "3.93.163.99")
KEY=lab1.pem
# USERNAME=ubuntu
# HOSTS=("54.225.31.11" "107.20.63.186")
# KEY=~/.aws/677kp.pem

# scp repo & install dependencies
for HOSTNAME in ${HOSTS[@]}; do
  echo "Current Server: "$HOSTNAME
  ssh -i $KEY -l ${USERNAME} ${HOSTNAME} "mkdir bazaar"
  scp -i $KEY -r ../../bazaar/* $USERNAME@$HOSTNAME:~/bazaar

  # snap is down (https://status.snapcraft.io/)
  # ssh -i $KEY -l ${USERNAME} ${HOSTNAME} "sudo snap install go --classic"

  # until snap starts working we'll use another source
  ssh -i $KEY -l $USERNAME $HOSTNAME "wget -q -N https://golang.org/dl/go1.16.2.linux-amd64.tar.gz"
  ssh -i $KEY -l $USERNAME $HOSTNAME "sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.16.2.linux-amd64.tar.gz"
  ssh -i $KEY -l $USERNAME $HOSTNAME "echo \"export PATH=\$PATH:/usr/local/go/bin\" > ~/.bash_profile"
done

echo "getting generatenodes dependencies..."
# generate node files from config
go get -v ../...

echo "building generatenodes..."
go build ../cmd/generatenodes
node_config=$1

node_folder=$(grep -A0 'outputDir:' $1 | awk '{ print $2}')
echo $node_folder


# Make the directory
mkdir -p $node_folder

hoststring=""
# loop through all of the hosts so we can create a long command
for host in ${HOSTS[@]}
do
    hoststring=$hoststring" --host $host"
done

echo "generating node configs, hoststring is $hoststring"
# Generate the .yml files in the directory using the config
./generatenodes --config $node_config $hoststring

# check if dir exists and cat everything if so
# if command -v bat &> /dev/null
# then
#     bat $node_folder/*.yml
# else
#     cat $node_folder/*.yml
# fi

# scp the node files into each of the server's ~/bazaar/nodes directory
# then compile the code on each server. Also populate the list of hosts.
yamls=`ls "$node_folder"/*.yml`
hostlist=()
for file in $yamls
do
    HOSTNAME=$(grep -A0 'nodeip' $file | tr -d '"' | awk '{ print $2}')
    hostlist+=($HOSTNAME)
    echo "Target Server: "$HOSTNAME
    ssh -i $KEY -l ${USERNAME} ${HOSTNAME} "mkdir ~/bazaar/nodes"
    scp -i $KEY $file ${USERNAME}@${HOSTNAME}:~/bazaar/nodes
    ssh -i $KEY -l ${USERNAME} ${HOSTNAME} "source ~/.bash_profile; cd ~/bazaar; go build"
done

# ssh into the servers and run nodes on each one
pids=()
ctr=0
for file in $yamls
do
  # this runs bazaar, gets the pid (on the host) and adds it to the list
  echo "getting file $(basename $file)"
  pids+=($(ssh -i $KEY -l ${USERNAME} ${hostlist[$ctr]} "~/bazaar/bazaar --config ~/bazaar/nodes/$(basename $file) &> ~/bazaarlog$ctr.txt & echo \$!"))
  ctr=$((ctr+1))
done

sleep 30s

# Create trap to kill all bazaar processes when script is stopped.
# go through host list and according pids
function kill_bazaars() {
    pidcounter=0
    echo "Killing bazaar processes..."

    # kill pids on each server
    for host in ${hostlist[*]}
    do
      echo "Logging on to $host to kill bazaar node with pid ${pids[$pidcounter]}"
      ssh -i $KEY -l ${USERNAME} $host "kill ${pids[$pidcounter]}"
      pidcounter=$((pidcounter+1))
    done
}
trap kill_bazaars INT TERM EXIT

# Remove the directory with the node configs
rm -rf $node_folder
rm generatenodes
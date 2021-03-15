#!/bin/bash

# read a list of hosts
USERNAME=ubuntu
HOSTS="3.89.254.207 54.235.60.16"

# scp repo & install dependencies
for HOSTNAME in ${HOSTS}; do
  echo "Current Server: "$HOSTNAME
  scp -i lab1.pem -r ../../bazaar/* ubuntu@$HOSTNAME:~/bazaar
  ssh -i lab1.pem -l ${USERNAME} ${HOSTNAME} "sudo snap install go --classic"
done

# generate node files from config
go build ../cmd/generatenodes
node_config=$1

node_folder=$(grep -A0 'outputDir:' $1 | awk '{ print $2}')
echo $node_folder

# Make the directory
mkdir -p $node_folder

# Generate the .yml files in the directory using the config
./generatenodes --config $node_config

# check if dir exists and cat everything if so
if command -v bat &> /dev/null
then
    bat $node_folder/*.yml
else
    cat $node_folder/*.yml
fi

# scp the node files into each of the server's ~/bazaar/nodes directory
# then compile the code on each server
yamls=`ls "$node_folder"/*.yml`
for file in $yamls
do
    HOSTNAME=$(grep -A0 'nodeip' $file | tr -d '"' | awk '{ print $2}')
    echo "Target Server: "$HOSTNAME
    ssh -i lab1.pem -l ${USERNAME} ${HOSTNAME} "mkdir ~/bazaar/nodes"
    scp -i lab1.pem $file ${USERNAME}@${HOSTNAME}:~/bazaar/nodes
    ssh -i lab1.pem -l ${USERNAME} ${HOSTNAME} "cd ~/bazaar; go build"
done

# ssh into the servers and run nodes on each one



# kill pids on each server



# Remove the directory with the node configs
rm -rf $node_folder
rm generatenodes
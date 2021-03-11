#!/bin/bash
# runtest.sh
# Takes 1 argument that is a path to a config yaml file


# Parse argument to get the config yml file
# node_config=$1

# Retrieve the node directory from the config file 
# node_folder=$(grep -A0 'outputDir:' $1 | awk '{ print $2}')
# echo $node_folder

# Make the directory 
# mkdir -p $node_folder

# Generate the .yml files in the directory using the config
# generate_nodes --config $node_config
(cd ../ && go build)
(cd ../ && cp bazaar test6)

pids=()

# Create trap to kill all bazaar processes when script is stopped
function kill_bazaars() {
    echo "Killing bazaar processes..."
    for pid in ${pids[*]}
    do
        kill -INT $pid
    done
    rm -rf $node_folder
}
trap kill_bazaars INT TERM EXIT

# Run bazaar for each of the .yml files
yamls=*.yml
for file in $yamls
do
    ../bazaar --config $file & pids+=( $! )
done

# Wait for all processes to finish
for pid in ${pids[*]}
do
    wait $pid
done

# Remove the directory with the node configs
# rm -rf $node_folder
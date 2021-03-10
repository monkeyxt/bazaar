#!/bin/bash
# Test case 5: 1 buyer, 3 sellers, 2 hops aparts.

# NOTE: this file is not yet functional, as the executable for generating the node .yml files is not yet complete

# Generate a folder for the .yml files for the nodes
node_folder="test5_nodes"
mkdir -p $node_folder

touch "$node_folder"/node1.yml
touch "$node_folder"/node2.yml

# Generate the .yml files in the folder using the config
# generate_nodes --config test5config.yml

pids=()

# Create trap to kill all bazaar processes when script is stopped
function kill_bazaars() {
    echo "Killing bazaar processes..."
    for pid in ${pids[*]}
    do
        kill -INT $pid
    done
}
trap kill_bazaars INT TERM EXIT

# Run bazaar for each of the .yml files
yamls="$node_folder"/*.yml
for f in $yamls
do
    ../bazaar $f & pids+=( $! )
done

# Wait for all processes to finish
for pid in ${pids[*]}
do
    wait $pid
done


# Steps
# 1. Run generateNodes with the test yml file (make folder before this)
# 2. Check in specified folder for node ymls
# 3. Run bazaar on each of ymls
# 4. Make function to kill processes (include cleanup)
# 5. Remove folder and kill processes
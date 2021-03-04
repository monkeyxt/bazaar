#!/bin/bash
# Test case 3: Assign random roles

# Generate random roles
role[0]="buyer"
role[1]="seller"
rand1=$[$RANDOM % 2]
rand2=$[$RANDOM % 2]
# echo "================================"
# echo $rand1
# echo $rand2
# echo "================================"

# YAML generation for node 1
export peerid="0"
export peer_port="10000"
export role="${role[$rand1]}"
export salt_amount="1"
export salt_unlimited="true"
export boars_amount="0"
export boars_unlimited="false"
export fish_amount="0"
export fish_unlimited="false"

export maxpeers="1"
export maxhops="1"
export nodeid="1"
export nodeport="10001"

rm -f node1.yml temp.yml
( echo "cat <<EOF >node1.yml";
  cat template.yml;
  echo "EOF";
) >temp.yml
. temp.yml
cat node1.yml

## YAML generation for node 2
export peerid="1"
export peer_port="10001"
export role="${role[$rand2]}"
export salt_amount="1"
export salt_unlimited="true"
export boars_amount="0"
export boars_unlimited="false"
export fish_amount="0"
export fish_unlimited="false"

export maxpeers="1"
export maxhops="1"
export nodeid="0"
export nodeport="10000"

rm -f node2.yml temp.yml
( echo "cat <<EOF >node2.yml";
  cat template.yml;
  echo "EOF";
) >temp.yml
. temp.yml
cat node2.yml

## Cleanup
rm temp.yml
rm -rf node1 node2

## Compile the go code and move the binary to each folder
mkdir node1 && mkdir node2
(cd ../ && go build)
(cd ../ && cp bazaar test/node1 && cp bazaar test/node2)

## Move the config files into each folder
mv node1.yml node1/bazaar.yml
mv node2.yml node2/bazaar.yml

function kill_bazaars() {
  echo "Killing processes..."
  kill -INT $PID_BAZAAR_ONE $PID_BAZAAR_TWO
}
trap kill_bazaars INT TERM EXIT

## Run both nodes
cd node1
./bazaar & PID_BAZAAR_ONE=$!
cd ../node2
./bazaar & PID_BAZAAR_TWO=$!

wait $PID_BAZAAR_ONE $PID_BAZAAR_TWO
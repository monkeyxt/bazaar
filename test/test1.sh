# Test case 1: Assign one peer to be a buyer of fish and another to 
# be a seller of fish. Ensure that all fish is sold and restocked forever.

# YAML generation for node 1
export peerid="0"
export peer_port="10000"
export role="buyer"
export salt_amount="0"
export salt_unlimited="false"
export boars_amount="0"
export boars_unlimited="false"
export fish_amount="1"
export fish_unlimited="false"

export maxpeers="1"
export maxhops="10"
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
export role="seller"
export salt_amount="0"
export salt_unlimited="false"
export boars_amount="10"
export boars_unlimited="false"
export fish_amount="10"
export fish_unlimited="true"

export maxpeers="1"
export maxhops="10"
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

## Run both nodes
node1/bazaar &
node2/bazaar &
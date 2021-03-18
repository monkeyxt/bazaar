#!/bin/bash

# Be sure to change the USERNAME, HOSTS, and KEY values in run.sh to match your desired deployment
# Note that due to the time it takes to copy files to AWS this will take longer than local tests

cd ../run
echo "Running all test scripts remotely..."

echo "Running test 1..."
bash run.sh ../test/test1config.yml

echo "Running test 2..."
bash run.sh ../test/test2config.yml

echo "Running test 3..."
bash run.sh ../test/test3config.yml

echo "Running test 4..."
bash run.sh ../test/test4config.yml

echo "Running test 5..."
bash run.sh ../test/test5config.yml

echo "Running test 6..."
bash run.sh ../test/test6config.yml

echo "Running test 7..."
bash run.sh ../test/test7config.yml

echo "Running test 8..."
bash run.sh ../test/test8config.yml

echo "Running test 9..."
bash run.sh ../test/test9config.yml

echo "Running test 10..."
bash run.sh ../test/test10config.yml

echo "Running test 11..."
bash run.sh ../test/test11config.yml
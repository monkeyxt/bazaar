#!/bin/bash
echo "Running all test scripts for 30 seconds each..."

echo "Running test 1..."
timeout 30s bash runtest.sh test1config.yml

echo "Running test 2..."
timeout 30s bash runtest.sh test2config.yml

echo "Running test 3..."
timeout 30s bash runtest.sh test3config.yml

echo "Running test 4..."
timeout 30s bash runtest.sh test4config.yml

echo "Running test 5..."
timeout 30s bash runtest.sh test5config.yml

echo "Running test 6..."
timeout 30s bash runtest.sh test6config.yml

echo "Running test 7..."
timeout 30s bash runtest.sh test7config.yml

echo "Running test 8..."
timeout 30s bash runtest.sh test8config.yml

echo "Running test 9..."
timeout 30s bash runtest.sh test9config.yml

echo "Running test 10..."
timeout 30s bash runtest.sh test10config.yml

echo "Running test 11..."
timeout 30s bash runtest.sh test11config.yml
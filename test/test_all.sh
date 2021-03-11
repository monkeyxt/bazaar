#!/bin/bash

echo "Running test 1..."
timeout 20s bash runtest.sh test1config.yml

echo "Running test 2..."
timeout 20s bash runtest.sh test2config.yml

echo "Running test 3..."
timeout 20s bash runtest.sh test3config.yml

echo "Running test 4..."
timeout 20s bash runtest.sh test4config.yml

echo "Running test 5..."
timeout 20s bash runtest.sh test5config.yml

echo "Running test 6..."
timeout 20s bash runtest.sh test6config.yml

bash test_cleanup.sh

#!/bin/bash

echo "Running test 1..."
timeout 20s bash test1.sh

echo "Running test 2..."
timeout 20s bash test2.sh

echo "Running test 3..."
timeout 20s bash test3.sh

echo "Running test 4..."
timeout 20s bash test4.sh

bash test_cleanup.sh

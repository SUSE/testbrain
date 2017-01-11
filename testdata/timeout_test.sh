#!/bin/bash

echo Timeout = $TESTBRAIN_TIMEOUT
while true
do
    echo "Stuck in an infinite loop!"
    sleep 0.4
done
echo "Goodbye World!"

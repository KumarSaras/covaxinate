#!/bin/sh
set -o xtrace
python3 statelistgenerator.py
for i in {1..37}
do
    python3 districtlistgenerator.py $i
done
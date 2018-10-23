#!/bin/bash
# This script ssh's to instances listed in "eye_addresses.txt"
# and restart's eye processes

EYE_ADDRESSES_FILE_PATH="eye_addresses.txt"
PEM_KEY_PATH="/Users/anov/Desktop/aws_keys/TokyoTest.pem"
USER="ubuntu"

addresses=()
while read -r line
do
	addresses+=($line)
done < $EYE_ADDRESSES_FILE_PATH

echo "Initiated restart for eyes:"
for each in ${addresses[@]}
do
	echo $each
done

restart_eye () {
	echo "Restarting "$addr"..."
        ssh -i $PEM_KEY_PATH $USER@$addr 'bash -s' < restart_eye_local.sh
	echo "Finished"
}

for addr in ${addresses[@]}
do
        restart_eye $addr
done
echo "All done"

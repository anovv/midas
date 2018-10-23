#!/bin/bash
# This script ssh's to instances listed in "eye_addresses.txt"
# pull git, rebuilds binaries and restart's eye processes

EYE_ADDRESSES_FILE_PATH="eye_addresses.txt"
PEM_KEY_PATH="/Users/anov/Desktop/aws_keys/TokyoTest.pem"
USER="ubuntu"

addresses=()
while read -r line
do
        addresses+=($line)
done < $EYE_ADDRESSES_FILE_PATH

echo "Initiated update and restart for eyes:"
for each in ${addresses[@]}
do
        echo $each
done

update_and_restart_eye () {
        echo "Updating and restarting "$addr"..."
        ssh -i $PEM_KEY_PATH $USER@$addr 'bash -s' < update_and_restart_eye_local.sh
        echo "Finished"
}

for addr in ${addresses[@]}
do
        update_and_restart_eye $addr
done
echo "All done"


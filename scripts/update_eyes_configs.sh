#!/bin/bash
# This script scp to instances listed in "eye_addresses.txt"
# and uploads eye_config.json

EYE_ADDRESSES_FILE_PATH="eye_addresses.txt"
PEM_KEY_PATH="/Users/anov/Desktop/aws_keys/TokyoTest.pem"
USER="ubuntu"
CONFIG_PATH=/Users/anov/go/projects/bin/eye_config.json

addresses=()
while read -r line
do
        addresses+=($line)
done < $EYE_ADDRESSES_FILE_PATH

echo "Updating configs for eyes:"
for each in ${addresses[@]}
do
        echo $each
done

update_eye_config () {
        echo "Updating config for "$addr"..."
        scp -i $PEM_KEY_PATH $CONFIG_PATH $USER@$addr:/home/ubuntu/go/projects/bin/eye_config.json
        echo "Finished"
}

for addr in ${addresses[@]}
do
        update_eye_config $addr
done
echo "All done"


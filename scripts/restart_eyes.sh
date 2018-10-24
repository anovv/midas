#!/bin/bash
# This script ssh's to instances listed in "eye_addresses.txt"
# and restart's eye processes

EYE_ADDRESSES_FILE_PATH="eye_addresses.txt"
PEM_KEY_PATH="/Users/anov/Desktop/aws_keys/TokyoTest.pem"
USER="ubuntu"

offset=$1
if [ -z $offset ]; then
	offset=100000
fi

counter=0
addresses=()
while read -r line
do
	addresses+=($line)
done < $EYE_ADDRESSES_FILE_PATH

echo "Initiated restart for eyes:"
for each in ${addresses[@]}
do
	if ((counter < offset)); then
		echo $each
		((counter++))
	else
		break
	fi
done

restart_eye () {
	echo "Restarting "$addr"..."
        ssh -i $PEM_KEY_PATH $USER@$addr 'bash -s' < restart_eye_local.sh
	echo "Finished"
}

counter=0
for addr in ${addresses[@]}
do
	if ((counter < offset)); then
        	restart_eye $addr
		((counter++))
	else
		break
	fi
done
echo "All done"

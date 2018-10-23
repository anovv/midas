#!/bin/bash
# Script to run locally on the eye instance. Executed over ssh
cd ~/go/projects/bin
pid=$(pgrep eye_exec)
if [ ! -z $pid ]; then
	kill $pid
fi
nohup ./eye_exec > /dev/null 2>&1 &
sleep 1
exit

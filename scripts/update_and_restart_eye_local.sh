#!/bin/bash
# Script to run locally on the eye instance. Executed over ssh

# TODO on the eye for this to work:
# edit /etc/ssh/sshd_config, add PermitUserEnvironment yes
# edit ~/.ssh/environment, add GOPATH=/home/ubuntu/go/projects
# run  sudo /etc/init.d/ssh restart

cd ~/go/projects/src/midas
git pull
/usr/local/go/bin/go install midas/execs/eye_exec
cd ~/go/projects/bin
pid=$(pgrep eye_exec)
if [ ! -z $pid ]; then
        kill $pid
fi
nohup ./eye_exec > /dev/null 2>&1 &
sleep 1
exit


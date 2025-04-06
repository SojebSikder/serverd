# Description
Automate ubuntu server configuration


# For bash script
## Step 1: Copy script
scp -P 2223 setup_ubuntu_server.sh ubuntu@192.168.4.2:/tmp/

## Step 2: SSH and run
ssh -t -p 2223 ubuntu@192.168.4.2 'bash /tmp/setup_ubuntu_server.sh'
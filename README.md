# Description
Automate ubuntu server configuration

# Run
```
go run .
```
Run with another file
```
go run . -file=config.yml
```

# Usage
```yml
server:
  ssh:
    host: 192.168.4.2
    port: 2223
    user: ubuntu
    pass: root

  sudo_password: root
  ssl: true
  name: blog

  services:
    backend:
      type: nestjs
      domain: test.com
      port: 4000
    frontend:
      type: nextjs
      domain: test.com
      port: 3000

```

# For bash script
## Step 1: Copy script
scp -P 2223 setup_ubuntu_server.sh ubuntu@192.168.4.2:/tmp/

## Step 2: SSH and run
ssh -t -p 2223 ubuntu@192.168.4.2 'bash /tmp/setup_ubuntu_server.sh'
# Description

Automate ubuntu server configuration

# Run

```
go run . apply
```

Run with another file

```
go run . apply -file=config.yml
```

# Build
```
./build.sh
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
      repository: https://github.com/SojebSikder/nestjs-boilerplate.git
    frontend:
      type: nextjs
      domain: test.com
      repository: https://github.com/SojebSikder/nextjs-boilerplate.git
      port: 3000
```

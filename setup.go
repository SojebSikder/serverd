package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func printMessage(msg string) {
	fmt.Printf("\033[1;32m%s\033[0m\n", msg)
}

func prompt(message string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(message + ": ")
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func promptYesNo(message string) bool {
	answer := prompt(message + " (y/n)")
	return strings.ToLower(answer) == "y"
}

func gitCloneRepoWithToken(ssh *SSHClient) {
	printMessage("Cloning repo...")
	repo := prompt("Enter the GitHub repo (e.g., user/repo)")
	token := prompt("Enter GitHub token")
	ssh.runInteractive(fmt.Sprintf("git clone https://%s@github.com/%s.git .", token, repo))
}

func gitCloneRepo(ssh *SSHClient, repo string) {
	printMessage("Cloning repo...")
	ssh.runInteractive(fmt.Sprintf("git clone %s .", repo))
}

func setupFirewall(ssh *SSHClient) {
	printMessage("Setting up firewall...")
	// ssh.runInteractive(fmt.Sprintf("adduser %s", user))
	// ssh.runInteractive(fmt.Sprintf("usermod -aG sudo %s", user))
	ssh.runInteractive("ufw allow OpenSSH")
	ssh.runInteractive("ufw --force enable")
}

func setupNginx(ssh *SSHClient) {
	printMessage("Installing Nginx...")
	ssh.runInteractive("apt update")
	ssh.runInteractive("apt install -y nginx")
	ssh.runInteractive("ufw allow 'Nginx HTTP'")
	ssh.runInteractive("systemctl enable nginx")
	ssh.runInteractive("systemctl start nginx")
}

func setupNginxBlock(ssh *SSHClient, domain string, project string, service string, port string) {
	printMessage(fmt.Sprintf("Configuring Nginx for %s...", service))
	path := fmt.Sprintf("/var/www/%s/%s", project, service)
	confPath := fmt.Sprintf("/etc/nginx/sites-available/%s_%s", project, service)

	fmt.Println("check url")
	fmt.Println(confPath)

	block := fmt.Sprintf(`server {
	listen 80;
	listen [::]:80;
	server_name %s www.%s;

	location / {
		proxy_pass http://localhost:%s;
		proxy_http_version 1.1;
		proxy_set_header Upgrade $http_upgrade;
		proxy_set_header Connection 'upgrade';
		proxy_set_header Host $host;
		proxy_cache_bypass $http_upgrade;
	}

	location /public {
		root /var/www/%s/%s;
		add_header Access-Control-Allow-Origin *;
		add_header Access-Control-Allow-Methods 'GET, POST, OPTIONS';
		add_header Access-Control-Allow-Headers 'Content-Type, Authorization';
		try_files $uri $uri/ =404;
	}
}`, domain, domain, port, project, service)

	ssh.runInteractive(fmt.Sprintf("sudo mkdir -p %s", path))
	ssh.runInteractive(fmt.Sprintf(`echo '%s' | sudo tee %s`, block, confPath))
	ssh.runInteractive(fmt.Sprintf("sudo ln -s %s /etc/nginx/sites-enabled/", confPath))
	ssh.runInteractive("sudo nginx -t")
	ssh.runInteractive("sudo systemctl restart nginx")
}

func setupPostgres(ssh *SSHClient) {
	printMessage("Installing PostgreSQL...")
	ssh.runInteractive("apt update")
	ssh.runInteractive("apt install -y postgresql postgresql-contrib")
	ssh.runInteractive("systemctl start postgresql")
	ssh.runInteractive(`-u postgres psql -c "ALTER USER postgres PASSWORD 'root'"`)
}

func setupNode(ssh *SSHClient) {
	printMessage("Installing Node.js & PM2...")
	ssh.runInteractive("curl -sL https://deb.nodesource.com/setup_20.x | sudo -E bash -")
	ssh.runInteractive("apt install -y nodejs")
	ssh.runInteractive("npm install -g pm2 yarn")
}

func setupNestApp(ssh *SSHClient, project string, name string, repo string) {
	printMessage("Setting up NestJS app...")
	dir := fmt.Sprintf("/var/www/%s/%s", project, name)
	ssh.runInteractive(fmt.Sprintf("cd %s && rm -rf *", dir)) // clean if needed
	ssh.runInteractive(fmt.Sprintf("mkdir -p %s && cd %s", dir, dir))
	ssh.runInteractive(fmt.Sprintf("cd %s && ", dir)) // for consistency
	gitCloneRepo(ssh, repo)
	ssh.runInteractive(fmt.Sprintf("cd %s && yarn install && yarn build && pm2 start dist/src/main.js --name %s", dir, name))
}

func setupNextApp(ssh *SSHClient, project string, name string, repo string) {
	printMessage("Setting up NextJS app...")
	dir := fmt.Sprintf("/var/www/%s/%s", project, name)
	ssh.runInteractive(fmt.Sprintf("cd %s && rm -rf *", dir))
	ssh.runInteractive(fmt.Sprintf("mkdir -p %s && cd %s", dir, dir))
	gitCloneRepo(ssh, repo)
	ssh.runInteractive(fmt.Sprintf("cd %s && yarn install && yarn build && pm2 start 'yarn start' --name %s", dir, name))
}

func setupSSL(ssh *SSHClient, domain string) {
	printMessage("🔒 Setting up SSL...")
	ssh.runInteractive("snap install core && sudo snap refresh core")
	ssh.runInteractive("apt remove certbot -y || true")
	ssh.runInteractive("snap install --classic certbot")
	ssh.runInteractive("ln -s /snap/bin/certbot /usr/bin/certbot")
	ssh.runInteractive(fmt.Sprintf("certbot --nginx -d %s -d www.%s", domain, domain))
}

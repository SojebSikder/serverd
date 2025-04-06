package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

type Config struct {
	SSH struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
		User string `yaml:"user"`
		Pass string `yaml:"pass"`
	} `yaml:"ssh"`
	Server struct {
		ProjectName  string `yaml:"project_name"`
		SudoPassword string `yaml:"sudo_password"`
		Domain       string `yaml:"domain"`
		SSL          bool   `yaml:"ssl"`
	} `yaml:"server"`
}

type SSHClient struct {
	client *ssh.Client
}

var sudoPassword string

func connectSSH(user string, password string, host string, port string) (*SSHClient, error) {
	if port == "" {
		port = "22"
	}
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", host+":"+port, config)
	if err != nil {
		return nil, err
	}
	return &SSHClient{client: client}, nil
}

// func (s *SSHClient) run(cmd string) error {
// 	session, err := s.client.NewSession()
// 	if err != nil {
// 		return err
// 	}
// 	defer session.Close()
// 	session.Stdout = os.Stdout
// 	session.Stderr = os.Stderr
// 	fmt.Println("🔧 Running:", cmd)
// 	return session.Run(cmd)
// }

func (s *SSHClient) runWithSudo(cmd string, password string) error {
	session, err := s.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Set up stdin for password piping
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	fullCmd := fmt.Sprintf("sudo -S bash -c '%s'", cmd)
	fmt.Println("🔧 Running (sudo):", fullCmd)

	if err := session.Start(fullCmd); err != nil {
		return err
	}

	// Write the password followed by newline
	_, err = stdin.Write([]byte(password + "\n"))
	if err != nil {
		return err
	}

	return session.Wait()
}

func (s *SSHClient) runInteractive(cmd string) {
	// err := s.run(cmd)
	err := s.runWithSudo(cmd, sudoPassword)
	if err != nil {
		log.Fatalf("❌ SSH command failed: %v", err)
	}
}

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

func gitCloneRepo(ssh *SSHClient, repoType string) {
	printMessage("⚙️ Cloning repo...")
	repo := prompt("Enter the GitHub repo (e.g., user/repo)")
	token := prompt("Enter GitHub token")
	ssh.runInteractive(fmt.Sprintf("git clone https://%s@github.com/%s.git .", token, repo))
}

func setupFirewall(ssh *SSHClient) {
	printMessage("⚙️ Setting up firewall...")
	// ssh.runInteractive(fmt.Sprintf("adduser %s", user))
	// ssh.runInteractive(fmt.Sprintf("usermod -aG sudo %s", user))
	ssh.runInteractive("ufw allow OpenSSH")
	ssh.runInteractive("ufw --force enable")
}

func setupNginx(ssh *SSHClient) {
	printMessage("⚙️ Installing Nginx...")
	ssh.runInteractive("apt update")
	ssh.runInteractive("apt install -y nginx")
	ssh.runInteractive("ufw allow 'Nginx HTTP'")
	ssh.runInteractive("systemctl enable nginx")
	ssh.runInteractive("systemctl start nginx")
}

func setupNginxBlock(ssh *SSHClient, domain, project, service string, port string) {
	printMessage(fmt.Sprintf("⚙️ Configuring Nginx for %s...", service))
	path := fmt.Sprintf("/var/www/%s/%s", project, service)
	confPath := fmt.Sprintf("/etc/nginx/sites-available/%s_%s", project, service)

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
	printMessage("⚙️ Installing PostgreSQL...")
	ssh.runInteractive("apt update")
	ssh.runInteractive("apt install -y postgresql postgresql-contrib")
	ssh.runInteractive("systemctl start postgresql")
	ssh.runInteractive(`-u postgres psql -c "ALTER USER postgres PASSWORD 'root'"`)
}

func setupNode(ssh *SSHClient) {
	printMessage("⚙️ Installing Node.js & PM2...")
	ssh.runInteractive("curl -sL https://deb.nodesource.com/setup_20.x | sudo -E bash -")
	ssh.runInteractive("apt install -y nodejs")
	ssh.runInteractive("npm install -g pm2 yarn")
}

func setupNestApp(ssh *SSHClient, project string) {
	printMessage("⚙️ Setting up NestJS app...")
	dir := fmt.Sprintf("/var/www/%s/backend", project)
	ssh.runInteractive(fmt.Sprintf("cd %s && rm -rf *", dir)) // clean if needed
	ssh.runInteractive(fmt.Sprintf("mkdir -p %s && cd %s", dir, dir))
	ssh.runInteractive(fmt.Sprintf("cd %s && ", dir)) // for consistency
	gitCloneRepo(ssh, "backend")
	ssh.runInteractive(fmt.Sprintf("cd %s && yarn install && yarn build && pm2 start dist/src/main.js --name backend", dir))
}

func setupNextApp(ssh *SSHClient, project string) {
	printMessage("⚙️ Setting up NextJS app...")
	dir := fmt.Sprintf("/var/www/%s/frontend", project)
	ssh.runInteractive(fmt.Sprintf("cd %s && rm -rf *", dir))
	ssh.runInteractive(fmt.Sprintf("mkdir -p %s && cd %s", dir, dir))
	gitCloneRepo(ssh, "frontend")
	ssh.runInteractive(fmt.Sprintf("cd %s && yarn install && yarn build && pm2 start 'yarn start' --name frontend", dir))
}

func setupSSL(ssh *SSHClient, domain string) {
	printMessage("🔒 Setting up SSL...")
	ssh.runInteractive("snap install core && sudo snap refresh core")
	ssh.runInteractive("apt remove certbot -y || true")
	ssh.runInteractive("snap install --classic certbot")
	ssh.runInteractive("ln -s /snap/bin/certbot /usr/bin/certbot")
	ssh.runInteractive(fmt.Sprintf("certbot --nginx -d %s -d www.%s", domain, domain))
}

func main() {
	var configFile = "config.yml"
	// Read YAML file
	data, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	// Unmarshal into struct
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}
	// Access command-line arguments
	args := os.Args

	if len(args) > 1 {
		switch args[1] {
		case "apply":
			apply(config)
		case "version":
			fmt.Println("Version: 0.0.1")
		}

	} else {
		fmt.Println("No arguments passed.")
		return
	}
}

func apply(config Config) {
	host := config.SSH.Host
	port := config.SSH.Port
	user := config.SSH.User
	pass := config.SSH.Pass
	sudoPassword = config.Server.SudoPassword

	project := config.Server.ProjectName
	domain := config.Server.Domain
	needSSL := config.Server.SSL

	ssh, err := connectSSH(user, pass, host, port)
	if err != nil {
		log.Fatalf("SSH connection failed: %v", err)
	}
	defer ssh.client.Close()

	printMessage("🔑 Connected. Starting setup...")

	setupFirewall(ssh)
	setupNginx(ssh)
	setupNginxBlock(ssh, domain, project, "backend", "4000")
	setupNginxBlock(ssh, domain, project, "frontend", "3000")
	setupPostgres(ssh)
	setupNode(ssh)
	setupNestApp(ssh, project)
	setupNextApp(ssh, project)

	if needSSL {
		setupSSL(ssh, domain)
	}

	printMessage("✅ Server setup completed via SSH!")
}

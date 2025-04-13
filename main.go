package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

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

	// fullCmd := fmt.Sprintf(`sudo -S bash -c "%s"`, cmd)
	fullCmd := fmt.Sprintf("echo %s | sudo -S %s", password, cmd)
	fmt.Println("Running (sudo):", fullCmd)

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
		log.Fatalf("SSH command failed: %v", err)
	}
}

func main() {
	// Define flags
	var configFile string
	flag.StringVar(&configFile, "file", "config.yml", "Specify config file")

	// Parse the flags
	flag.Parse()

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
		case "test":
			test(config)
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

func test(config Config) {
	host := config.Server.SSH.Host
	port := config.Server.SSH.Port
	user := config.Server.SSH.User
	pass := config.Server.SSH.Pass
	sudoPassword = config.Server.SudoPassword

	ssh, err := connectSSH(user, pass, host, port)
	if err != nil {
		log.Fatalf("SSH connection failed: %v", err)
	}
	defer ssh.client.Close()

	ssh.runInteractive("ufw status")

	ssh.runInteractive(`-u postgres psql -c "ALTER USER postgres PASSWORD 'root'"`)
}

func apply(config Config) {
	host := config.Server.SSH.Host
	port := config.Server.SSH.Port
	user := config.Server.SSH.User
	pass := config.Server.SSH.Pass
	sudoPassword = config.Server.SudoPassword
	needSSL := config.Server.SSL
	project := config.Server.Name

	ssh, err := connectSSH(user, pass, host, port)
	if err != nil {
		log.Fatalf("SSH connection failed: %v", err)
	}
	defer ssh.client.Close()

	printMessage("Connected. Starting setup...")
	// global setup
	setupFirewall(ssh)
	setupNginx(ssh)
	setupPostgres(ssh)
	setupNode(ssh)

	// Setup services
	for name, svc := range config.Server.Services {
		appType := svc.Type
		domain := svc.Domain
		port := svc.Port
		repository := svc.Repository

		setupNginxBlock(ssh, domain, project, name, port)

		if appType == "nestjs" {
			setupNestApp(ssh, project, name, repository)
		}
		if appType == "nextjs" {
			setupNextApp(ssh, project, name, repository)
		}

		if needSSL {
			setupSSL(ssh, domain)
		}
	}

	printMessage("Server setup completed!")
}

package main

type Config struct {
	Server ServerConfig `yaml:"server"`
}
type ServerConfig struct {
	SSH          SSHConfig             `yaml:"ssh"`
	SudoPassword string                `yaml:"sudo_password"`
	SSL          bool                  `yaml:"ssl"`
	Name         string                `yaml:"name"`
	Services     map[string]ServiceApp `yaml:"services"`
}
type SSHConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type ServiceApp struct {
	Type   string `yaml:"type"`
	Domain string `yaml:"domain"`
	Port   string `yaml:"port"`
}

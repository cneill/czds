package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

// Config specifies auth info, URL info, etc.
type Config struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	AuthBaseURL string `json:"auth_base_url"`
	CZDSBaseURL string `json:"czds_base_url"`
	WorkingDir  string `json:"working_dir"`
}

// ParseConfig reads the contents of a config file and unmarshals it into a Config object
func ParseConfig(filename string) (*Config, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	if err := json.Unmarshal(contents, &c); err != nil {
		return nil, err
	}

	if c.Username == "" {
		fmt.Printf("Enter your ICANN username: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			un := strings.TrimSpace(scanner.Text())
			if un == "" {
				return nil, fmt.Errorf("you must provide username/password credentials")
			}
			c.Username = un
		}

	}

	if c.Password == "" {
		fmt.Println("Enter your ICANN password:")
		bytePass, err := terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			return nil, fmt.Errorf("you must provide username/password credentials")
		}
		pw := strings.TrimSpace(string(bytePass))
		if pw == "" {
			return nil, fmt.Errorf("you must provide username/password credentials")
		}
		c.Password = pw
	}
	return c, nil
}

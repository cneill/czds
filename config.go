package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	if c.Username == "" || c.Password == "" {
		return nil, fmt.Errorf("You must include username/password credentials")
	}

	return c, nil
}

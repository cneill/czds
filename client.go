package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

// Client handles making requests to ICANN
type Client struct {
	*http.Client
	Conf        *Config
	AccessToken string
}

// NewClient returns an initialized Client
func NewClient(conf *Config) *Client {
	return &Client{
		Client: http.DefaultClient,
		Conf:   conf,
	}
}

type authCreds struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken string `json:"accessToken"`
}

// Auth gets the authentication token with the provided credentials
func (c *Client) Auth() error {
	authURL := fmt.Sprintf("%s/api/authenticate", c.Conf.AuthBaseURL)
	creds := authCreds{c.Conf.Username, c.Conf.Password}
	body, err := json.Marshal(&creds)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Couldn't read response body: %v", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		ar := &authResponse{}
		err := json.Unmarshal(respBody, &ar)
		if err != nil {
			return fmt.Errorf("Invalid response returned: %s", respBody)
		}
		c.AccessToken = ar.AccessToken

	case http.StatusNotFound:
		return fmt.Errorf("Invalid URL: %s", req.URL.String())

	case http.StatusUnauthorized:
		return fmt.Errorf("Invalid credentials: %s", respBody)

	case http.StatusInternalServerError:
		return fmt.Errorf("Internal server error: %s", respBody)

	default:
		return fmt.Errorf("Unknown error")
	}

	return nil
}

// Get overrides the default http.Client Get to add auth info
func (c *Client) Get(URL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	return c.Client.Do(req)
}

// GetZoneLinks returns the list of URLs pointing to the zone files
func (c *Client) GetZoneLinks() ([]string, error) {
	var linkList = []string{}
	linksURL := fmt.Sprintf("%s/czds/downloads/links", c.Conf.CZDSBaseURL)
	resp, err := c.Get(linksURL)
	if err != nil {
		return nil, fmt.Errorf("error getting zone links: %v", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Couldn't read response body: %v", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		if err := json.Unmarshal(respBody, &linkList); err != nil {
			return nil, fmt.Errorf("Couldn't unmarshal links in zone list")
		}

	case http.StatusUnauthorized:
		fmt.Println("Access token expired; reauthenticating")
		c.Auth()
		return c.GetZoneLinks()

	default:
		return nil, fmt.Errorf("unknown status error: (%d): %s", resp.StatusCode, respBody)
	}

	return linkList, nil
}

// DownloadZoneFiles takes a list of ZoneFiles and downloads them one by one
func (c *Client) DownloadZoneFiles(URLs []string) error {
	for _, u := range URLs {
		if err := c.DownloadZoneFile(u); err != nil {
			return err
		}
		time.Sleep(time.Second * 2)
	}

	return nil
}

// DownloadZoneFile downloads an individual zone file
func (c *Client) DownloadZoneFile(URL string) error {
	fmt.Printf("Downloading %s...\n", URL)
	resp, err := c.Get(URL)
	if err != nil {
		return err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Couldn't read response body: %v", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		u, err := url.Parse(URL)
		if err != nil {
			return fmt.Errorf("invalid URL")
		}
		_, file := path.Split(u.Path)
		path := fmt.Sprintf("%s/%s.gz", c.Conf.WorkingDir, file)
		fmt.Printf("\tWriting to file %s...\n", path)
		if err := ioutil.WriteFile(path, respBody, 0644); err != nil {
			return fmt.Errorf("failed to write to path %s: %v", path, err)
		}
	case http.StatusUnauthorized:
		fmt.Println("Access token expired; reauthenticating")
		c.Auth()
		return c.DownloadZoneFile(URL)
	case http.StatusNotFound:
		return fmt.Errorf("URL not found: %s", URL)
	default:
		return fmt.Errorf("unknown status error: (%d): %s", resp.StatusCode, respBody)

	}

	return nil
}

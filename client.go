package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cavaliercoder/grab"
	"github.com/dustin/go-humanize"
)

// Client handles making requests to ICANN
type Client struct {
	*http.Client
	Conf        *Config
	Verbose     bool
	AccessToken string
}

// NewClient returns an initialized Client
func NewClient(conf *Config, verbose bool) *Client {
	tr := http.Transport{DisableKeepAlives: true}
	cl := http.Client{Transport: &tr}
	return &Client{
		Client:  &cl,
		Verbose: verbose,
		Conf:    conf,
	}
}

type authCreds struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken string `json:"accessToken"`
}

type WriteCounter struct {
	Total   uint64
	verbose bool
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
		if c.Verbose {
			fmt.Printf("Couldn't read response body: %v", err)
		}
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

	req.Header.Add("User-Agent", "czds / v0.0.2")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Connection", "close") // prevent connect reset by peer error
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))
	req.Close = true

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
		if c.Verbose {
			fmt.Printf("Couldn't read response body: %v", err)
		}
	}

	switch resp.StatusCode {
	case http.StatusOK:
		if err := json.Unmarshal(respBody, &linkList); err != nil {
			return nil, fmt.Errorf("Couldn't unmarshal links in zone list")
		}

	case http.StatusUnauthorized:
		if c.Verbose {
			fmt.Println("Access token expired; reauthenticating")
		}
		c.Auth()
		return c.GetZoneLinks()

	default:
		return nil, fmt.Errorf("unknown status error: (%d): %s", resp.StatusCode, respBody)
	}

	return linkList, nil
}

// DownloadZoneFiles takes a list of ZoneFiles and downloads them one by one
func (c *Client) DownloadZoneFiles(URLs []string) error {
	if err := c.grabZoneFiles(URLs); err != nil {
		return err
	}
	return nil
}

func (c *Client) makeGrabRequest(url string) *grab.Request {
	req, err := grab.NewRequest("./tmp/", url)
	if err != nil {
		return nil
	}
	req.NoResume = true
	req.HTTPRequest.Header.Set("User-Agent", "czds / v0.0.1 https://github.com/cneill/czds")
	req.HTTPRequest.Header.Set("Content-Type", "application/json")
	req.HTTPRequest.Header.Set("Accept", "application/json")
	req.HTTPRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))
	return req
}

func (c *Client) grabZoneFiles(URLs []string) error {
	client := grab.NewClient()

	var urlfailed []string

	// create requests from command arguments
	reqs := make([]*grab.Request, 0)
	for _, url := range URLs {
		req := c.makeGrabRequest(url)
		reqs = append(reqs, req)
	}
	// start file downloads, 3 at a time
	if c.Verbose {
		fmt.Printf("Downloading %d files...\n", len(reqs))
	}
	respch := client.DoBatch(10, reqs...)

	// start a ticker to update progress every 200ms
	t := time.NewTicker(200 * time.Millisecond)

	// monitor downloads
	completed := 0
	inProgress := 0
	responses := make([]*grab.Response, 0)
	for {
		select {
		case resp := <-respch:
			if resp != nil {
				responses = append(responses, resp)
			}

		case <-t.C:
			// clear lines
			if inProgress > 0 {
				if c.Verbose {
					fmt.Printf("\033[%dA\033[K", inProgress)
				}
			}
			// update completed downloads
			for i, resp := range responses {
				if resp != nil && resp.IsComplete() {
					// print final result
					if resp.Err() != nil {
						fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", resp.Request.URL(), resp.Err())
						err := os.Remove(resp.Filename)
						if err != nil {
							fmt.Printf("%s remove Error!", resp.Filename)
						}
						var url string
						url = resp.Request.URL().String()
						urlfailed = append(urlfailed, url)
					} else {
						if c.Verbose {
							fmt.Printf("Finished %s (%d%%)\n", resp.Filename, int(100*resp.Progress()))
						}
						errfail := os.Rename(resp.Filename, strings.Replace(resp.Filename, "tmp", "files", -1))
						if errfail != nil {
							return fmt.Errorf("failed to write to path %s: %v", strings.Replace(resp.Filename, "tmp", "files", -1), errfail)
						}

					}

					// mark completed
					responses[i] = nil
					completed++
				}
			}

			// update downloads in progress
			inProgress = 0
			for _, resp := range responses {
				if resp != nil {
					inProgress++
					if c.Verbose {
						fmt.Printf("Downloading %s %d%%\n", resp.Filename, int(100*resp.Progress()))
					}
				}
			}
		}

		if completed >= len(reqs) {
			goto Done
		}

	}

Done:
	t.Stop()
	c.grabZoneFiles(urlfailed)

	if c.Verbose {
		fmt.Printf("%d files successfully downloaded.\n", len(reqs))
	}
	return nil
}

// DownloadZoneFile downloads an individual zone file
func (c *Client) DownloadZoneFile(URL string) error {
	var filename string

	u, err := url.Parse(URL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	_, file := path.Split(u.Path)
	filename = fmt.Sprintf("%s.gz", file)

	if c.Verbose {
		fmt.Printf("Downloading %s...\n", URL)
	}
	resp, err := c.Get(URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err != nil {
		if c.Verbose {
			fmt.Printf("Couldn't read response body: %v", err)
		}
	}

	switch resp.StatusCode {
	case http.StatusOK:
		break
	case http.StatusUnauthorized:
		if c.Verbose {
			fmt.Println("Access token expired; reauthenticating")
		}
		c.Auth()
		return c.DownloadZoneFile(URL)
	case http.StatusNotFound:
		return fmt.Errorf("URL not found: %s", URL)
	default:
		return fmt.Errorf("unknown status error: (%d): %s", resp.StatusCode, resp.Body)

	}

	cd := resp.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		if c.Verbose {
			fmt.Printf("error parsing Content-Disposition header: %v", err)
		}
	} else {
		if n, ok := params["filename"]; ok {
			filename = n
		}

	}

	path := fmt.Sprintf("%s/%s", c.Conf.WorkingDir, filename)

	out, err := os.Create(filename + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()

	counter := &WriteCounter{}
	counter.verbose = c.Verbose
	var n int64
	n, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}

	fmt.Println()

	if c.Verbose {
		fmt.Printf("\tWriting to file %s\n", path)
	}

	for {
		if n == int64(counter.Total) {
			out.Close()
			err = os.Rename(filename+".tmp", path)
			if err != nil {
				return fmt.Errorf("failed to write to path %s: %v", path, err)
			}
			goto Done
		}
		time.Sleep(200 * time.Millisecond)
	}

Done:
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

// PrintProgress prints the progress of a file write
func (wc WriteCounter) PrintProgress() {
	if wc.verbose {
		fmt.Printf("\r%s", strings.Repeat(" ", 50))
		fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
	}
}

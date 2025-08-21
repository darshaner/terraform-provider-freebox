package freebox

import (
	"bytes"
	"net/http"
)

type FreeboxClient struct {
	httpClient *http.Client
}

func NewFreeboxClient() *FreeboxClient {
	return &FreeboxClient{
		httpClient: &http.Client{},
	}
}

func (c *FreeboxClient) DoRequest(method, url string, body []byte) (*http.Response, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}

	// TODO: add authentication headers if required
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// global client for now
var fbClient = NewFreeboxClient()

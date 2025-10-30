package birdwatcher

// Http Birdwatcher Client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// ClientResponse is a json key value mapping
type ClientResponse map[string]any

// A Client uses the http client to talk
// to the birdwatcher API.
type Client struct {
	api string
}

// NewClient creates a new client instance
func NewClient(api string) *Client {
	// Strip trailing slashes from api base
	api = strings.TrimSuffix(api, "/")

	client := &Client{
		api: api,
	}
	return client
}

// GetEndpoint makes an API request and returns the
// response. The response body will be parsed further
// downstream.
func (c *Client) GetEndpoint(
	ctx context.Context,
	endpoint string,
) (*http.Response, error) {
	client := &http.Client{}
	url := c.api + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

// GetJSON makes an API request.
// Parse JSON response and return map or error.
func (c *Client) GetJSON(
	ctx context.Context,
	endpoint string,
) (ClientResponse, error) {
	res, err := c.GetEndpoint(ctx, endpoint)
	if err != nil {
		return ClientResponse{}, err
	}

	// Read body
	defer res.Body.Close()
	payload, err := io.ReadAll(res.Body)
	if err != nil {
		return ClientResponse{}, err
	}

	// Decode json payload
	result := make(ClientResponse)
	err = json.Unmarshal(payload, &result)
	if err != nil {
		return ClientResponse{}, err
	}
	return result, nil
}

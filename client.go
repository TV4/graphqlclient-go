// Package graphqlclient provides boilerplate for interfacing with a GraphQL
// server.
package graphqlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// Client is a generic GraphQL client
type Client struct {
	url        string
	httpClient *http.Client
	reqOpts    []func(*http.Request)
}

// New returns a new client. The optional reqOpts will be applied to all
// requests.
func New(url string, httpClient *http.Client, reqOpts ...func(*http.Request)) *Client {
	return &Client{
		httpClient: httpClient,
		url:        url,
		reqOpts:    reqOpts,
	}
}

// Query sends the given query and variables to the server. If the "errors"
// array in the response object contains any items, these will be unmarshaled
// and returned as an error. If there are no errors, the value of the "data"
// field of the response object with be unmarshaled into the "data" argument.
// reqOpts can be used to inspect or modify the request before it gets sent.
// These reqOpts are run after any reqOpts passed to func New.
func (c *Client) Query(ctx context.Context, query string, variables map[string]interface{}, data interface{}, reqOpts ...func(*http.Request)) error {
	body, err := json.Marshal(
		map[string]interface{}{
			"query":     query,
			"variables": variables,
		},
	)
	if err != nil {
		return fmt.Errorf("error encoding variables: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	for _, o := range c.reqOpts {
		o(req)
	}

	for _, o := range reqOpts {
		o(req)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error performing request: %v", err)
	}
	defer func() {
		io.CopyN(ioutil.Discard, resp.Body, 64)
		resp.Body.Close()
	}()

	var response struct {
		Data   json.RawMessage `json:"data"`
		Errors []Error         `json:"errors"`
	}

	var respBody io.Reader = resp.Body
	var respBodyBuf bytes.Buffer
	respBody = io.TeeReader(respBody, &respBodyBuf)

	if err := json.NewDecoder(respBody).Decode(&response); err != nil {
		if resp.StatusCode/100 != 2 {
			return &ErrorResponse{
				StatusCode: resp.StatusCode,
				Body:       respBodyBuf.Next(2048),
			}
		}
		return fmt.Errorf("error decoding response: %v", err)
	}

	if resp.StatusCode/100 != 2 || len(response.Errors) > 0 {
		return &ErrorResponse{
			StatusCode: resp.StatusCode,
			Errors:     response.Errors,
			Body:       respBodyBuf.Next(2048),
		}
	}

	if err := json.Unmarshal(response.Data, &data); err != nil {
		return fmt.Errorf("error decoding data payload: %v", err)
	}

	return nil
}

// ErrorResponse wraps the HTTP status code returned from the server and the
// value of the response object's "errors" array. If the response body is not
// JSON, up to the first 2048 bytes of it will be stored in the Body field.
type ErrorResponse struct {
	StatusCode int
	Body       []byte
	Errors     []Error
}

// Error represents one item in the response object's "errors" array. Its
// structure is based on http://facebook.github.io/graphql/June2018/#sec-Errors.
type Error struct {
	Message   string `json:"message,omitempty"`
	Locations []struct {
		Line   int `json:"line,omitempty"`
		Column int `json:"column,omitempty"`
	} `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// Error returns a string representation of the error.
func (e *ErrorResponse) Error() string {
	var errMsg string
	if len(e.Errors) > 0 {
		errMsg = e.Errors[0].Message
	} else {
		errMsg = string(e.Body)
	}
	return fmt.Sprintf("%d %s: %s", e.StatusCode, http.StatusText(e.StatusCode), errMsg)
}

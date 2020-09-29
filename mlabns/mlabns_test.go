package mlabns

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
)

// reponseBody is a fake HTTP response body.
type reponseBody struct {
	reader io.Reader
}

// newResponseBody creates a new response body.
func newResponseBody(data []byte) io.ReadCloser {
	return &reponseBody{
		reader: bytes.NewReader(data),
	}
}

// Read reads the response body.
func (r *reponseBody) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

// Close closes the response body.
func (r *reponseBody) Close() error {
	return nil
}

type httpTransport struct {
	Response *http.Response
	Error    error
}

// newHTTPClient returns a mocked *http.Client.
func newHTTPClient(code int, body []byte, err error) *http.Client {
	return &http.Client{
		Transport: &httpTransport{
			Error: err,
			Response: &http.Response{
				Body:       newResponseBody(body),
				StatusCode: code,
			},
		},
	}
}

func (r *httpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Cannot be more concise than this (i.e. `return r.Error, r.Response`) because
	// http.Client.Do warns if both Error and Response are non nil
	if r.Error != nil {
		return nil, r.Error
	}
	return r.Response, nil
}

const (
	toolName  = "ndt7"
	userAgent = "ndt7-client-go/0.1.0"
)

func TestQueryCommonCase(t *testing.T) {
	const expectedFQDN = "ndt7-mlab1-nai01.measurementlab.org"
	client := NewClient(toolName, userAgent)
	client.HTTPClient = newHTTPClient(
		200, []byte(fmt.Sprintf(`{"fqdn":"%s"}`, expectedFQDN)), nil,
	)
	fqdn, err := client.Query(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if fqdn != expectedFQDN {
		t.Fatal("Not the FQDN we were expecting")
	}
}

func TestQueryURLError(t *testing.T) {
	client := NewClient(toolName, userAgent)
	client.BaseURL = "\t" // breaks the parser
	_, err := client.Query(context.Background())
	if err == nil {
		t.Fatal("We were expecting an error here")
	}
}

func TestQueryNewRequestError(t *testing.T) {
	mockedError := errors.New("mocked error")
	client := NewClient(toolName, userAgent)
	client.RequestMaker = func(
		method, url string, body io.Reader) (*http.Request, error,
	) {
		return nil, mockedError
	}
	_, err := client.Query(context.Background())
	if err != mockedError {
		t.Fatal("Not the error we were expecting")
	}
}

func TestQueryNetworkError(t *testing.T) {
	mockedError := errors.New("mocked error")
	client := NewClient(toolName, userAgent)
	client.HTTPClient = newHTTPClient(
		0, []byte{}, mockedError,
	)
	_, err := client.Query(context.Background())
	// According to Go docs, the return value of http.Client.Do is always
	// of type `*url.Error` and wraps the original error.
	if err.(*url.Error).Err != mockedError {
		t.Fatal("Not the error we were expecting")
	}
}

func TestQueryInvalidStatusCode(t *testing.T) {
	client := NewClient(toolName, userAgent)
	client.HTTPClient = newHTTPClient(
		500, []byte{}, nil,
	)
	_, err := client.Query(context.Background())
	if err != ErrQueryFailed {
		t.Fatal("Not the error we were expecting")
	}
}

func TestQueryJSONParseError(t *testing.T) {
	client := NewClient(toolName, userAgent)
	client.HTTPClient = newHTTPClient(
		200, []byte("{"), nil,
	)
	_, err := client.Query(context.Background())
	if err == nil {
		t.Fatal("We expected an error here")
	}
}

func TestQueryNoServers(t *testing.T) {
	client := NewClient(toolName, userAgent)
	client.HTTPClient = newHTTPClient(
		204, []byte(""), nil,
	)
	_, err := client.Query(context.Background())
	if err != ErrNoAvailableServers {
		t.Fatal("Not the error we were expecting")
	}
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	client := NewClient(toolName, userAgent)
	fqdn, err := client.Query(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if fqdn == "" {
		t.Fatal("unexpected empty fqdn")
	}
}

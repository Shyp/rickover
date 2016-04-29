package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/Shyp/rickover/config"
)

var defaultTimeout = 6500 * time.Millisecond
var defaultHttpClient = &http.Client{Timeout: defaultTimeout}

// Client is a generic Rest client for making HTTP requests.
type Client struct {
	Id     string
	Token  string
	Client *http.Client
	Base   string
}

// NewClient returns a new Client with the given user and password. Base is the
// scheme+domain to hit for all requests. By default, the request timeout is
// set to 6.5 seconds.
func NewClient(user, pass, base string) *Client {
	return &Client{
		Id:     user,
		Token:  pass,
		Client: defaultHttpClient,
		Base:   base,
	}
}

// NewRequest creates a new Request and sets basic auth based on
// the client's authentication information.
func (c *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.Base+path, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Id, c.Token)
	req.Header.Add("User-Agent", fmt.Sprintf("shyp-go/v%s", config.Version))
	if method == "POST" || method == "PUT" {
		req.Header.Add("Content-Type", "application/json; charset=utf-8")
	}
	return req, nil
}

// Do performs the HTTP request. If the HTTP response is in the 2xx range,
// Unmarshal the response body into v, otherwise return an error.
func (c *Client) Do(r *http.Request, v interface{}) error {
	b := new(bytes.Buffer)
	if os.Getenv("DEBUG_HTTP_TRAFFIC") == "true" || os.Getenv("DEBUG_HTTP_REQUEST") == "true" {
		bits, err := httputil.DumpRequestOut(r, true)
		if err != nil {
			return err
		}
		b.Write(bits)
	}
	res, err := c.Client.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if os.Getenv("DEBUG_HTTP_TRAFFIC") == "true" || os.Getenv("DEBUG_HTTP_RESPONSES") == "true" {
		bits, err := httputil.DumpResponse(res, true)
		if err != nil {
			return err
		}
		b.Write(bits)
	}
	if b.Len() > 0 {
		_, err = b.WriteTo(os.Stderr)
		if err != nil {
			return err
		}
	}
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		var errMap map[string]interface{}
		err = json.Unmarshal(resBody, &errMap)
		if err != nil {
			return fmt.Errorf("invalid response body: %s", string(resBody))
		}

		if e, ok := errMap["title"]; ok {
			err := &Error{
				Title:      e.(string),
				StatusCode: res.StatusCode,
			}
			if detail, ok := errMap["detail"]; ok {
				err.Detail = detail.(string)
			}
			if id, ok := errMap["id"]; ok {
				err.Id = id.(string)
			}
			if instance, ok := errMap["instance"]; ok {
				err.Instance = instance.(string)
			}
			if t, ok := errMap["type"]; ok {
				err.Type = t.(string)
			}
			return err
		} else {
			return fmt.Errorf("invalid response body: %s", string(resBody))
		}
	}

	if v == nil {
		return nil
	} else {
		return json.Unmarshal(resBody, v)
	}
}

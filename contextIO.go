/*
Package contextio provides a simple way to sign API requests for http://Context.IO.

The simplest usage is to use DoJSON() to return a json byte array that you can use elsewhere in your code.
For more advanced usage, you can use Do() and parse through the http.Response struct yourself. It is not
specific to an API version, so you can use it to make any request you would make through http://console.Context.IO.
*/
package contextio

import (
	"bytes"
	"flag"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/garyburd/go-oauth/oauth"
)

// ContextIO is a struct containing the authentication information and a pointer to the oauth client
type ContextIO struct {
	key    string
	secret string
	client *oauth.Client
}

// NewContextIO returns a ContextIO struct based on your CIO User and Secret
func NewContextIO(key, secret string) *ContextIO {
	c := &oauth.Client{
		Credentials: oauth.Credentials{
			Token:  key,
			Secret: secret,
		},
	}

	return &ContextIO{
		key:    key,
		secret: secret,
		client: c,
	}
}

var apiHost = flag.String("apiHost", "api.context.io", "Use a specific host for the API")

// Do signs the request and returns an *http.Response. The body is a standard response.
// Body and must have defer response.Body.close().
// This is 2 legged authentication, and will not currently work with 3 legged authentication.
func (c *ContextIO) Do(method, q string, params url.Values, body *string) (response *http.Response, err error) {
	// make sure q has a slash in front of it
	if q[0:1] != "/" {
		q = "/" + q
	}

	req, _ := http.NewRequest(method, "https://"+*apiHost+q, bytes.NewBufferString(*body))
	req.URL.Opaque = q
	req.Header.Set("User-Agent", "GoContextIO Simple Library v. 0.1")
	v := url.Values{}
	switch method {
	case "PUT", "POST", "DELETE":
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		v, err = url.ParseQuery(*body)
		if err != nil {
			return
		}
	}

	err = c.client.SetAuthorizationHeader(req.Header, nil, req.Method, req.URL, v)
	if err != nil {
		return
	}
	return http.DefaultClient.Do(req)
}

// DoJSON passes the request to Do and then returns the json in a []byte array
func (c *ContextIO) DoJSON(method, q string, params url.Values, body *string) (json []byte, err error) {
	response, err := c.Do(method, q, params, body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	json, err = ioutil.ReadAll(response.Body)
	return json, err
}

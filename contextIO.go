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
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/garyburd/go-oauth/oauth"
)

const (
	defaultMaxMemory = 32 << 21 // 64 MB
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

func (c *ContextIO) NewRequest(method, q string, queryParams, postParams url.Values) (req *Request, err error) {
	// make sure q has a slash in front of it
	if q[0:1] != "/" {
		q = "/" + q
	}

	query := *apiHost + q
	if len(queryParams) > 0 {
		query = query + "?" + queryParams.Encode()
	}
	req = &Request{
		C:          c,
		Query:      q,
		Attachment: "",
	}
	req.Request, err = http.NewRequest(method, "https://"+query, nil)
	if err != nil {
		return nil, err
	}
	req.URL.Opaque = q
	req.SetUserAgent("GoContextIO Simple Library v. 0.1")
	err = req.Sign()
	return req, err
}

// AttachFile will create a file upload in the request, assumes NewRequest has already been called
func (c *ContextIO) AttachFile(req *http.Request, fieldName, fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filepath.Base(fileName))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, f)

	// transfer the existing post vals into the new body
	for key, valSlice := range req.PostForm {
		for _, val := range valSlice {
			err = writer.WriteField(key, val)
			if err != nil {
				return err
			}
		}
	}
	err = writer.Close()
	if err != nil {
		return err
	}
	rc := ioutil.NopCloser(body)
	req.Body = rc
	// update the form
	req.ParseMultipartForm(defaultMaxMemory)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return nil
}

// Do signs the request and returns an *http.Response. The body is a standard response.Body
// and must have defer response.Body.close().  Does not support uploads, use NewRequest and AttachFile for that.
// This is 2 legged authentication, and will not currently work with 3 legged authentication.
func (c *ContextIO) Do(method, q string, queryParams, postParams url.Values) (response *http.Response, err error) {
	req, err := c.NewRequest(method, q, queryParams, postParams)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req.Request)
}

// DoJSON passes the request to Do and then returns the json in a []byte array
func (c *ContextIO) DoJSON(method, q string, queryParams, postParams url.Values, body *string) (json []byte, err error) {
	response, err := c.Do(method, q, queryParams, postParams)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	json, err = ioutil.ReadAll(response.Body)
	return json, err
}

type Request struct {
	*http.Request
	C          *ContextIO
	Query      string
	Attachment string
}

func (r *Request) SetUserAgent(ua string) {
	r.Header.Set("User-Agent", ua)
}

func (r *Request) Sign() error {
	return r.C.client.SetAuthorizationHeader(r.Header, nil, r.Method, r.URL, r.PostForm)
}

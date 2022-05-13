// Package millennium is a Millennium ERP library
// written in Go to facilitate the integration with
// Millennium ERP.
package millennium

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/go-ntlmssp"
	"github.com/hashicorp/go-retryablehttp"
)

// AuthType Millennium authentication type
type AuthType string

// Authentication types available for Millennium
const (
	NTLM    AuthType = "NTLM"
	Session AuthType = "SESSION"
)

// HTTPMethod type to communicate with Millennium
type HTTPMethod string

// HTTP methods available
const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	DELETE HTTPMethod = "DELETE"
)

const (
	RetryMax = 3
)

// Millennium struct has the essential information to communicate with Millennium ERP
type Millennium struct {
	// Server used to store the server address
	ServerAddr string

	// Client points to retryable http lib
	Client *retryablehttp.Client

	// Context
	Context context.Context

	Timeout time.Duration

	// Headers is a map of headers to pass to requests
	headers http.Header

	// credentials store the user data
	credentials struct {
		Username string
		Password string
		AuthType AuthType
		Session  string
	}
}

// ResponseLogin type is the standard response struct from login requests
type ResponseLogin struct {
	Session string `json:"session"`
}

// ResponseGet type is the standard response struct from GET requests
type ResponseGet struct {
	Count int              `json:"odata.count"`
	Value *json.RawMessage `json:"value"`
}

// ResponseError type is the standard response struct for errors
type ResponseError struct {
	Err struct {
		Code    int `json:"code"`
		Message struct {
			Lang  string `json:"lang"`
			Value string `json:"value"`
		} `json:"message"`
	} `json:"error"`
}

func (r *ResponseError) String() string {
	return r.Err.Message.Value
}

func (r *ResponseError) Error() string {
	return r.Err.Message.Value
}

// SetMessage sets a custom error message
func (r *ResponseError) SetMessage(message string) {
	r.Err.Message.Value = message
}

// SetCode sets a custom error code
func (r *ResponseError) SetCode(code int) {
	r.Err.Code = code
}

// Deprecated: just for backward compatibility
func Client(server string, timeout time.Duration) (*Millennium, error) {
	return NewClient(context.Background(), server, timeout)
}

// NewClient returns a new Millennium instance with the server address and timeout
func NewClient(ctx context.Context, server string, timeout time.Duration) (*Millennium, error) {
	if server == "" {
		return nil, errors.New("no server address defined")
	}

	if timeout == 0*time.Second {
		return nil, errors.New("timeout is zero")
	}

	m := &Millennium{
		ServerAddr: server,
		Context:    ctx,
		Timeout:    timeout,
		headers:    http.Header{},
	}

	if m.Context == nil {
		m.Context = context.Background()
	}

	m.Client = m.setClient()

	return m, nil
}

func (m *Millennium) setClient() *retryablehttp.Client {
	client := retryablehttp.NewClient()
	client.RetryMax = RetryMax

	return client
}

// Login requests login to Millennium server
// server should be a valid URL with Millennium port, like: https://127.0.0.1:6018
func (m *Millennium) Login(username string, password string, authType AuthType) error {
	// Set Username and Password in credentials
	m.credentials.Username = username
	m.credentials.Password = password

	// If AuthType equals NTLM then set client transport to ntlm negotiator
	if authType == NTLM {
		m.Client.HTTPClient.Transport = ntlmssp.Negotiator{
			RoundTripper: &http.Transport{},
		}
	}

	if authType == Session {
		var responseLogin ResponseLogin
		m.headers.Set("WTS-Authorization", fmt.Sprintf("%s/%s", strings.ToUpper(m.credentials.Username), strings.ToUpper(m.credentials.Password)))
		if err := m.Post("login", []byte{}, &responseLogin); err != nil {
			return err
		}

		m.headers.Del("WTS-Authorization")

		m.credentials.Session = responseLogin.Session
		m.headers.Set("WTS-Session", m.credentials.Session)
	}

	m.credentials.AuthType = authType

	return nil
}

// RequestMethod receive data to pass to Request function
type RequestMethod struct {
	HTTPMethod HTTPMethod
	Method     string
	Params     url.Values
	Body       []byte
	Response   interface{}
}

// Request a method from Millennium
func (m *Millennium) Request(r RequestMethod) (err error) {
	// Transform body of type []byte to io.Reader
	bodyReader := bytes.NewReader(r.Body)

	// Ensure that the Millennium method is defined before request
	if r.Method == "" {
		return errors.New("requested method could not be empty")
	}

	// Ensure Params set if it is empty (nil)
	if r.Params == nil {
		r.Params = url.Values{}
	}

	// Ensure Response defined if http methods are GET or POST
	if r.Response == nil && (r.HTTPMethod == http.MethodPost || r.HTTPMethod == http.MethodGet) {
		return errors.New("response should have something to point to")
	}

	// Add default parameters for Millennium request
	r.Params.Add("$format", "json")
	r.Params.Add("$dateformat", "iso")

	// Start a new request
	requestMethod := string(r.HTTPMethod)
	requestURL := fmt.Sprintf("%s/api/%s?%s", m.ServerAddr, r.Method, r.Params.Encode())
	requestBody := bodyReader

	req, err := retryablehttp.NewRequestWithContext(m.Context, requestMethod, requestURL, requestBody)
	if err != nil {
		return fmt.Errorf("unable to start new request to Millennium: %w", err)
	}

	if m.headers != nil {
		req.Header = m.headers
	}

	// If authType is NTLM, set basic auth on request
	if m.credentials.AuthType == NTLM {
		req.SetBasicAuth(m.credentials.Username, m.credentials.Password)
	}

	return m.sendRequest(req, &r.Response)
}

func (m *Millennium) sendRequest(request *retryablehttp.Request, response interface{}) error {
	// Request using the client
	ctx, cancel := context.WithTimeout(request.Context(), m.Timeout)
	request = request.WithContext(ctx)
	defer cancel()

	res, err := m.Client.Do(request)
	if err != nil {
		return fmt.Errorf("unable to send request: %w", err)
	}

	return m.getResponse(res, &response)
}

// Will handle the response from Millennium for GET requests
func (m *Millennium) getResponse(res *http.Response, output interface{}) error {
	// Convert the response body to []byte
	bodyRes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("unable to read body from Millennium response: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		var resErr ResponseError
		if err = json.Unmarshal(bodyRes, &resErr); err != nil {
			return fmt.Errorf("unable to unmarshal error response: %w", err)
		}

		return &resErr
	}

	// Unmarshal the response JSON to interface pointer
	return json.Unmarshal(bodyRes, &output)
}

// Get requests a method using GET http method
func (m *Millennium) Get(method string, params url.Values, response interface{}) (int, error) {
	var res ResponseGet

	// Send a GET request to Millennium server
	err := m.Request(RequestMethod{
		HTTPMethod: GET,
		Method:     method,
		Params:     params,
		Response:   &res,
	})

	if err != nil {
		return 0, fmt.Errorf("unable to make the request to Millennium: %w", err)
	}

	// Unmarshal response values to response parameter
	if err := json.Unmarshal(*res.Value, response); err != nil {
		return 0, fmt.Errorf("unable to unmarshal JSON: %w", err)
	}

	// If no error ocurs, return the total number of values
	return res.Count, nil
}

// Post requests a method using POST http method
func (m *Millennium) Post(method string, body []byte, response interface{}) error {
	return m.Request(RequestMethod{
		HTTPMethod: POST,
		Method:     method,
		Params:     url.Values{},
		Body:       body,
		Response:   &response,
	})
}

// Delete requests a method using DELETE http method
func (m *Millennium) Delete(method string, params url.Values) error {
	return m.Request(RequestMethod{
		HTTPMethod: DELETE,
		Method:     method,
		Params:     params,
	})
}

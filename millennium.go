// Author: Fábio Matavelli <fabiomatavelli@gmail.com>

// This is a Millennium SDK written in Go to facilitate
// the integration with the Millennium ERP.

package millennium

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	ntlmssp "github.com/Azure/go-ntlmssp"
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

// Millennium struct has the essential information to communicate with Millennium ERP
type Millennium struct {
	// Server used to store the server address
	ServerAddr string

	// Client HTTP
	Client *http.Client

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

// Client returns a new Millennium instance with the server address
func Client(server string, timeout time.Duration) (*Millennium, error) {
	if server == "" {
		return nil, errors.New("No server defined")
	}

	if timeout == 0*time.Second {
		return nil, errors.New("No timeout set")
	}

	// Parse the server address
	addr, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	// Test server connection
	conn, err := net.DialTimeout("tcp", addr.Host, timeout)
	if err != nil {
		return nil, err
	}
	conn.Close()

	return &Millennium{
		ServerAddr: server,
		Client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Login requests login to Millennium server
// server should be a valid URL with Millennium port, like: https://127.0.0.1:6018
func (m *Millennium) Login(username string, password string, authType AuthType) error {
	// Set Username and Password in credentials
	m.credentials.Username = username
	m.credentials.Password = password

	// If AuthType equals NTLM then set client transport to ntlm negotiator
	if authType == NTLM {
		m.Client.Transport = ntlmssp.Negotiator{
			RoundTripper: &http.Transport{},
		}
	}

	if authType == Session {
		var responseLogin ResponseLogin
		m.headers = http.Header{}
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

// Request a method from Millennium
func (m *Millennium) Request(httpMethod HTTPMethod, method string, params url.Values, body []byte, response interface{}) error {
	// Transform body of type []byte to io.Reader
	bodyReader := bytes.NewReader(body)

	// Add default parameters for Millennium request
	params.Add("$format", "json")
	params.Add("$dateformat", "iso")

	// Start a new request
	req, err := http.NewRequest(string(httpMethod), fmt.Sprintf("%s/api/%s?%s", m.ServerAddr, method, params.Encode()), bodyReader)
	req.Header = m.headers

	if err != nil {
		return err
	}

	// If authType is NTLM, set basic auth on request
	if m.credentials.AuthType == NTLM {
		req.SetBasicAuth(m.credentials.Username, m.credentials.Password)
	}

	// Request using the client
	res, err := m.Client.Do(req)

	if err != nil {
		return err
	}

	// Convert the response body to []byte
	bodyRes, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		var resErr ResponseError
		json.Unmarshal(bodyRes, &resErr)
		return &resErr
	}

	// Unmarshal the response JSON to interface pointer
	if err = json.Unmarshal(bodyRes, &response); err != nil {
		return err
	}

	return nil
}

// Get requests a method using GET http method
func (m *Millennium) Get(method string, params url.Values, response interface{}) (int, error) {
	var res ResponseGet

	// Send a GET request to Millennium server
	if err := m.Request(GET, method, params, []byte{}, &res); err != nil {
		return 0, err
	}

	// Unmarshal response values to response parameter
	if err := json.Unmarshal(*res.Value, response); err != nil {
		return 0, nil
	}

	// If no error ocurs, return the total number of values
	return res.Count, nil
}

// Post requests a method using POST http method
func (m *Millennium) Post(method string, body []byte, response interface{}) error {
	if err := m.Request(POST, method, url.Values{}, body, &response); err != nil {
		return err
	}
	return nil
}

// Delete requests a method using DELETE http method
func (m *Millennium) Delete(method string, params url.Values) error {
	if err := m.Request(DELETE, method, params, nil, nil); err != nil {
		return err
	}

	return nil
}

// Author: Fábio Matavelli <fabiomatavelli@gmail.com>

// This is a Millennium SDK written in Go to facilitate
// the integration with the Millennium ERP.

package millennium

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/tidwall/gjson"
)

var (
	apiHost     string
	apiProtocol string
	apiURL      string
	wtsSession  string
)

const (
	// ErrorNotAuthorized show Not Authorized error
	ErrorNotAuthorized = "Login not authorized."
	// ErrorNotLoggedIn show Not Logged in error
	ErrorNotLoggedIn = "Not logged in."
	// ErrorMethodNotFound show Method not found error
	ErrorMethodNotFound = "Method not found."
	// ErrorMethodExecution show Method execution error
	ErrorMethodExecution = "Problem to execute method."
	// ErrorUnmarshalling show Method unmarshalling error
	ErrorUnmarshalling = "Problem to parse JSON"
)

var client = &http.Client{}

// Login into Millennium and generate the token
func Login(hostname string, username string, password string, ssl bool) (bool, error) {
	apiHost = hostname

	if ssl == true {
		apiProtocol = "https"
	} else {
		apiProtocol = "http"
	}

	apiURL = fmt.Sprintf("%s://%s/api", apiProtocol, apiHost)

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/login?$format=json", apiURL), nil)
	req.Header.Set("WTS-Authorization", fmt.Sprintf("%s/%s", strings.ToUpper(username), strings.ToUpper(password)))
	res, _ := client.Do(req)

	if res.StatusCode == 401 {
		wtsSession = ""
		return false, errors.New(ErrorNotAuthorized)
	}

	body, _ := ioutil.ReadAll(res.Body)

	if err != nil {
		return false, err
	}

	data, ok := gjson.ParseBytes(body).Value().(map[string]interface{})

	if !ok {
		return false, errors.New(ErrorUnmarshalling)
	}

	wtsSession = data["session"].(string)

	return true, nil
}

// Logout from Millennium
func Logout() (bool, error) {
	if wtsSession == "" {
		return false, errors.New(ErrorNotLoggedIn)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/logout", apiURL), nil)
	req.Header.Set("WTS-Session", wtsSession)
	res, _ := client.Do(req)

	if err != nil {
		return false, err
	}

	if res.StatusCode == 200 {
		return true, nil
	}

	return false, nil
}

// Call Millennium API
func Call(method string, methodType string, params map[string]interface{}) (interface{}, error) {
	if wtsSession == "" {
		return nil, errors.New(ErrorNotLoggedIn)
	}

	p := url.Values{}
	p.Set("$format", "json")
	p.Add("$dateformat", "iso")

	for key, val := range params {
		p.Add(key, val.(string))
	}

	req, err := http.NewRequest(methodType, fmt.Sprintf("%s/%s?%s", apiURL, method, p.Encode()), nil)
	req.Header.Set("WTS-Session", wtsSession)
	res, _ := client.Do(req)

	if err != nil {
		return nil, err
	}

	switch res.StatusCode {
	case 401:
		return nil, errors.New(ErrorNotAuthorized)
	case 404:
		return nil, fmt.Errorf("%s: %s", method, ErrorMethodNotFound)
	case 400:
		return nil, fmt.Errorf("%s: %s", method, ErrorMethodExecution)
	case 500:
		return nil, fmt.Errorf("%s: %s", method, ErrorMethodExecution)
	}

	body, _ := ioutil.ReadAll(res.Body)

	result, ok := gjson.ParseBytes(body).Value().(map[string]interface{})
	if !ok {
		return nil, errors.New(ErrorUnmarshalling)
	}

	if methodType == "GET" {
		return result["value"].(interface{}), nil
	} else if methodType == "POST" {
		return result, nil
	}

	return nil, nil
}

// Get data from API
func Get(method string, params map[string]interface{}) (interface{}, error) {
	return Call(method, "GET", params)
}

// Post data to API
func Post(method string, params map[string]interface{}) (interface{}, error) {
	return Call(method, "POST", params)
}

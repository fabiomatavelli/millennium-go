// Author: Fábio Matavelli <fabiomatavelli@gmail.com>

// This is a Millennium SDK written in Go to facilitate
// the integration with the Millennium ERP.

package millennium

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var (
	api_host string
	api_protocol string
	api_url string
	wts_session string
)

const (
	ERROR_NOT_AUTHORIZED = "Login não autorizado."
	ERROR_NOT_LOGGED_IN = "Login não efetuado."
	ERROR_METHOD_NOT_FOUND = "Método não encontrado."
	ERROR_METHOD_EXECUTION = "Problema ao executar o método."
)

type MillenniumResult struct {
	Count int `json:"odata.count"`
	Result []interface{} `json:"value"`
}

type MillenniumLogin struct {
	Session string `json:"session"`
}

var client = &http.Client{}

// Login into Millennium and generate the token
func Login(hostname string, username string, password string, ssl bool) (bool, error) {
	api_host = hostname
	
	if ssl == true {
		api_protocol = "https"
	} else {
		api_protocol = "http"
	}

	api_url = fmt.Sprintf("%s://%s/api",api_protocol, api_host)

	req, err := http.NewRequest("GET",fmt.Sprintf("%s/login?$format=json",api_url),nil)
	req.Header.Set("WTS-Authorization",fmt.Sprintf("%s/%s",strings.ToUpper(username),strings.ToUpper(password)))
	res, _ := client.Do(req)

	if res.StatusCode == 401 {
		wts_session = ""
		return false, errors.New(ERROR_NOT_AUTHORIZED)
	}

	body, err := ioutil.ReadAll(res.Body)

	var session = new(MillenniumLogin)

	err = json.Unmarshal(body, &session)

	if err != nil {
		return false, err
	}

	wts_session = session.Session

	return true, nil
}

// Call Millennium API
func Call(method string, method_type string, params map[string]interface{}) (interface{}, error) {
	if wts_session == "" {
		return nil, errors.New(ERROR_NOT_LOGGED_IN)
	}

	p := url.Values{}
	p.Set("$format", "json")
	p.Add("$dateformat", "iso")

	for key,val := range params {
		p.Add(key, val.(string))
	}

	req, err := http.NewRequest(method_type, fmt.Sprintf("%s/%s?%s", api_url, method, p.Encode()), nil)
	req.Header.Set("WTS-Session", wts_session)
	res, _ := client.Do(req)

	if err != nil {
		return nil, err
	}

	switch(res.StatusCode) {
		case 401:
			return nil, errors.New(ERROR_NOT_AUTHORIZED)
		case 404:
			return nil, fmt.Errorf("%s: %s", method, ERROR_METHOD_NOT_FOUND)
		case 400:
			return nil, fmt.Errorf("%s: %s", method, ERROR_METHOD_EXECUTION)
		case 500:
			return nil, fmt.Errorf("%s: %s", method, ERROR_METHOD_EXECUTION)	
	}
	
	if method_type == "GET" {
		body, _ := ioutil.ReadAll(res.Body)

		var result = make(map[string]interface{})
		json.Unmarshal(body, &result)
		
		return result, nil
	}

	return nil, nil
}

// Get data from API
func Get(method string, params map[string]interface{}) (interface{}, error) {
	return Call(method, "GET", params)
}

//func Post(method string, params map[string]interface{})
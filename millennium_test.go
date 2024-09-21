package millennium

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

var (
	serverAddr string
)

type mockHTTPServer struct {
	testServer *httptest.Server
}

func (s *mockHTTPServer) jsonError(message string, errorCode int) []byte {
	response := ResponseError{}

	response.SetMessage(message)
	response.SetCode(errorCode)

	res, _ := json.Marshal(response)
	return res
}

type writeOutputParams struct {
	Writer      http.ResponseWriter
	Request     *http.Request
	StatusCode  int
	ContentType string
	Body        []byte
}

func (s *mockHTTPServer) writeOutput(p *writeOutputParams) {
	if p.ContentType != "" {
		p.Writer.Header().Set("Content-Type", p.ContentType)
	} else {
		p.Writer.Header().Set("Content-Type", "application/json")
	}

	if p.StatusCode > 0 {
		p.Writer.WriteHeader(p.StatusCode)
	} else {
		p.Writer.WriteHeader(200)
	}

	_, err := p.Writer.Write(p.Body)
	if err != nil {
		panic(err)
	}
}

func (s *mockHTTPServer) Start() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", http.NotFound)
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		var err error

		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("WTS-Authorization") == "TEST/TEST" {
			_, err = w.Write([]byte(`{"session":"{00000000-0000-0000-0000-000000000000}"}`))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			_, err = w.Write(s.jsonError("PERMISSÃO NEGADA:\r\rNão é possível autenticar o usuário. Senha inválida.", 401))
		}

		if err != nil {
			panic(err)
		}
	})
	mux.HandleFunc("/api/test.success.GET", func(w http.ResponseWriter, r *http.Request) {
		s.writeOutput(&writeOutputParams{
			Writer:     w,
			Request:    r,
			StatusCode: 200,
			Body:       []byte(`{"odata.count": 1,"value":[{"number":1,"string":"test","bool":true}]}`),
		})
	})
	mux.HandleFunc("/api/test.error400.GET", func(w http.ResponseWriter, r *http.Request) {
		s.writeOutput(&writeOutputParams{
			Writer:     w,
			Request:    r,
			StatusCode: http.StatusBadRequest,
			Body:       s.jsonError("Parameter not found", http.StatusBadRequest),
		})
	})
	mux.HandleFunc("/api/test.error500.GET", func(w http.ResponseWriter, r *http.Request) {
		s.writeOutput(&writeOutputParams{
			Writer:     w,
			Request:    r,
			StatusCode: http.StatusInternalServerError,
			Body:       s.jsonError("Query error", http.StatusInternalServerError),
		})
	})
	mux.HandleFunc("/api/test.error.invalidjson", func(w http.ResponseWriter, r *http.Request) {
		s.writeOutput(&writeOutputParams{
			Writer:  w,
			Request: r,
			Body:    []byte(`{"odata.count": 1,"value":["test":"test"}`),
		})
	})
	mux.HandleFunc("/api/test.error.invalidjsonerror", func(w http.ResponseWriter, r *http.Request) {
		s.writeOutput(&writeOutputParams{
			Writer:     w,
			Request:    r,
			StatusCode: http.StatusInternalServerError,
			Body:       []byte(`{"error":{"`),
		})
	})
	mux.HandleFunc("/api/test.error.empty", func(w http.ResponseWriter, r *http.Request) {
		s.writeOutput(&writeOutputParams{
			Writer:  w,
			Request: r,
			Body:    []byte(``),
		})
	})
	mux.HandleFunc("/api/test.success.POST", func(w http.ResponseWriter, r *http.Request) {
		s.writeOutput(&writeOutputParams{
			Writer:  w,
			Request: r,
			Body:    []byte(`{"number":1,"string":"test","bool":true}`),
		})
	})
	mux.HandleFunc("/api/test.error.POST", func(w http.ResponseWriter, r *http.Request) {
		s.writeOutput(&writeOutputParams{
			Writer:     w,
			Request:    r,
			StatusCode: http.StatusInternalServerError,
			Body:       s.jsonError("Internal Server Error", http.StatusInternalServerError),
		})
	})
	mux.HandleFunc("/api/test.success.DELETE", func(w http.ResponseWriter, r *http.Request) {
		s.writeOutput(&writeOutputParams{
			Writer:  w,
			Request: r,
			Body:    []byte(`{"odata.metadata":""}`),
		})
	})
	mux.HandleFunc("/api/test.error.DELETE", func(w http.ResponseWriter, r *http.Request) {
		s.writeOutput(&writeOutputParams{
			Writer:     w,
			Request:    r,
			StatusCode: http.StatusInternalServerError,
			Body:       s.jsonError("Query error", http.StatusInternalServerError),
		})
	})
	mux.HandleFunc("/api/test.basicauth", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "correct_user" || password != "correct_password" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "", http.StatusUnauthorized)
			return
		}

		s.writeOutput(&writeOutputParams{
			Writer:     w,
			Request:    r,
			StatusCode: 200,
			Body:       []byte(`{"odata.count": 1,"value":[{"number":1,"string":"test","bool":true}]}`),
		})
	})

	s.testServer = httptest.NewServer(mux)
	return s.testServer
}

func (s *mockHTTPServer) Stop() {
	s.testServer.Close()
}

func NewTestClient(t *testing.T) *Millennium {
	client, err := NewClient(context.Background(), serverAddr, 30*time.Second)
	if err != nil {
		t.Fatalf("Got error: %v", err)
	}
	return client
}

func TestMain(m *testing.M) {
	server := &mockHTTPServer{}
	server.Start()

	serverAddr = server.testServer.URL

	defer server.Stop()

	os.Exit(m.Run())
}

func TestClient(t *testing.T) {
	cases := []struct {
		Context     context.Context
		Server      string
		Timeout     time.Duration
		ExpectError bool
	}{
		{
			Context:     context.Background(),
			Server:      "",
			Timeout:     30 * time.Second,
			ExpectError: true,
		},
		{
			Context:     context.Background(),
			Server:      serverAddr,
			Timeout:     0 * time.Second,
			ExpectError: true,
		},
		{
			Context:     context.Background(),
			Server:      "http://1.2.3.4:6018",
			Timeout:     3 * time.Second,
			ExpectError: true,
		},
		{
			Context:     context.Background(),
			Server:      "http://1.2.3.4:6018\n",
			Timeout:     3 * time.Second,
			ExpectError: true,
		},
		{
			Context:     context.Background(),
			Server:      serverAddr,
			Timeout:     30 * time.Second,
			ExpectError: false,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s %v", c.Server, c.Timeout), func(t *testing.T) {
			var x []interface{}
			cli, err := NewClient(c.Context, c.Server, c.Timeout)
			if (err != nil) == c.ExpectError {
				t.Logf("Got expected error: %s", err)
			} else {
				t.Logf("Trying to send request to: '%v' with timeout %v", c.Server, c.Timeout)
				err = cli.Request(RequestMethod{
					HTTPMethod: http.MethodGet,
					Method:     "x.x.x",
					Response:   &x,
				})

				if (err != nil) == c.ExpectError {
					t.Logf("Got expected error: %s", err)
				} else {
					t.Errorf("Got unexpected error: %s!", err)
				}
			}
		})
	}
}

func TestLogin(t *testing.T) {
	client := NewTestClient(t)
	cases := []struct {
		Username    string
		Password    string
		AuthType    AuthType
		ExpectError bool
	}{
		{
			Username:    "test",
			Password:    "wrongpassword",
			AuthType:    Session,
			ExpectError: true,
		},
		{
			Username:    "test",
			Password:    "test",
			AuthType:    Session,
			ExpectError: false,
		},
		{
			Username:    "test",
			Password:    "test",
			AuthType:    NTLM,
			ExpectError: false,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s:%s_%s", c.Username, c.Password, c.AuthType), func(t *testing.T) {
			err := client.Login(c.Username, c.Password, c.AuthType)
			if (err == nil) == c.ExpectError {
				t.Error(err)
			}
		})
	}
}

func TestNTLM(t *testing.T) {
	client := NewTestClient(t)
	err := client.Login("test", "test", NTLM)
	if err != nil {
		t.Error(err)
	}

	var _r interface{}
	x, err := client.Get("test.success.GET", url.Values{}, &_r)
	if err != nil {
		t.Error(err)
	}

	if x == 0 {
		t.Error("Zero records returned")
	}
}

func TestBasicAuth(t *testing.T) {
	client := NewTestClient(t)
	err := client.Login("test", "test", Basic)
	if err != nil {
		t.Error(err)
	}

	var _r interface{}

	_, err = client.Get("test.basicauth", url.Values{}, &_r)
	if err == nil {
		t.Error("Expected error")
	}

	err = client.Login("correct_user", "correct_password", Basic)
	if err != nil {
		t.Error(err)
	}

	x, err := client.Get("test.basicauth", url.Values{}, &_r)
	if err != nil {
		t.Fatal(err)
	}

	if x == 0 {
		t.Error("Zero records returned")
	}
}

func TestRequest(t *testing.T) {
	client := NewTestClient(t)

	var r interface{}
	cases := []struct {
		Params      RequestMethod
		ExpectError bool
	}{
		{
			Params: RequestMethod{
				HTTPMethod: "[GET",
				Method:     "test.success.GET",
			},
			ExpectError: true,
		},
		{
			Params: RequestMethod{
				HTTPMethod: "GET",
				Method:     "test.success.GET",
			},
			ExpectError: true,
		},
		{
			Params: RequestMethod{
				HTTPMethod: "GET",
			},
			ExpectError: true,
		},
		{
			Params: RequestMethod{
				HTTPMethod: "GET",
				Method:     "test.success.GET",
				Response:   &r,
			},
			ExpectError: false,
		},
		{
			Params: RequestMethod{
				HTTPMethod: "GET",
				Method:     "test.error.invalidjsonerror",
				Response:   &r,
			},
			ExpectError: true,
		},
		{
			Params: RequestMethod{
				HTTPMethod: "GET",
				Method:     "test.error.invalidjson",
				Response:   &r,
			},
			ExpectError: true,
		},
	}

	for x, c := range cases {
		err := client.Request(c.Params)

		if (err == nil) == c.ExpectError {
			t.Errorf("Test #%v got error '%v'", x, err)
		}
	}
}

func TestGet(t *testing.T) {
	client := NewTestClient(t)

	type ResponseTestGET struct {
		Number int    `json:"number"`
		String string `json:"string"`
		Bool   bool   `json:"bool"`
	}

	type Expect struct {
		Error    bool
		Count    int
		Response interface{}
	}

	var responseTestGET []ResponseTestGET

	cases := []struct {
		Method   string
		Params   url.Values
		Response interface{}
		Expect   Expect
	}{
		{
			Method:   "test.success.GET",
			Response: &responseTestGET,
			Params: url.Values{
				"test": []string{"test"},
			},
			Expect: Expect{
				Error: false,
				Count: 1,
			},
		},
		{
			Method:   "test.error400.GET",
			Response: &responseTestGET,
			Params:   url.Values{},
			Expect: Expect{
				Error: true,
				Count: 0,
			},
		},
		{
			Method:   "test.error500.GET",
			Response: &responseTestGET,
			Params:   url.Values{},
			Expect: Expect{
				Error: true,
				Count: 0,
			},
		},
		{
			Method:   "test.error.invalidjson",
			Response: &responseTestGET,
			Params:   url.Values{},
			Expect: Expect{
				Error: true,
				Count: 0,
			},
		},
		{
			Method:   "test.error.empty",
			Response: &responseTestGET,
			Params:   url.Values{},
			Expect: Expect{
				Error: true,
				Count: 0,
			},
		},
	}

	for x, c := range cases {
		t.Run(c.Method, func(t *testing.T) {
			count, err := client.Get(c.Method, c.Params, &c.Response)
			if (err == nil) == c.Expect.Error {
				t.Error(err)
			}

			if c.Expect.Error {
				if fmt.Sprint(err) == "" {
					t.Errorf("No error string returned on case %v", x)
				}
			}

			if count != c.Expect.Count {
				t.Errorf("Expected %v results but got %v", c.Expect.Count, count)
			}
		})
	}
}

func TestPost(t *testing.T) {
	client := NewTestClient(t)

	type ResponseTestPOST struct {
		Number int    `json:"number"`
		String string `json:"string"`
		Bool   bool   `json:"bool"`
	}

	cases := []struct {
		Method      string
		Body        []byte
		ExpectError bool
	}{
		{
			Method:      "test.success.POST",
			Body:        []byte(`{"test":"test"}`),
			ExpectError: false,
		},
		{
			Method:      "test.error.POST",
			Body:        []byte(`{"test":"test"}`),
			ExpectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.Method, func(t *testing.T) {
			var res *ResponseTestPOST
			err := client.Post(c.Method, c.Body, &res)
			if (err == nil) == c.ExpectError {
				t.Error(err)
			}
			_ = res
		})
	}
}

func TestDelete(t *testing.T) {
	client := NewTestClient(t)

	cases := []struct {
		Method      string
		Params      url.Values
		Response    interface{}
		ExpectError bool
	}{
		{
			Method: "test.success.DELETE",
			Params: url.Values{
				"test": []string{"test"},
			},
			ExpectError: false,
		},
		{
			Method: "test.error.DELETE",
			Params: url.Values{
				"test": []string{"test"},
			},
			ExpectError: true,
		},
	}

	for _, c := range cases {
		err := client.Delete(c.Method, c.Params)
		if (err == nil) == c.ExpectError {
			t.Error(err)
		}
	}
}

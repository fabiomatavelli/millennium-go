package millennium

import (
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


func (s *mockHTTPServer) Start() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", http.NotFound)
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("WTS-Authorization") == "TEST/TEST" {
			w.Write([]byte(`{"session":"{00000000-0000-0000-0000-000000000000}"}`))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"code":401,"message":{"lang":"en-us","value":"PERMISSÃO NEGADA:\r\rNão é possível autenticar o usuário. Senha inválida."}}}`))
		}
	})
	mux.HandleFunc("/api/test.success.GET", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"odata.count": 1,"value":[{"number":1,"string":"test","bool":true}]}`))
	})

	s.testServer = httptest.NewServer(mux)
	return s.testServer
}

func (s *mockHTTPServer) Stop() {
	s.testServer.Close()
}

func NewClient(t *testing.T) *Millennium {
	client, err := Client(serverAddr, 30*time.Second)
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
		Server      string
		Timeout     time.Duration
		ExpectError bool
	}{
		{
			Server:      "",
			Timeout:     30 * time.Second,
			ExpectError: true,
		},
		{
			Server:      serverAddr,
			Timeout:     0 * time.Second,
			ExpectError: true,
		},
		{
			Server:      "http://127.0.0.2:6018",
			Timeout:     1 * time.Second,
			ExpectError: true,
		},
		{
			Server:      serverAddr,
			Timeout:     30 * time.Second,
			ExpectError: false,
		},
	}

	for _, c := range cases {
		_, err := Client(c.Server, c.Timeout)
		t.Logf("Trying to connect to: '%v' with timeout %v", c.Server, c.Timeout)
		if (err == nil) == c.ExpectError {
			t.Error(err)
		} else {
			t.Logf("Passed verication of address '%v' with success!", c.Server)
		}
	}
}

func TestLogin(t *testing.T) {
	client := NewClient(t)
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
	}

	for _, c := range cases {
		err := client.Login(c.Username, c.Password, c.AuthType)
		if (err == nil) == c.ExpectError {
			t.Error(err)
		}
	}
}

func TestGet(t *testing.T) {
	client := NewClient(t)

	type ResponseTestGET struct {
		Number int    `json:"number"`
		String string `json:"string"`
		Bool   bool   `json:"bool"`
	}

	type Expect struct {
		Error bool
		Count int
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
	}

	for _, c := range cases {
		count, err := client.Get(c.Method, c.Params, &responseTestGET)
		if (err == nil) == c.Expect.Error {
			t.Error(err)
		}

		if count != c.Expect.Count {
			t.Errorf("Expected %v results but got %v", c.Expect.Count, count)
		}
	}
}

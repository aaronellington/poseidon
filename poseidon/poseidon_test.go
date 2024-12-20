package poseidon_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aaronellington/poseidon/poseidon"
)

type TestCase struct {
	Request            *http.Request
	ExpectedStatusCode int
	ExpectedHeaders    map[string]string
	ExpectedBody       string
	ConfigFuncs        []poseidon.ConfigFunc
}

func TestMiddlewareOrder(t *testing.T) {
	testService(t, TestCase{
		Request: httptest.NewRequest(
			http.MethodGet, "/",
			nil,
		),
		ExpectedStatusCode: http.StatusOK,
		ExpectedBody:       "middleware1",
		ConfigFuncs: []poseidon.ConfigFunc{
			poseidon.WithMiddleware(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("middleware1"))
				})
			}),
			poseidon.WithMiddleware(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("middleware2"))
				})
			}),
		},
	})
}

func TestCachePolicyIndex(t *testing.T) {
	testService(t, TestCase{
		Request: httptest.NewRequest(
			http.MethodGet, "/",
			nil,
		),
		ExpectedStatusCode: http.StatusOK,
		ExpectedBody:       "Hello there.\n",
		ExpectedHeaders: map[string]string{
			"content-typE":  "text/html; charset=utf-8",
			"Cache-Control": "no-store, no-cache, must-revalidate",
		},
		ConfigFuncs: []poseidon.ConfigFunc{
			poseidon.WithCachePolicy(),
		},
	})
}

func TestCachePolicyNext(t *testing.T) {
	testService(t, TestCase{
		Request: httptest.NewRequest(
			http.MethodGet, "/_next/assets.css",
			nil,
		),
		ExpectedStatusCode: http.StatusOK,
		ExpectedBody:       "* {margin: 0;}\n",
		ExpectedHeaders: map[string]string{
			"content-typE":  "text/css; charset=utf-8",
			"Cache-Control": "public,max-age=31536000,immutable",
		},
		ConfigFuncs: []poseidon.ConfigFunc{
			poseidon.WithCachePolicy(),
		},
	})
}

func TestCustomNotFoundFile(t *testing.T) {
	testService(t, TestCase{
		Request: httptest.NewRequest(
			http.MethodGet, "/foobar",
			nil,
		),
		ExpectedStatusCode: http.StatusNotFound,
		ExpectedBody:       "custom not found\n",
		ConfigFuncs: []poseidon.ConfigFunc{
			poseidon.WithCustomNotFoundFile("404.html"),
		},
	})
}

func TestSPANotFound(t *testing.T) {
	testService(t, TestCase{
		Request: httptest.NewRequest(
			http.MethodGet, "/foobar",
			nil,
		),
		ExpectedStatusCode: 200,
		ExpectedBody:       "Hello there.\n",
		ConfigFuncs: []poseidon.ConfigFunc{
			poseidon.WithSPA(),
		},
	})
}

func TestSPANotFoundNotFound(t *testing.T) {
	testService(t, TestCase{
		Request: httptest.NewRequest(
			http.MethodGet, "/foobar/",
			nil,
		),
		ExpectedStatusCode: 404,
		ExpectedBody:       "404 page not found\n",
		ConfigFuncs: []poseidon.ConfigFunc{
			poseidon.WithSPA(),
			poseidon.WithCustomIndex("non-existing-index.html"),
		},
	})
}

func TestRootIndex(t *testing.T) {
	testService(t, TestCase{
		Request: httptest.NewRequest(
			http.MethodGet, "/",
			nil,
		),
		ExpectedStatusCode: 200,
		ExpectedBody:       "Hello there.\n",
	})
}

func TestRootFile(t *testing.T) {
	testService(t, TestCase{
		Request: httptest.NewRequest(
			http.MethodGet, "/robots.txt",
			nil,
		),
		ExpectedStatusCode: 200,
		ExpectedBody:       "Disallow: *\n",
		ExpectedHeaders: map[string]string{
			"content-type": "text/plain; charset=utf-8",
		},
	})
}

func TestFolderIndexWithoutTrailingSlash(t *testing.T) {
	testService(t, TestCase{
		Request: httptest.NewRequest(
			http.MethodGet, "/folder",
			nil,
		),
		ExpectedStatusCode: 200,
		ExpectedBody:       "This is in a folder.\n",
		ExpectedHeaders: map[string]string{
			"content-type": "text/html; charset=utf-8",
		},
	})
}

func TestFolderIndexWithTrailingSlash(t *testing.T) {
	testService(t, TestCase{
		Request: httptest.NewRequest(
			http.MethodGet, "/folder",
			nil,
		),
		ExpectedStatusCode: 200,
		ExpectedBody:       "This is in a folder.\n",
		ExpectedHeaders: map[string]string{
			"content-type": "text/html; charset=utf-8",
		},
	})
}

func testService(t *testing.T, testCase TestCase) {
	service, err := poseidon.New(os.DirFS("test_data"), testCase.ConfigFuncs...)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Request: %s", r.URL.Path)
		service.ServeHTTP(w, r)
	}).ServeHTTP(recorder, testCase.Request)

	if testCase.ExpectedStatusCode != recorder.Code {
		t.Fatalf("Unexpected Status Code, Got: %d, Expected: %d", recorder.Code, testCase.ExpectedStatusCode)
	}

	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != testCase.ExpectedBody {
		t.Fatalf("Unexpected Body, Got: %s, Expected: %s", body, testCase.ExpectedBody)
	}

	for key, value := range testCase.ExpectedHeaders {
		if recorder.Header().Get(key) != value {
			t.Fatalf("Unexpected Header %s, Got: %s, Expected: %s", key, recorder.Header().Get(key), value)
		}
	}
}
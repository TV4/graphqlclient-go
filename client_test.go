package graphqlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestClient_Query(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			gotFooHeader   string
			gotContentType string
			gotBody        []byte
			gotCtxValue    interface{}
		)

		ts := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				var err error

				gotContentType = r.Header.Get("Content-Type")
				gotFooHeader = r.Header.Get("Foo-Header")

				gotBody, err = ioutil.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				w.Write([]byte(`{"data":"foo-data"}`))
			},
		))
		defer ts.Close()

		type key int
		var ctxKey key

		ctx := context.WithValue(context.Background(), ctxKey, "ctx-value")
		query := "foo-query"
		variables := map[string]interface{}{
			"foo-variable": "foo-value",
			"bar-variable": 123,
		}
		var data interface{}

		calledReqOpts := []string{}

		reqOptNew := func(req *http.Request) {
			calledReqOpts = append(calledReqOpts, "opt-passed-to-new")
			req.Header.Set("Foo-Header", "old-foo-header-value")
		}

		reqOptQuery := func(req *http.Request) {
			calledReqOpts = append(calledReqOpts, "opt-passed-to-query")
			req.Header.Set("Foo-Header", "foo-header-value")
			gotCtxValue = req.Context().Value(ctxKey)
		}

		c := New(ts.URL, &http.Client{}, reqOptNew)

		if err := c.Query(ctx, query, variables, &data, reqOptQuery); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got, want := gotContentType, "application/json; charset=utf-8"; got != want {
			t.Errorf("Content-Type = %q, want %q", got, want)
		}

		if got, want := gotFooHeader, "foo-header-value"; got != want {
			t.Errorf("Foo-Header = %q, want %q", got, want)
		}

		wantCalledReqOpts := []string{"opt-passed-to-new", "opt-passed-to-query"}
		if len(calledReqOpts) != len(wantCalledReqOpts) {
			t.Errorf("req opts called = %q, want %q", calledReqOpts, wantCalledReqOpts)
		} else {
			for n := range wantCalledReqOpts {
				if calledReqOpts[n] != wantCalledReqOpts[n] {
					t.Errorf("req opts called = %q, want %q", calledReqOpts, wantCalledReqOpts)
					break
				}
			}
		}

		wantBody := []byte(`{"query":"foo-query","variables":{"bar-variable":123,"foo-variable":"foo-value"}}`)
		if got, want := gotBody, wantBody; !bytes.Equal(got, want) {
			t.Errorf("request body = `%s`, want `%s`", got, want)
		}

		if s, ok := gotCtxValue.(string); !ok || s != "ctx-value" {
			t.Errorf("ctx not propagated (%q, %t)", s, ok)
		}

		wantDataValue := "foo-data"
		if gotDataValue, ok := data.(string); ok {
			if gotDataValue != wantDataValue {
				t.Errorf("response data is %q, want %q", gotDataValue, wantDataValue)
			}
		} else {
			t.Errorf("response data is %q (%[1]T), want %q (%[2]T)", data, wantDataValue)
		}
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTeapot)
				w.Write([]byte(`{"errors":[{"message":"error-msg-1"},{"message":"error-msg-2"}]}`))
			},
		))
		defer ts.Close()

		c := New(ts.URL, &http.Client{})

		err := c.Query(context.Background(), "", nil, nil)

		if err == nil {
			t.Fatal("err is nil")
		}

		errResp, ok := err.(*ErrorResponse)
		if !ok {
			t.Fatalf("err is %T, want %T", err, &ErrorResponse{})
		}

		if got, want := errResp.StatusCode, http.StatusTeapot; got != want {
			t.Errorf("errResp.StatusCode = %d, want %d", got, want)
		}

		if got, want := len(errResp.Errors), 2; got != want {
			t.Fatalf("len(errResp.Errors) = %d, want %d", got, want)
		}

		if got, want := errResp.Errors[0].Message, "error-msg-1"; got != want {
			t.Errorf("errResp.Errors[0].Message = %q, want %q", got, want)
		}

		if got, want := errResp.Errors[1].Message, "error-msg-2"; got != want {
			t.Errorf("errResp.Errors[1].Message = %q, want %q", got, want)
		}
	})

	t.Run("MalformedJSONResponse", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`not JSON`))
			},
		))
		defer ts.Close()

		c := New(ts.URL, &http.Client{})

		err := c.Query(context.Background(), "", nil, nil)

		if err == nil {
			t.Fatal("err is nil")
		}

		if _, ok := err.(*json.SyntaxError); !ok {
			t.Errorf("err is %T, want %T", err, &json.SyntaxError{})
		}
	})
}

func ExampleClient_Query_detailed() {
	mockGraphQLServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"data":{"foo":"bar"}}`))
		},
	))
	defer mockGraphQLServer.Close()

	reqOpt := func(req *http.Request) {
		req.Header.Set("My-Header", "value")
		req.URL.RawQuery = url.Values{"myqueryparam": {"value"}}.Encode()
	}

	query := `query { foo }`

	vars := map[string]interface{}{
		"myvar":      "value",
		"myothervar": 123,
	}

	var data struct {
		Foo string `json:"foo"`
	}

	c := New(mockGraphQLServer.URL, &http.Client{Timeout: 2 * time.Second})

	if err := c.Query(context.Background(), query, vars, &data, reqOpt); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("data.Foo = %q\n", data.Foo)

	// Output:
	// data.Foo = "bar"
}

func ExampleClient_Query_errorResponse() {
	mockGraphQLServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
			w.Write([]byte(`{"errors":[{"message":"error-msg"}]}`))
		},
	))
	defer mockGraphQLServer.Close()

	query := `query { foo }`

	var data interface{}

	c := New(mockGraphQLServer.URL, &http.Client{Timeout: 2 * time.Second})

	if err := c.Query(context.Background(), query, nil, &data); err != nil {
		fmt.Printf("%v\n", err)

		if errResp, ok := err.(*ErrorResponse); ok {
			fmt.Printf("HTTP status: %d\n", errResp.StatusCode)
			fmt.Printf("first error: %q\n", errResp.Errors[0].Message)
		}
	}

	// Output:
	// 418 I'm a teapot: error-msg
	// HTTP status: 418
	// first error: "error-msg"
}

func ExampleClient_Query_simple() {
	mockGraphQLServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"data":{"foo":"bar"}}`))
		},
	))
	defer mockGraphQLServer.Close()

	query := `query { foo }`

	var data struct {
		Foo string `json:"foo"`
	}

	c := New(mockGraphQLServer.URL, &http.Client{Timeout: 2 * time.Second})

	if err := c.Query(context.Background(), query, nil, &data); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("data.Foo = %q\n", data.Foo)

	// Output:
	// data.Foo = "bar"
}

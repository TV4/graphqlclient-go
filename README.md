# graphqlclient-go

[![Build Status](https://travis-ci.com/TV4/graphqlclient-go.svg?branch=master)](https://travis-ci.com/TV4/graphqlclient-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/TV4/graphqlclient-go)](https://goreportcard.com/report/github.com/TV4/graphqlclient-go)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/TV4/graphqlclient-go)
[![License MIT](https://img.shields.io/badge/license-MIT-lightgrey.svg?style=flat)](https://github.com/TV4/graphqlclient-go#license)

`graphqlclient-go` is a Go package that provides boilerplate for interfacing
with a GraphQL server.

## Installation
```
go get -u github.com/TV4/graphqlclient-go
```

## Usage
```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	graphqlclient "github.com/TV4/graphqlclient-go"
)

func main() {
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

	c := graphqlclient.New(mockGraphQLServer.URL, &http.Client{Timeout: 2 * time.Second})

	if err := c.Query(context.Background(), query, nil, &data); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("data.Foo = %q\n", data.Foo)
}
```

```
data.Foo = "bar"
```

## License

Copyright (c) 2018-2020 TV4

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

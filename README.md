# R2

[![GitHub Actions](https://github.com/aofei/r2/workflows/Test/badge.svg)](https://github.com/aofei/r2)
[![codecov](https://codecov.io/gh/aofei/r2/branch/master/graph/badge.svg)](https://codecov.io/gh/aofei/r2)
[![Go Report Card](https://goreportcard.com/badge/github.com/aofei/r2)](https://goreportcard.com/report/github.com/aofei/r2)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/aofei/r2)](https://pkg.go.dev/github.com/aofei/r2)

A minimalist HTTP request routing helper for Go.

The name "R2" stands for "Request Routing". That's all, R2 is just a capable
little helper for HTTP request routing, not another fancy web framework that
wraps [net/http](https://pkg.go.dev/net/http).

R2 is built for people who:

* Think [net/http](https://pkg.go.dev/net/http) is powerful enough and easy to use.
* Don't want to use any web framework that wraps [net/http](https://pkg.go.dev/net/http).
* Don't want to use any variant of [`http.Handler`](https://pkg.go.dev/net/http#Handler).
* Want [`http.ServeMux`](https://pkg.go.dev/net/http#ServeMux) to have better performance and support path parameters.

## Features

* Extremely easy to use
* Blazing fast (see [benchmarks](https://github.com/aofei/go-http-request-routing-benchmark#readme))
* Based on [radix tree](https://en.wikipedia.org/wiki/Radix_tree)
* Sub-router support
* Path parameter support
* No [`http.Handler`](https://pkg.go.dev/net/http#Handler) variant
* Middleware support
* Zero third-party dependencies
* 100% code coverage

## Installation

Open your terminal and execute

```bash
$ go get github.com/aofei/r2
```

done.

> The only requirement is the [Go](https://go.dev), at least v1.13.

## Hello, 世界

Create a file named `hello.go`

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/aofei/r2"
)

func main() {
	r := &r2.Router{}
	r.Handle("", "/hello/:name", http.HandlerFunc(hello))
	http.ListenAndServe("localhost:8080", r)
}

func hello(rw http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(rw, "Hello, %s\n", r2.PathParam(req, "name"))
}
```

and run it

```bash
$ go run hello.go
```

then visit `http://localhost:8080/hello/世界`.

## Community

If you want to discuss R2, or ask questions about it, simply post questions or
ideas [here](https://github.com/aofei/r2/issues).

## Contributing

If you want to help build R2, simply follow
[this](https://github.com/aofei/r2/wiki/Contributing) to send pull requests
[here](https://github.com/aofei/r2/pulls).

## License

This project is licensed under the MIT License.

License can be found [here](LICENSE).

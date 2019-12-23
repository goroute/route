<div align="center">
    
[![Build Status](https://travis-ci.com/goroute/route.svg?branch=master)](https://travis-ci.com/goroute/route)
[![codecov](https://codecov.io/gh/goroute/route/branch/master/graph/badge.svg)](https://codecov.io/gh/goroute/route) 
[![GoDoc](https://godoc.org/github.com/goroute/route?status.svg)](http://godoc.org/github.com/goroute/route) 
[![Go Report Card](https://goreportcard.com/badge/github.com/goroute/route)](https://goreportcard.com/report/github.com/goroute/route)

</div>

## Few main features

* Minimal core.
* No external runtime dependencies. Custom middlewares which requires 3th party dependecies are places in separates repositories under goroute org.
* HTTP Routing.
* Middlewares support.
* Global error handling.

## Getting Started

### Prerequisites

You need to have at least go 1.11 installed on you local machine.

### Installing

Install go route package with go get

```
go get -u github.com/goroute/route
```

Start your first server. Create main.go file and add:
```go
package main

import (
    "net/http"
    "log"
    "github.com/goroute/route"
)

type helloResponse struct {
	Title string `json:"title"`
}

func main() {
	mux := route.NewServeMux()
	
	mux.Use(func(c route.Context, next route.HandlerFunc) error {
	    log.Println("Hello, Middleware!")
	    return next(c)
	})
	
	mux.GET("/", func(c route.Context) error {
	    return c.JSON(http.StatusOK, &helloResponse{Title:"Hello, JSON!"})
	})
	
	log.Fatal(http.ListenAndServe(":9000", mux))
}

```

Run it

```
go run main.go
```

## More examples

See [examples](https://github.com/goroute/route/tree/master/examples)

## Built With

* [Go](https://www.golang.org/)

## Contributing

Please read [CONTRIBUTING.md](https://github.com/goroute/route/CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/goroute/route/tags). 

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE) file for details

## Acknowledgments

* This project is largely inspired by [echo](https://echo.labstack.com/). Parts of the code are adopted from echo. See NOTICE. 

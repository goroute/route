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
* "Correct" handler func signature.

## Why goroute

There are a lot of good web frameworks / routers for go already so why created yet another one. Few reasons:

### 1. Imperfect handler signature

Routers like httprouter, gorilla etc. mux are trying to be compliant with http standard library. This is a good goal but http standard library isn't ideal. Don't get me wrong. Go has one of the best http standard libraries but I found it be too low level for 90% of use cases. Let's look at the example:

Go http standard library has a HandleFunc method which allows to execute handlers based on a path. If you look at function signature it allows to get the request from http.Request and write response using http.ResponseWriter.

```go
http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello")
})
```

Since it doesn't force to return error we should terminate execution with explicit return statement. We also need to think how to log error and it's quite hard to have global error handling in one place.

```go
http.HandleFunc("/users", func (w http.ResponseWriter, r *http.Request) {
	users, err := db.GetUsers()
	if err != nil {
		w.WriteHeader(http.StatusInternalError)
		log.Errorf("cannot get users: %v", err)
		return
	}
	
	usersJSON, err := json.Marshal(users)
	if err != nil {
		w.WriteHeader(http.StatusInternalError)
		log.Errorf("cannot marshal users to json: %v", err)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	w.Write(usersJSON)
})
```

Similar handler signatures like `func (w http.ResponseWriter, r *http.Request)`, `func (ctx framework.Context)` can be found in 90% of all go web frameworks / routers.

Let's look how can we improve this but changing function signature.

```go
mux := route.NewServeMux()
mux.GET("/users", func (c route.Context) error {
	return c.String(http.StatusOK, "Hello")
})
```

The same use case with error handling.

```go
mux := route.NewServeMux()
mux.GET("/users", func (c route.Context) error {
	users, err := db.GetUsers()
	if err != nil {
		return fmt.Errorf("cannot get users: %v", err")
	}
	return c.JSON(http.StatusOK, users)
})
```

By simply forcing to return error we now have much cleaner code which not only make it more clear and less boilerplate but also allows to have much simpler global error handling implementation. 

P.S If you look at grpc go it has similar handlers signature which allows returns error.

```go
func (s *server) GetUsers(ctx context.Context, in *pb.GetUsersRequest) (*pb.GetUsersReply, error) {
	users, err := s.db.GetUsers()
	if err != nil {
		return fmt.Errorf("cannot get users: %v", err")
	}
	return &pb.GetUsersReply{Users: users}, nil
}
```

### 2. Being minimalist but depending on external libraries

There is one great web framework called [echo](https://echo.labstack.com/). I used it for my few first projects and really liked it. But... while echo calls itself minimalist web framework it depends on 3th party libraries like `github.com/dgrijalva/jwt-go`, `github.com/valyala/fasttemplate`, `golang.org/x/crypto`. Lets look why.

1. github.com/dgrijalva/jwt-go is used for jwt middleware which is placed under middleware/jwt.go. This means that you automatically forced to add this dependency even if you are not using jwt middleware.

2. github.com/valyala/fasttemplate is used for logging templates which is built into the framework.

3. golang.org/x/crypto is used for auto HTTPS via Let's Encrypt. What if I'm running my web service in kubernetes or docker swarm and I have cluster level load balancer which terminates HTTPS traffic? If you need https on lowest level you can create it by using golang.org/x/crypto/acme/autocert package in few lines of code.

How can we solve this?

Goroute github.com organization is structured in a way that each custom middleware is placed in it's own repo with it's own 3th party dependencies. Such approach not only allows to have separate versioning but also keeps main goroute [route](https://github.com/goroute/route) small and minimalist.

### 3. Seeking "best" performance

You may notice that there is a trend for go web frameworks developers to advertise how their frameworks are faster than others and they are playing hello world benchmark battles. What does it mean to have high performance? In real world services your web api endpoint is probably going to call database which will bottleneck first. If not then simply scale horizontally and you are good to go. If you really have a huge load since you working on ads serving etc. then even standard underlying go http primitives are not going to help and you are probably already switched to some custom http/tcp implementations or already using Rust.

Goroute tries to be minimal and uses standard go http server by implementing ServeHTTP interface. There are some simple optimizations like sync.Pool for having less allocations but nothing that feels hacky or over-engineered.

### 4. Overengineering

Session control, dependency injection, caching, logging, configuration parsing, automatic HTTPS, performance supervising, context handling, ORM supporting, requests simulating, Webassembly, MVC, sessions, caching, Websocket.

If you need all these features in one place then visit [beego](https://github.com/astaxie/beego) or [iris](github.com/kataras/iris) and you are well covered. Well.. maybe not. I was surprised when I saw that beego has builtin orm for SQL and popular databases like MySQL, Postgres etc. If you writing boring CRUD you may want to us ORM for sure, but there are event better packages for that like [gorm](https://github.com/jinzhu/gorm). For one it may sound awesome but it really isn't. What if I'm using NoSQL database like MongoDB or googles DataStore? When I don't need traditional ORM at all.


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

package route

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: Fix me
func TestGroup(t *testing.T) {
	g := NewServeMux().Group("/group")
	h := func(Context) error { return nil }
	g.CONNECT("/", h)
	g.DELETE("/", h)
	g.GET("/", h)
	g.HEAD("/", h)
	g.OPTIONS("/", h)
	g.PATCH("/", h)
	g.POST("/", h)
	g.PUT("/", h)
	g.TRACE("/", h)
	g.Any("/", h)
	g.Match([]string{http.MethodGet, http.MethodPost}, "/", h)
	g.Static("/static", "/tmp")
	g.File("/walle", "../testdata/images//walle.png")
}

func TestGroupRouteMiddleware(t *testing.T) {
	// Ensure middleware slices are not re-used
	e := NewServeMux()
	g := e.Group("/group")
	h := func(Context) error { return nil }
	m1 := func(c Context, next HandlerFunc) error {
		return next(c)
	}
	m2 := func(c Context, next HandlerFunc) error {
		return next(c)
	}
	m3 := func(c Context, next HandlerFunc) error {
		return next(c)
	}
	m4 := func(c Context, next HandlerFunc) error {
		return c.NoContent(404)
	}
	m5 := func(c Context, next HandlerFunc) error {
		return c.NoContent(405)
	}
	g.Use(m1, m2, m3)
	g.GET("/404", h, m4)
	g.GET("/405", h, m5)

	c, _ := request(http.MethodGet, "/group/404", e)
	assert.Equal(t, 404, c)
	c, _ = request(http.MethodGet, "/group/405", e)
	assert.Equal(t, 405, c)
}

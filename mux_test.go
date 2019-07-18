package route

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type (
	user struct {
		ID   int    `json:"id" xml:"id" form:"id" query:"id"`
		Name string `json:"name" xml:"name" form:"name" query:"name"`
	}
)

const (
	userJSON            = `{"id":1,"name":"Jon Snow"}`
	userForm            = `id=1&name=Jon Snow`
	invalidContent      = "invalid content"
	userJSONInvalidType = `{"id":"1","name":"Jon Snow"}`
)

const userJSONPretty = `{
  "id": 1,
  "name": "Jon Snow"
}`

func TestMux(t *testing.T) {
	mux := NewServeMux()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := mux.NewContext(req, rec)

	mux.defaultHTTPErrorHandler(errors.New("error"), c)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestMuxStatic(t *testing.T) {
	mux := NewServeMux()

	assert := assert.New(t)

	// OK
	mux.Static("/images", "testdata/images")
	c, b := request(http.MethodGet, "/images/walle.png", mux)
	assert.Equal(http.StatusOK, c)
	assert.NotEmpty(b)

	// No file
	mux.Static("/images", "testdata/scripts")
	c, _ = request(http.MethodGet, "/images/bolt.png", mux)
	assert.Equal(http.StatusNotFound, c)

	// Directory
	mux.Static("/images", "testdata/images")
	c, _ = request(http.MethodGet, "/images", mux)
	assert.Equal(http.StatusNotFound, c)

	// Directory with index.html
	mux.Static("/", "testdata")
	c, r := request(http.MethodGet, "/", mux)
	assert.Equal(http.StatusOK, c)
	assert.Equal(true, strings.HasPrefix(r, "<!doctype html>"))

	// Sub-directory with index.html
	c, r = request(http.MethodGet, "/folder", mux)
	assert.Equal(http.StatusOK, c)
	assert.Equal(true, strings.HasPrefix(r, "<!doctype html>"))
}

func TestMuxWithOptions(t *testing.T) {
	binder := &mockBinder{}
	renderer := &mockRenderer{}
	mockHTTPErrorHandler := func(error, Context) {
	}

	mux := NewServeMux(
		WithBinder(binder),
		WithRenderer(renderer),
		WithHTTPErrorHandler(mockHTTPErrorHandler),
	)

	assert.Equal(t, binder, mux.binder)
	assert.Equal(t, renderer, mux.renderer)
	assert.NotNil(t, mux.httpErrorHandler)
}

func TestMuxFile(t *testing.T) {
	mux := NewServeMux()
	mux.File("/walle", "testdata/images/walle.png")
	c, b := request(http.MethodGet, "/walle", mux)
	assert.Equal(t, http.StatusOK, c)
	assert.NotEmpty(t, b)
}

func TestMuxMiddleware(t *testing.T) {
	mux := NewServeMux()
	buf := new(bytes.Buffer)

	mux.Pre(func(c Context, next HandlerFunc) error {
		assert.Empty(t, c.Path())
		buf.WriteString("-1")
		return next(c)
	})

	mux.Use(func(c Context, next HandlerFunc) error {
		buf.WriteString("1")
		return next(c)
	})

	mux.Use(func(c Context, next HandlerFunc) error {
		buf.WriteString("2")
		return next(c)
	})

	mux.Use(func(c Context, next HandlerFunc) error {
		buf.WriteString("3")
		return next(c)
	})

	// Route
	mux.GET("/", func(c Context) error {
		return c.String(http.StatusOK, "OK")
	})

	c, b := request(http.MethodGet, "/", mux)

	assert.Equal(t, "-1123", buf.String())
	assert.Equal(t, http.StatusOK, c)
	assert.Equal(t, "OK", b)
}

func TestMuxMiddlewareError(t *testing.T) {
	mux := NewServeMux()
	buf := new(bytes.Buffer)

	mux.Use(func(c Context, next HandlerFunc) error {
		buf.WriteString("1")
		return next(c)
	})

	mux.Use(func(c Context, next HandlerFunc) error {
		buf.WriteString("2")
		return errors.New("error")
	})

	mux.Use(func(c Context, next HandlerFunc) error {
		buf.WriteString("3")
		return next(c)
	})

	mux.GET("/", NotFoundHandler)
	c, _ := request(http.MethodGet, "/", mux)

	assert.Equal(t, "12", buf.String())
	assert.Equal(t, http.StatusInternalServerError, c)
}

func TestMuxHandler(t *testing.T) {
	mux := NewServeMux()

	// HandlerFunc
	mux.GET("/ok", func(c Context) error {
		return c.String(http.StatusOK, "OK")
	})

	c, b := request(http.MethodGet, "/ok", mux)
	assert.Equal(t, http.StatusOK, c)
	assert.Equal(t, "OK", b)
}

func TestMuxWrapHandler(t *testing.T) {
	mux := NewServeMux()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := mux.NewContext(req, rec)
	h := WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))
	if assert.NoError(t, h(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "test", rec.Body.String())
	}
}

func TestMuxConnect(t *testing.T) {
	mux := NewServeMux()
	testMethod(t, http.MethodConnect, "/", mux)
}

func TestMuxDelete(t *testing.T) {
	mux := NewServeMux()
	testMethod(t, http.MethodDelete, "/", mux)
}

func TestMuxGet(t *testing.T) {
	mux := NewServeMux()
	testMethod(t, http.MethodGet, "/", mux)
}

func TestMuxHead(t *testing.T) {
	mux := NewServeMux()
	testMethod(t, http.MethodHead, "/", mux)
}

func TestMuxOptions(t *testing.T) {
	mux := NewServeMux()
	testMethod(t, http.MethodOptions, "/", mux)
}

func TestMuxPatch(t *testing.T) {
	mux := NewServeMux()
	testMethod(t, http.MethodPatch, "/", mux)
}

func TestMuxPost(t *testing.T) {
	mux := NewServeMux()
	testMethod(t, http.MethodPost, "/", mux)
}

func TestMuxPut(t *testing.T) {
	mux := NewServeMux()
	testMethod(t, http.MethodPut, "/", mux)
}

func TestMuxTrace(t *testing.T) {
	mux := NewServeMux()
	testMethod(t, http.MethodTrace, "/", mux)
}

func TestMuxAny(t *testing.T) { // JFC
	mux := NewServeMux()
	mux.Any("/", func(c Context) error {
		return c.String(http.StatusOK, "Any")
	})
}

func TestMuxMatch(t *testing.T) { // JFC
	mux := NewServeMux()
	mux.Match([]string{http.MethodGet, http.MethodPost}, "/", func(c Context) error {
		return c.String(http.StatusOK, "Match")
	})
}

func TestMuxRoutes(t *testing.T) {
	mux := NewServeMux()
	routes := []*Route{
		{http.MethodGet, "/users/:user/events", ""},
		{http.MethodGet, "/users/:user/events/public", ""},
		{http.MethodPost, "/repos/:owner/:repo/git/refs", ""},
		{http.MethodPost, "/repos/:owner/:repo/git/tags", ""},
	}
	for _, r := range routes {
		mux.Add(r.Method, r.Path, func(c Context) error {
			return c.String(http.StatusOK, "OK")
		})
	}

	if assert.Equal(t, len(routes), len(mux.Routes())) {
		for _, r := range mux.Routes() {
			found := false
			for _, rr := range routes {
				if r.Method == rr.Method && r.Path == rr.Path {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Route %s %s not found", r.Method, r.Path)
			}
		}
	}
}

func TestMuxEncodedPath(t *testing.T) {
	mux := NewServeMux()
	mux.GET("/:id", func(c Context) error {
		return c.NoContent(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/with%2Fslash", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMuxGroup(t *testing.T) {
	mux := NewServeMux()
	buf := new(bytes.Buffer)
	mux.Use(MiddlewareFunc(func(c Context, next HandlerFunc) error {
		buf.WriteString("0")
		return next(c)
	}))
	h := func(c Context) error {
		return c.NoContent(http.StatusOK)
	}

	//--------
	// Routes
	//--------

	mux.GET("/users", h)

	// Group
	g1 := mux.Group("/group1")
	g1.Use(func(c Context, next HandlerFunc) error {
		buf.WriteString("1")
		return next(c)
	})
	g1.GET("", h)

	// Nested groups with middleware
	g2 := mux.Group("/group2")
	g2.Use(func(c Context, next HandlerFunc) error {
		buf.WriteString("2")
		return next(c)
	})
	g3 := g2.Group("/group3")
	g3.Use(func(c Context, next HandlerFunc) error {
		buf.WriteString("3")
		return next(c)
	})
	g3.GET("", h)

	request(http.MethodGet, "/users", mux)
	assert.Equal(t, "0", buf.String())

	buf.Reset()
	request(http.MethodGet, "/group1", mux)
	assert.Equal(t, "01", buf.String())

	buf.Reset()
	request(http.MethodGet, "/group2/group3", mux)
	assert.Equal(t, "023", buf.String())
}

func TestMuxNotFound(t *testing.T) {
	mux := NewServeMux()
	req := httptest.NewRequest(http.MethodGet, "/files", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestMuxMethodNotAllowed(t *testing.T) {
	mux := NewServeMux()
	mux.GET("/", func(c Context) error {
		return c.String(http.StatusOK, "Mux!")
	})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestMuxContext(t *testing.T) {
	mux := NewServeMux()
	c := mux.pool.Get().(*context)
	assert.IsType(t, new(context), c)
	mux.pool.Put(c)
}

func TestMuxStart(t *testing.T) {
	mux := NewServeMux()
	go func() {
		err := http.ListenAndServe(":0", mux)
		assert.NoError(t, err)
	}()
	time.Sleep(200 * time.Millisecond)
}

func TestMuxStartTLS(t *testing.T) {
	mux := NewServeMux()
	go func() {
		err := http.ListenAndServeTLS(":0", "testdata/certs/cert.pem", "testdata/certs/key.pem", mux)
		// Prevent the test to fail after closing the servers
		if err != http.ErrServerClosed {
			assert.NoError(t, err)
		}
	}()
	time.Sleep(200 * time.Millisecond)
}

func testMethod(t *testing.T, method, path string, e *Mux) {
	p := reflect.ValueOf(path)
	h := reflect.ValueOf(func(c Context) error {
		return c.String(http.StatusOK, method)
	})
	i := interface{}(e)
	reflect.ValueOf(i).MethodByName(method).Call([]reflect.Value{p, h})
	_, body := request(method, path, e)
	assert.Equal(t, method, body)
}

func request(method, path string, e *Mux) (int, string) {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func TestHTTPError(t *testing.T) {
	err := NewHTTPError(http.StatusBadRequest, map[string]interface{}{
		"code": 12,
	})
	assert.Equal(t, "code=400, message=map[code:12]", err.Error())
}

type mockBinder struct{}

func (mockBinder) Bind(i interface{}, c Context) error {
	return nil
}

type mockRenderer struct{}

func (mockRenderer) Render(io.Writer, string, interface{}, Context) error {
	return nil
}

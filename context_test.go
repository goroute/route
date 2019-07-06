package route

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
)

type (
	Template struct {
		templates *template.Template
	}
)

func (t *Template) Render(w io.Writer, name string, data interface{}, c Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func TestContext(t *testing.T) {
	e := NewServeMux()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(userJSON))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec).(*context)

	assert := assert.New(t)

	// Request
	assert.NotNil(c.Request())

	// Response
	assert.NotNil(c.Response())

	//--------
	// Render
	//--------

	tmpl := &Template{
		templates: template.Must(template.New("hello").Parse("Hello, {{.}}!")),
	}
	c.mux.renderer = tmpl
	err := c.Render(http.StatusOK, "hello", "Jon Snow")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("Hello, Jon Snow!", rec.Body.String())
	}

	c.mux.renderer = nil
	err = c.Render(http.StatusOK, "hello", "Jon Snow")
	assert.Error(err)

	// JSON
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.JSON(http.StatusOK, user{1, "Jon Snow"})
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMEApplicationJSONCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(userJSON, rec.Body.String())
	}

	// JSON with "?pretty"
	req = httptest.NewRequest(http.MethodGet, "/?pretty", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.JSON(http.StatusOK, user{1, "Jon Snow"})
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMEApplicationJSONCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(userJSONPretty, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/", nil) // reset

	// jsonPretty
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.jsonPretty(http.StatusOK, user{1, "Jon Snow"}, "  ")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMEApplicationJSONCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(userJSONPretty, rec.Body.String())
	}

	// JSON (error)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.JSON(http.StatusOK, make(chan bool))
	assert.Error(err)

	// String
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.String(http.StatusOK, "Hello, World!")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMETextPlainCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal("Hello, World!", rec.Body.String())
	}

	// HTML
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.HTML(http.StatusOK, "Hello, <strong>World!</strong>")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal(MIMETextHTMLCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal("Hello, <strong>World!</strong>", rec.Body.String())
	}

	// Stream
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	r := strings.NewReader("response from a stream")
	err = c.Stream(http.StatusOK, "application/octet-stream", r)
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("application/octet-stream", rec.Header().Get(HeaderContentType))
		assert.Equal("response from a stream", rec.Body.String())
	}

	// Attachment
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.Attachment("testdata/images/walle.png", "walle.png")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("attachment; filename=\"walle.png\"", rec.Header().Get(HeaderContentDisposition))
		assert.Equal(219885, rec.Body.Len())
	}

	// Inline
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	err = c.Inline("testdata/images/walle.png", "walle.png")
	if assert.NoError(err) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("inline; filename=\"walle.png\"", rec.Header().Get(HeaderContentDisposition))
		assert.Equal(219885, rec.Body.Len())
	}

	// NoContent
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	c.NoContent(http.StatusOK)
	assert.Equal(http.StatusOK, rec.Code)

	// Error
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec).(*context)
	c.Error(errors.New("error"))
	assert.Equal(http.StatusInternalServerError, rec.Code)

	// Reset
	c.SetParamNames("foo")
	c.SetParamValues("bar")
	c.Set("foe", "ban")
	c.query = url.Values(map[string][]string{"fon": {"baz"}})
	c.reset(req, httptest.NewRecorder())
	assert.Equal(0, len(c.ParamValues()))
	assert.Equal(0, len(c.ParamNames()))
	assert.Equal(0, len(c.store))
	assert.Equal("", c.Path())
	assert.Equal(0, len(c.QueryParams()))
}

func TestContextCookie(t *testing.T) {
	e := NewServeMux()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	theme := "theme=light"
	user := "user=Jon Snow"
	req.Header.Add(HeaderCookie, theme)
	req.Header.Add(HeaderCookie, user)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec).(*context)

	assert := assert.New(t)

	// Read single
	cookie, err := c.Cookie("theme")
	if assert.NoError(err) {
		assert.Equal("theme", cookie.Name)
		assert.Equal("light", cookie.Value)
	}

	// Read multiple
	for _, cookie := range c.Cookies() {
		switch cookie.Name {
		case "theme":
			assert.Equal("light", cookie.Value)
		case "user":
			assert.Equal("Jon Snow", cookie.Value)
		}
	}

	// Write
	cookie = &http.Cookie{
		Name:     "SSID",
		Value:    "Ap4PGTEq",
		Domain:   "dostack.com",
		Path:     "/",
		Expires:  time.Now(),
		Secure:   true,
		HttpOnly: true,
	}
	c.SetCookie(cookie)
	assert.Contains(rec.Header().Get(HeaderSetCookie), "SSID")
	assert.Contains(rec.Header().Get(HeaderSetCookie), "Ap4PGTEq")
	assert.Contains(rec.Header().Get(HeaderSetCookie), "dostack.com")
	assert.Contains(rec.Header().Get(HeaderSetCookie), "Secure")
	assert.Contains(rec.Header().Get(HeaderSetCookie), "HttpOnly")
}

func TestContextPath(t *testing.T) {
	e := NewServeMux()

	e.Add(http.MethodGet, "/users/:id", nil)
	c := e.NewContext(nil, nil)
	e.router.find(http.MethodGet, "/users/1", c)

	assert := assert.New(t)

	assert.Equal("/users/:id", c.Path())

	e.Add(http.MethodGet, "/users/:uid/files/:fid", nil)
	c = e.NewContext(nil, nil)
	e.router.find(http.MethodGet, "/users/1/files/1", c)
	assert.Equal("/users/:uid/files/:fid", c.Path())
}

func TestContextPathParam(t *testing.T) {
	e := NewServeMux()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, nil)

	// ParamNames
	c.SetParamNames("uid", "fid")
	assert.EqualValues(t, []string{"uid", "fid"}, c.ParamNames())

	// ParamValues
	c.SetParamValues("101", "501")
	assert.EqualValues(t, []string{"101", "501"}, c.ParamValues())

	// Param
	assert.Equal(t, "501", c.Param("fid"))
}

func TestContextFormValue(t *testing.T) {
	f := make(url.Values)
	f.Set("name", "Jon Snow")
	f.Set("email", "jon@dostack.com")

	e := NewServeMux()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(f.Encode()))
	req.Header.Add(HeaderContentType, MIMEApplicationForm)
	c := e.NewContext(req, nil)

	// FormValue
	assert.Equal(t, "Jon Snow", c.FormValue("name"))
	assert.Equal(t, "jon@dostack.com", c.FormValue("email"))

	// FormParams
	params, err := c.FormParams()
	if assert.NoError(t, err) {
		assert.Equal(t, url.Values{
			"name":  []string{"Jon Snow"},
			"email": []string{"jon@dostack.com"},
		}, params)
	}
}

func TestContextQueryParam(t *testing.T) {
	q := make(url.Values)
	q.Set("name", "Jon Snow")
	q.Set("email", "jon@dostack.com")
	req := httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
	e := NewServeMux()
	c := e.NewContext(req, nil)

	// QueryParam
	assert.Equal(t, "Jon Snow", c.QueryParam("name"))
	assert.Equal(t, "jon@dostack.com", c.QueryParam("email"))

	// QueryParams
	assert.Equal(t, url.Values{
		"name":  []string{"Jon Snow"},
		"email": []string{"jon@dostack.com"},
	}, c.QueryParams())
}

func TestContextFormFile(t *testing.T) {
	e := NewServeMux()
	buf := new(bytes.Buffer)
	mr := multipart.NewWriter(buf)
	w, err := mr.CreateFormFile("file", "test")
	if assert.NoError(t, err) {
		w.Write([]byte("test"))
	}
	mr.Close()
	req := httptest.NewRequest(http.MethodPost, "/", buf)
	req.Header.Set(HeaderContentType, mr.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	f, err := c.FormFile("file")
	if assert.NoError(t, err) {
		assert.Equal(t, "test", f.Filename)
	}
}

func TestContextMultipartForm(t *testing.T) {
	e := NewServeMux()
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	mw.WriteField("name", "Jon Snow")
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/", buf)
	req.Header.Set(HeaderContentType, mw.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	f, err := c.MultipartForm()
	if assert.NoError(t, err) {
		assert.NotNil(t, f)
	}
}

func TestContextRedirect(t *testing.T) {
	e := NewServeMux()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	assert.Equal(t, nil, c.Redirect(http.StatusMovedPermanently, "http://dostack.github.io/mux"))
	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	assert.Equal(t, "http://dostack.github.io/mux", rec.Header().Get(HeaderLocation))
	assert.Error(t, c.Redirect(310, "http://dostack.github.io/mux"))
}

func TestContextStore(t *testing.T) {
	var c Context
	c = new(context)
	c.Set("name", "Jon Snow")
	assert.Equal(t, "Jon Snow", c.Get("name"))
}

func TestContextHandler(t *testing.T) {
	e := NewServeMux()
	b := new(bytes.Buffer)

	e.Add(http.MethodGet, "/handler", func(Context) error {
		_, err := b.Write([]byte("handler"))
		return err
	})
	c := e.NewContext(nil, nil)
	e.router.find(http.MethodGet, "/handler", c)
	c.Handler()(c)
	assert.Equal(t, "handler", b.String())
}

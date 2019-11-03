// Package route implements HTTP request miltiplexer.
package route

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
)

type (
	// Mux is the top-level framework instance.
	Mux struct {
		premiddleware   []MiddlewareFunc
		middleware      []MiddlewareFunc
		maxParam        *int
		router          *router
		notFoundHandler HandlerFunc
		pool            sync.Pool

		Debug            bool
		HTTPErrorHandler HTTPErrorHandler
		Binder           Binder
		Renderer         Renderer
	}

	// Route contains a handler and information for matching against requests.
	Route struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Name   string `json:"name"`
	}

	// HTTPError represents an error that occurred while handling a request.
	HTTPError struct {
		Code     int
		Message  interface{}
		Internal error // Stores the error returned by an external dependency
	}

	// HandlerFunc defines a function to serve HTTP requests.
	HandlerFunc func(Context) error

	// HTTPErrorHandler is a centralized HTTP error handler.
	HTTPErrorHandler func(error, Context)

	// Renderer is the interface that wraps the Render function.
	Renderer interface {
		Render(io.Writer, string, interface{}, Context) error
	}

	// i is the interface for Mux and Group.
	i interface {
		GET(string, HandlerFunc, ...MiddlewareFunc) *Route
	}
)

// MIME types
const (
	MIMEApplicationJSON                  = "application/json"
	MIMEApplicationJSONCharsetUTF8       = MIMEApplicationJSON + "; " + charsetUTF8
	MIMEApplicationJavaScript            = "application/javascript"
	MIMEApplicationJavaScriptCharsetUTF8 = MIMEApplicationJavaScript + "; " + charsetUTF8
	MIMEApplicationXML                   = "application/xml"
	MIMEApplicationXMLCharsetUTF8        = MIMEApplicationXML + "; " + charsetUTF8
	MIMETextXML                          = "text/xml"
	MIMETextXMLCharsetUTF8               = MIMETextXML + "; " + charsetUTF8
	MIMEApplicationForm                  = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf              = "application/protobuf"
	MIMEApplicationMsgpack               = "application/msgpack"
	MIMETextHTML                         = "text/html"
	MIMETextHTMLCharsetUTF8              = MIMETextHTML + "; " + charsetUTF8
	MIMETextPlain                        = "text/plain"
	MIMETextPlainCharsetUTF8             = MIMETextPlain + "; " + charsetUTF8
	MIMEMultipartForm                    = "multipart/form-data"
	MIMEOctetStream                      = "application/octet-stream"
)

const (
	charsetUTF8 = "charset=UTF-8"
	// PROPFIND HTTP Header
	PROPFIND = "PROPFIND"
)

// Headers
const (
	HeaderAccept              = "Accept"
	HeaderAcceptEncoding      = "Accept-Encoding"
	HeaderAllow               = "Allow"
	HeaderAuthorization       = "Authorization"
	HeaderContentDisposition  = "Content-Disposition"
	HeaderContentEncoding     = "Content-Encoding"
	HeaderContentLength       = "Content-Length"
	HeaderContentType         = "Content-Type"
	HeaderCookie              = "Cookie"
	HeaderSetCookie           = "Set-Cookie"
	HeaderIfModifiedSince     = "If-Modified-Since"
	HeaderLastModified        = "Last-Modified"
	HeaderLocation            = "Location"
	HeaderUpgrade             = "Upgrade"
	HeaderVary                = "Vary"
	HeaderWWWAuthenticate     = "WWW-Authenticate"
	HeaderXForwardedFor       = "X-Forwarded-For"
	HeaderXForwardedProto     = "X-Forwarded-Proto"
	HeaderXForwardedProtocol  = "X-Forwarded-Protocol"
	HeaderXForwardedSsl       = "X-Forwarded-Ssl"
	HeaderXUrlScheme          = "X-Url-Scheme"
	HeaderXHTTPMethodOverride = "X-HTTP-Method-Override"
	HeaderXRealIP             = "X-Real-IP"
	HeaderXRequestID          = "X-Request-ID"
	HeaderXRequestedWith      = "X-Requested-With"
	HeaderServer              = "Server"
	HeaderOrigin              = "Origin"

	// Access control
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"

	// Security
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXXSSProtection          = "X-XSS-Protection"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderXCSRFToken              = "X-CSRF-Token"
)

var (
	methods = [...]string{
		http.MethodConnect,
		http.MethodDelete,
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPost,
		PROPFIND,
		http.MethodPut,
		http.MethodTrace,
	}
)

// Errors
var (
	ErrUnsupportedMediaType        = NewHTTPError(http.StatusUnsupportedMediaType)
	ErrNotFound                    = NewHTTPError(http.StatusNotFound)
	ErrUnauthorized                = NewHTTPError(http.StatusUnauthorized)
	ErrForbidden                   = NewHTTPError(http.StatusForbidden)
	ErrMethodNotAllowed            = NewHTTPError(http.StatusMethodNotAllowed)
	ErrStatusRequestEntityTooLarge = NewHTTPError(http.StatusRequestEntityTooLarge)
	ErrTooManyRequests             = NewHTTPError(http.StatusTooManyRequests)
	ErrBadRequest                  = NewHTTPError(http.StatusBadRequest)
	ErrBadGateway                  = NewHTTPError(http.StatusBadGateway)
	ErrInternalServerError         = NewHTTPError(http.StatusInternalServerError)
	ErrRequestTimeout              = NewHTTPError(http.StatusRequestTimeout)
	ErrServiceUnavailable          = NewHTTPError(http.StatusServiceUnavailable)
	ErrValidatorNotRegistered      = errors.New("validator not registered")
	ErrRendererNotRegistered       = errors.New("Renderer not registered")
	ErrInvalidRedirectCode         = errors.New("invalid redirect status code")
	ErrCookieNotFound              = errors.New("cookie not found")
)

// Error handlers
var (
	NotFoundHandler = func(c Context) error {
		return ErrNotFound
	}

	MethodNotAllowedHandler = func(c Context) error {
		return ErrMethodNotAllowed
	}
)

type options struct {
	binder           Binder
	renderer         Renderer
	httpErrorHandler HTTPErrorHandler
}

// A Option sets options such as credentials, tls, etc.
type Option func(*options)

// WithBinder allows to override default mux Binder.
func WithBinder(binder Binder) Option {
	return func(o *options) {
		o.binder = binder
	}
}

// WithRenderer allows to register mux view Renderer.
func WithRenderer(renderer Renderer) Option {
	return func(o *options) {
		o.renderer = renderer
	}
}

// WithHTTPErrorHandler allows to override default mux global error handler.
func WithHTTPErrorHandler(handler HTTPErrorHandler) Option {
	return func(o *options) {
		o.httpErrorHandler = handler
	}
}

// NewServeMux creates an instance of mux.
func NewServeMux(opt ...Option) (e *Mux) {
	opts := options{
		binder:   &DefaultBinder{},
		renderer: nil,
	}
	for _, o := range opt {
		o(&opts)
	}

	e = &Mux{
		maxParam: new(int),
		Binder:   opts.binder,
		Renderer: opts.renderer,
	}

	// http error handler must be set after mux instance.
	if opts.httpErrorHandler != nil {
		e.HTTPErrorHandler = opts.httpErrorHandler
	} else {
		e.HTTPErrorHandler = e.defaultHTTPErrorHandler
	}

	e.pool.New = func() interface{} {
		return e.NewContext(nil, nil)
	}
	e.router = newRouter(e)
	return
}

// NewContext returns a Context instance.
func (mux *Mux) NewContext(r *http.Request, w http.ResponseWriter) Context {
	return &context{
		request:  r,
		response: NewResponse(w),
		store:    make(map[string]interface{}),
		mux:      mux,
		pvalues:  make([]string, *mux.maxParam),
		handler:  NotFoundHandler,
	}
}

// Pre adds middleware to the chain which is run before router.
func (mux *Mux) Pre(middleware ...MiddlewareFunc) {
	mux.premiddleware = append(mux.premiddleware, middleware...)
}

// Use adds middleware to the chain which is run after router.
func (mux *Mux) Use(middleware ...MiddlewareFunc) {
	mux.middleware = append(mux.middleware, middleware...)
}

// CONNECT registers a new CONNECT route for a path with matching handler in the
// router with optional route-level middleware.
func (mux *Mux) CONNECT(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return mux.Add(http.MethodConnect, path, h, m...)
}

// DELETE registers a new DELETE route for a path with matching handler in the router
// with optional route-level middleware.
func (mux *Mux) DELETE(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return mux.Add(http.MethodDelete, path, h, m...)
}

// GET registers a new GET route for a path with matching handler in the router
// with optional route-level middleware.
func (mux *Mux) GET(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return mux.Add(http.MethodGet, path, h, m...)
}

// HEAD registers a new HEAD route for a path with matching handler in the
// router with optional route-level middleware.
func (mux *Mux) HEAD(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return mux.Add(http.MethodHead, path, h, m...)
}

// OPTIONS registers a new OPTIONS route for a path with matching handler in the
// router with optional route-level middleware.
func (mux *Mux) OPTIONS(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return mux.Add(http.MethodOptions, path, h, m...)
}

// PATCH registers a new PATCH route for a path with matching handler in the
// router with optional route-level middleware.
func (mux *Mux) PATCH(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return mux.Add(http.MethodPatch, path, h, m...)
}

// POST registers a new POST route for a path with matching handler in the
// router with optional route-level middleware.
func (mux *Mux) POST(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return mux.Add(http.MethodPost, path, h, m...)
}

// PUT registers a new PUT route for a path with matching handler in the
// router with optional route-level middleware.
func (mux *Mux) PUT(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return mux.Add(http.MethodPut, path, h, m...)
}

// TRACE registers a new TRACE route for a path with matching handler in the
// router with optional route-level middleware.
func (mux *Mux) TRACE(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return mux.Add(http.MethodTrace, path, h, m...)
}

// Any registers a new route for all HTTP methods and path with matching handler
// in the router with optional route-level middleware.
func (mux *Mux) Any(path string, handler HandlerFunc, middleware ...MiddlewareFunc) []*Route {
	routes := make([]*Route, len(methods))
	for i, m := range methods {
		routes[i] = mux.Add(m, path, handler, middleware...)
	}
	return routes
}

// Match registers a new route for multiple HTTP methods and path with matching
// handler in the router with optional route-level middleware.
func (mux *Mux) Match(methods []string, path string, handler HandlerFunc, middleware ...MiddlewareFunc) []*Route {
	routes := make([]*Route, len(methods))
	for i, m := range methods {
		routes[i] = mux.Add(m, path, handler, middleware...)
	}
	return routes
}

// Static registers a new route with path prefix to serve static files from the
// provided root directory.
func (mux *Mux) Static(prefix, root string) *Route {
	if root == "" {
		root = "." // For security we want to restrict to CWD.
	}
	return static(mux, prefix, root)
}

func static(i i, prefix, root string) *Route {
	h := func(c Context) error {
		p, err := url.PathUnescape(c.Param("*"))
		if err != nil {
			return err
		}
		name := filepath.Join(root, path.Clean("/"+p)) // "/"+ for security
		return c.File(name)
	}
	i.GET(prefix, h)
	if prefix == "/" {
		return i.GET(prefix+"*", h)
	}

	return i.GET(prefix+"/*", h)
}

// File registers a new route with path to serve a static file with optional route-level middleware.
func (mux *Mux) File(path, file string, m ...MiddlewareFunc) *Route {
	return mux.GET(path, func(c Context) error {
		return c.File(file)
	}, m...)
}

// Add registers a new route for an HTTP method and path with matching handler
// in the router with optional route-level middleware.
func (mux *Mux) Add(method, path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	name := handlerName(handler)
	mux.router.add(method, path, func(c Context) error {
		h := handler
		// Chain middleware
		for i := len(middleware) - 1; i >= 0; i-- {
			h = compose(h, middleware[i])
		}
		return h(c)
	})
	r := &Route{
		Method: method,
		Path:   path,
		Name:   name,
	}
	mux.router.routes[method+path] = r
	return r
}

// Group creates a new router group with prefix and optional group-level middleware.
func (mux *Mux) Group(prefix string, m ...MiddlewareFunc) (g *Group) {
	g = &Group{prefix: prefix, mux: mux}
	g.Use(m...)
	return
}

// Routes returns the registered routes.
func (mux *Mux) Routes() []*Route {
	routes := make([]*Route, 0, len(mux.router.routes))
	for _, v := range mux.router.routes {
		routes = append(routes, v)
	}
	return routes
}

// ServeHTTP implements `http.Handler` interface, which serves HTTP requests.
func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Acquire context
	c := mux.pool.Get().(*context)
	c.reset(r, w)

	var h HandlerFunc

	if mux.premiddleware == nil {
		mux.router.find(r.Method, getPath(r), c)
		h = c.Handler()
		for i := len(mux.middleware) - 1; i >= 0; i-- {
			h = compose(h, mux.middleware[i])
		}
	} else {
		h = func(c Context) error {
			mux.router.find(r.Method, getPath(r), c)
			h := c.Handler()
			for i := len(mux.middleware) - 1; i >= 0; i-- {
				h = compose(h, mux.middleware[i])
			}
			return h(c)
		}
		for i := len(mux.premiddleware) - 1; i >= 0; i-- {
			h = compose(h, mux.premiddleware[i])
		}
	}

	// Execute chain
	if err := h(c); err != nil {
		mux.HTTPErrorHandler(err, c)
	}

	// Release context
	mux.pool.Put(c)
}

// defaultHTTPErrorHandler is the default HTTP error handler. It sends a JSON response
// with status code.
func (mux *Mux) defaultHTTPErrorHandler(err error, c Context) {
	var (
		code = http.StatusInternalServerError
		msg  interface{}
	)

	if he, ok := err.(*HTTPError); ok {
		code = he.Code
		msg = he.Message
		if he.Internal != nil {
			err = fmt.Errorf("%v, %v", err, he.Internal)
		}
	} else if mux.Debug {
		msg = err.Error()
	} else {
		msg = http.StatusText(code)
	}
	if _, ok := msg.(string); ok {
		msg = map[string]interface{}{"message": msg}
	}

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			_ = c.NoContent(code)
		} else {
			_ = c.JSON(code, msg)
		}
	}
}

// WrapHandler wraps `http.Handler` into `mux.HandlerFunc`.
func WrapHandler(h http.Handler) HandlerFunc {
	return func(c Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

func getPath(r *http.Request) string {
	rawPath := r.URL.RawPath
	if rawPath == "" {
		rawPath = r.URL.Path
	}
	return rawPath
}

func handlerName(h HandlerFunc) string {
	t := reflect.ValueOf(h).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	}
	return t.String()
}

// NewDefaultTemplateRenderer creates default template Renderer with given pattern.
func NewDefaultTemplateRenderer(pattern string) *templateRenderer {
	return &templateRenderer{
		templates: template.Must(template.ParseGlob(pattern)),
	}
}

type templateRenderer struct {
	templates *template.Template
}

func (t *templateRenderer) Render(w io.Writer, name string, data interface{}, c Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

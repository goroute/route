package route

import "net/http"

type (
	// MiddlewareFunc defines a function to process middleware.
	MiddlewareFunc func(c Context, next HandlerFunc) error

	// Skipper defines a function to skip middleware. Returning true skips processing
	// the middleware.
	Skipper func(Context) bool
)

// WrapMiddleware wraps `func(http.Handler) http.Handler` into `mux.MiddlewareFunc`
func WrapMiddleware(m func(http.Handler) http.Handler) MiddlewareFunc {
	return func(c Context, next HandlerFunc) (err error) {
		m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.SetRequest(r)
			err = next(c)
		})).ServeHTTP(c.Response(), c.Request())
		return
	}
}

// DefaultSkipper returns false which processes the middleware.
func DefaultSkipper(Context) bool {
	return false
}

// compose chains given handler with next middleware.
func compose(h HandlerFunc, m MiddlewareFunc) HandlerFunc {
	return func(c Context) error {
		return m(c, h)
	}
}

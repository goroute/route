package route

import (
	"fmt"
	"net/http"
)

// NewHTTPError creates a new HTTPError instance.
func NewHTTPError(code int, message ...interface{}) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(message) > 0 {
		he.Message = message[0]
	}
	return he
}

// Error makes it compatible with `error` interface.
func (he *HTTPError) Error() string {
	return fmt.Sprintf("code=%d, message=%v", he.Code, he.Message)
}

// SetInternal sets internal error on HTTPError
func (he *HTTPError) SetInternal(err error) *HTTPError {
	he.Internal = err
	return he
}

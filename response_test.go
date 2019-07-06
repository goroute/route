package route

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	e := NewServeMux()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	res := &Response{Writer: rec}

	// Before
	res.Before(func() {
		c.Response().Header().Set(HeaderServer, "mux")
	})
	res.Write([]byte("test"))
	assert.Equal(t, "mux", rec.Header().Get(HeaderServer))
}

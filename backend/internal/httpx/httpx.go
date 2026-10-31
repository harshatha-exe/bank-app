package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxBodyBytes = 64 << 10 // 64 KB is plenty for this API

type Response struct {
	Data interface{}            `json:"data,omitempty"`
	Meta map[string]interface{} `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Error ErrorBody              `json:"error"`
	Meta  map[string]interface{} `json:"meta,omitempty"`
}

type ErrorBody struct {
	Code    string            `json:"code" example:"VALIDATION_ERROR"`
	Message string            `json:"message" example:"Request validation failed."`
	Fields  map[string]string `json:"fields,omitempty"`
}

// BindJSON is a STRICT replacement for c.ShouldBindJSON:
//   - must be application/json
//   - body capped at 64 KB
//   - unknown fields rejected  (stops mass assignment: {"role":"ADMIN"} dies here)
//   - exactly one JSON value, no trailing junk
func BindJSON(c *gin.Context, dst any) error {
	if c.ContentType() != "application/json" {
		return apperr.New(http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Unsupported media type.")
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return apperr.New(http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Request too large.")
		}
		return apperr.BadRequest().WithCause(err) // real reason logged, client sees generic
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) { // anything after the JSON?
		return apperr.BadRequest()
	}
	return nil
}

// Session returns the logged-in session that Authenticate stored.
// Identity ALWAYS comes from here, never from the body or URL.
func Session(c *gin.Context) (auth.Session, bool) {
	v, _ := c.Get("session")
	s, ok := v.(auth.Session)
	return s, ok
}

// Fail records the error and stops the chain. ErrorHandler writes the response.
func Fail(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}

func OK(c *gin.Context, status int, data any, meta gin.H) {
	if meta == nil {
		meta = gin.H{}
	}
	meta["correlationId"] = c.GetString("correlation_id")
	c.JSON(status, gin.H{"data": data, "meta": meta})
}
func CorrelationID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(c.GetString("correlation_id"))
	return id
}

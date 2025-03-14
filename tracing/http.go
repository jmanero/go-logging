package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/jmanero/go-logging"
	"go.uber.org/zap"
)

// BaseContext supplies a context for a listener with an annotated logger
func BaseContext(ctx context.Context, logger *zap.Logger) func(listener net.Listener) context.Context {
	return func(listener net.Listener) context.Context {
		return logging.WithLogger(ctx, logger.With(zap.Stringer("listener", listener.Addr())))
	}
}

// ConnContext annotates a context logger for a connection
func ConnContext(ctx context.Context, conn net.Conn) context.Context {
	ctx, _ = logging.With(ctx, zap.Stringer("conn", conn.RemoteAddr()))

	return ctx
}

// ReadCloserProxy accumulates the number of bytes read from an underlying Reader
type ReadCloserProxy struct {
	io.ReadCloser

	Size int
}

// Read accumulates the number of bytes read from the underlying Reader
func (p *ReadCloserProxy) Read(b []byte) (n int, err error) {
	n, err = p.ReadCloser.Read(b)
	p.Size += n

	return
}

// ResponseWriterProxy captures the status code and body size of an HTTP response
type ResponseWriterProxy struct {
	http.ResponseWriter

	Status int
	Size   int
}

// WriteHeader captures the status code of an HTTP response
func (p *ResponseWriterProxy) WriteHeader(status int) {
	p.Status = status
	p.ResponseWriter.WriteHeader(status)
}

// Write accumulates size of an HTTP response's body
func (p *ResponseWriterProxy) Write(b []byte) (n int, err error) {
	n, err = p.ResponseWriter.Write(b)
	p.Size += n

	return
}

// GenerateID is a helper to generate a random identifier string
func GenerateID() (string, error) {
	var buf [32]byte

	_, err := rand.Read(buf[:])
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(buf[:]), nil
}

// Identifier is a middleware function that ensures an X-Request-ID header is present on the request context
func Identifier(next http.Handler) http.HandlerFunc {
	return func(wr http.ResponseWriter, req *http.Request) {
		// Try to use an existing tracing ID from downstream
		id := req.Header.Get("X-Request-ID")
		if len(id) == 0 {
			var err error

			// Generate a new tracing identifier
			id, err = GenerateID()
			if err != nil {
				panic(err)
			}

			// Ensure that the generated X-Request-ID header is included in upstream requests
			req.Header.Set("X-Request-ID", id)
		}

		// Ensure that the downstream response contains the X-Request-ID header
		wr.Header().Set("X-Request-ID", id)

		ctx, _ := logging.With(req.Context(), zap.String("id", id))
		next.ServeHTTP(wr, req.WithContext(ctx))
	}
}

// Logger is a middleware function that injects request information into the request's context logger, then logs HTTP
// request/response information after the request has completed
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {
		ctx, logger := logging.Named(req.Context(), "request",
			zap.String("host", req.Host),
			zap.String("proto", req.Proto),
			zap.String("method", req.Method),
			zap.String("path", req.RequestURI))

		// Wrap request reader and response writer in observable proxies
		reader := &ReadCloserProxy{ReadCloser: req.Body}
		writer := &ResponseWriterProxy{ResponseWriter: wr, Status: http.StatusOK}
		start := time.Now()

		req.Body = reader

		next.ServeHTTP(writer, req.WithContext(ctx))

		logger.Info("request completed",
			zap.Int("req_size", reader.Size),
			zap.Int("status", writer.Status),
			zap.Int("res_size", writer.Size),
			zap.Duration("duration", time.Since(start)),
		)
	})
}

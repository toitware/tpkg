package network

import (
	"context"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ErrorCode(err error) codes.Code {
	code := status.Code(err)
	if code != codes.Unknown {
		return code
	}

	switch err {
	case context.DeadlineExceeded:
		return codes.DeadlineExceeded
	case context.Canceled:
		return codes.Canceled
	default:
		return code
	}
}

type HttpHandlerWithError func(http.ResponseWriter, *http.Request) error

func HTTPHandle(handler HttpHandlerWithError) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			if err, ok := err.(ErrorWithStatusCode); ok {
				w.WriteHeader(err.StatusCode())
				w.Write([]byte(err.Error()))
				return
			}
			if status, ok := status.FromError(err); ok {
				w.WriteHeader(GRPCCodeToHTTPStatus(status.Code()))
				w.Write([]byte(status.Message()))
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}
}

type ErrorWithStatusCode interface {
	error
	StatusCode() int
}

type httpError struct {
	error
	statusCode int
}

func (err *httpError) StatusCode() int {
	return err.statusCode
}

func HTTPError(err error, statusCode int) error {
	return &httpError{error: err, statusCode: statusCode}
}

func GRPCCodeToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.InvalidArgument, codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.NotFound:
		return http.StatusNotFound
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.DataLoss:
		return http.StatusBadGateway
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

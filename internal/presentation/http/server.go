package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ogen-go/ogen/ogenerrors"
	api "github.com/whicu/hsa/api/http"
)

func NewServer(
	addr string,
	readTimeout time.Duration,
	readHeaderTimeout time.Duration,
	writeTimeout time.Duration,
	idleTimeout time.Duration,
	maxHeaderBytes int,
	handler http.Handler,
) *http.Server {
	return &http.Server{
		Addr:              addr,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		Handler:           handler,
	}
}

func NewRouter(
	logger *slog.Logger,
	h api.Handler,
	secHandler api.SecurityHandler,
	requestSizeLimit int64,
	trustedProxies, allowedOrigins []string,
) (http.Handler, error) {
	r := chi.NewRouter()

	ipMiddleware, err := NewClientIPMiddleware(logger, trustedProxies...)
	if err != nil {
		return nil, err
	}

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(ipMiddleware)
	r.Use(middleware.Logger)
	r.Use(middleware.RequestSize(requestSizeLimit))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
		},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	ogenServer, err := api.NewServer(
		h,
		secHandler,
		api.WithErrorHandler(newErrorHandler(logger)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init ogen server: %w", err)
	}

	r.Mount("/", ogenServer)

	return r, nil
}

func newErrorHandler(log *slog.Logger) func(context.Context, http.ResponseWriter, *http.Request, error) {
	return func(ctx context.Context, w http.ResponseWriter, _ *http.Request, err error) {
		if secErr, ok := errors.AsType[*ogenerrors.SecurityError](err); ok {
			switch {
			case errors.Is(secErr, ErrForbidden):
				writeForbidden(ctx, log, w)
			default:
				writeUnauthorized(ctx, log, w)
			}
			return
		}
		if sizeErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
			writeError(
				ctx,
				log,
				w,
				http.StatusRequestEntityTooLarge,
				NewErrorResponse[api.ErrorResponse](
					api.ErrorResponseErrorCodeVALIDATIONERROR,
					sizeErr.Error(),
				),
			)
			return
		}

		if errOgen, ok := errors.AsType[ogenerrors.Error](err); ok {
			writeError(
				ctx,
				log,
				w,
				errOgen.Code(),
				NewErrorResponse[api.ErrorResponse](
					api.ErrorResponseErrorCodeVALIDATIONERROR,
					errOgen.Error(),
				),
			)
			return
		}
		writeInternalError(ctx, log, w)
	}
}

func writeUnauthorized(ctx context.Context, log *slog.Logger, w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")

	writeError(
		ctx,
		log,
		w,
		http.StatusUnauthorized,
		NewErrorResponse[api.ErrorResponse](
			api.ErrorResponseErrorCodeUNAUTHORIZED,
			"unauthorized",
		),
	)
}

func writeForbidden(ctx context.Context, log *slog.Logger, w http.ResponseWriter) {
	writeError(
		ctx,
		log,
		w,
		http.StatusForbidden,
		NewErrorResponse[api.ErrorResponse](
			api.ErrorResponseErrorCodeFORBIDDEN,
			"insufficient permissions",
		),
	)
}

func writeInternalError(ctx context.Context, log *slog.Logger, w http.ResponseWriter) {
	writeError(
		ctx,
		log,
		w,
		http.StatusInternalServerError,
		NewErrorResponse[api.ErrorResponse](
			api.ErrorResponseErrorCodeINTERNALERROR,
			"internal server error",
		),
	)
}

func writeError(
	ctx context.Context,
	log *slog.Logger,
	w http.ResponseWriter,
	status int,
	response *api.ErrorResponse,
) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.DebugContext(ctx, "http: client disconnected before error response was written", slog.Any("error", err))
	}
}

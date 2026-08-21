package telemetry

// import (
// 	"context"
// 	// "time"

// 	"github.com/whicu/hsa/pkg/endpoint"
// 	"go.opentelemetry.io/otel"

// 	// "go.opentelemetry.io/otel/attribute"
// 	"go.opentelemetry.io/otel/codes"
// 	// "go.opentelemetry.io/otel/metric"
// 	"go.opentelemetry.io/otel/trace"
// )

// func TracingMiddleware[IN, OUT any](traceName, spanName string) endpoint.Middleware[IN, OUT] {
// 	tracer := otel.Tracer(traceName)
// 	return func(next endpoint.Handler[IN, OUT]) endpoint.Handler[IN, OUT] {
// 		return func(ctx context.Context, req IN) (OUT, error) {
// 			ctx, span := tracer.Start(ctx, spanName,
// 				trace.WithSpanKind(trace.SpanKindInternal),
// 			)
// 			defer span.End()

// 			res, err := next(ctx, req)
// 			if err != nil {
// 				span.RecordError(err)
// 				span.SetStatus(codes.Error, err.Error())
// 			} else {
// 				span.SetStatus(codes.Ok, "")
// 			}
// 			return res, err
// 		}
// 	}
// }

// func TracingMiddleware(operation string) endpoint.RawMiddleware {
// 	tracer := otel.Tracer("endpoint")
// 	return func(next endpoint.HandlerFunc) endpoint.HandlerFunc {
// 		return func(ctx context.Context, data []byte) (resp []byte, err error) {
// 			ctx, span := tracer.Start(ctx, operation,
// 				trace.WithSpanKind(trace.SpanKindInternal),
// 			)
// 			defer span.End()

// 			res, err := next(ctx, data)
// 			if err != nil {
// 				span.RecordError(err)
// 				span.SetStatus(codes.Error, err.Error())
// 			} else {
// 				span.SetStatus(codes.Ok, "")
// 			}
// 			return res, err
// 		}
// 	}
// }

// // - inflight gauge   — сколько запросов обрабатывается прямо сейчас
// - duration histogram — время выполнения в секундах
// // - requests counter  — общее число вызовов с разбивкой по success/error
// type Metrics struct {
// 	inflight metric.Int64UpDownCounter
// 	duration metric.Float64Histogram
// 	requests metric.Int64Counter
// }

// func NewMetrics(meter metric.Meter) (*Metrics, error) {
// 	inflight, err := meter.Int64UpDownCounter("endpoint.requests.inflight",
// 		metric.WithDescription("Number of in-flight requests"),
// 		metric.WithUnit("{request}"),
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	duration, err := meter.Float64Histogram("endpoint.request.duration",
// 		metric.WithDescription("Request duration"),
// 		metric.WithUnit("s"),
// 		metric.WithExplicitBucketBoundaries(.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5),
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	requests, err := meter.Int64Counter("endpoint.requests",
// 		metric.WithDescription("Number of requests"),
// 		metric.WithUnit("{request}"),
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &Metrics{inflight: inflight, duration: duration, requests: requests}, nil
// }

// func MetricsMiddleware(operation string) endpoint.RawMiddleware {
// 	meter := otel.Meter("endpoint")

// 	inflight, _ := meter.Int64UpDownCounter("endpoint.requests.inflight",
// 		metric.WithDescription("Number of in-flight requests"),
// 		metric.WithUnit("{request}"),
// 	)
// 	duration, _ := meter.Float64Histogram("endpoint.request.duration",
// 		metric.WithDescription("Request duration in seconds"),
// 		metric.WithUnit("s"),
// 		metric.WithExplicitBucketBoundaries(.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5),
// 	)
// 	total, _ := meter.Int64Counter("endpoint.requests.total",
// 		metric.WithDescription("Total number of requests"),
// 		metric.WithUnit("{request}"),
// 	)

// 	opAttr := attribute.String("operation", operation)

// 	return func(next endpoint.HandlerFunc) endpoint.HandlerFunc {
// 		return func(ctx context.Context, data []byte) (resp []byte, err error) {
// 			attrs := metric.WithAttributes(opAttr)

// 			inflight.Add(ctx, 1, attrs)
// 			defer inflight.Add(ctx, -1, attrs)

// 			start := time.Now()
// 			res, err := next(ctx, data)
// 			elapsed := time.Since(start).Seconds()

// 			status := attribute.String("status", "success")
// 			if err != nil {
// 				status = attribute.String("status", "error")
// 			}

// 			duration.Record(ctx, elapsed, metric.WithAttributes(opAttr, status))
// 			total.Add(ctx, 1, metric.WithAttributes(opAttr, status))

// 			return res, err
// 		}
// 	}
// }

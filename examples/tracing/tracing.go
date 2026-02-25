// Package tracing demonstrates httpx OpenTelemetry tracing:
// - Instrumenting an HTTP client with OTel spans
// - Trace context propagation via W3C headers
// - Span attributes (method, URL, status)
// - Error recording
package tracing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/n0l3r/httpx"
	httpxtracing "github.com/n0l3r/httpx/tracing"
)

// Run executes all tracing examples.
func Run() {
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  OPENTELEMETRY TRACING EXAMPLES")
	fmt.Println("═══════════════════════════════════════════")

	exampleBasicTracing()
	exampleTracePropagation()
	exampleErrorSpan()
	exampleManualSpan()
}

// setupTracer creates an in-memory span exporter and returns a tracer + exporter.
func setupTracer() (trace.Tracer, *tracetest.SpanRecorder) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp.Tracer("httpx-demo"), sr
}

// [1] Basic tracing — create a span per HTTP request.
func exampleBasicTracing() {
	fmt.Println("\n[1] Basic OTel tracing — span per HTTP request")

	tracer, recorder := setupTracer()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back trace context header so we can verify propagation
		w.Header().Set("X-Traceparent", r.Header.Get("Traceparent"))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	transport := &httpxtracing.Transport{
		Tracer:     tracer,
		Propagator: otel.GetTextMapPropagator(),
	}

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithTransport(transport))

	ctx, span := tracer.Start(context.Background(), "example-operation")
	defer span.End()

	resp, err := c.Get(ctx, "/api/users")
	if err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}

	fmt.Printf("  ✓ status=%d\n", resp.StatusCode())
	fmt.Printf("    Traceparent in request: %v\n", resp.Header("X-Traceparent") != "")

	// Inspect recorded spans
	spans := recorder.Ended()
	fmt.Printf("    Recorded spans: %d\n", len(spans))
	for _, s := range spans {
		fmt.Printf("    → span name: %q status: %s\n", s.Name(), s.Status().Code)
	}
}

// [2] Trace context propagation.
func exampleTracePropagation() {
	fmt.Println("\n[2] W3C trace context propagation")

	tracer, _ := setupTracer()

	var receivedTraceparent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTraceparent = r.Header.Get("Traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := &httpxtracing.Transport{Tracer: tracer}
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithTransport(transport))

	ctx, span := tracer.Start(context.Background(), "parent-operation")
	defer span.End()

	c.Get(ctx, "/downstream")

	fmt.Printf("  ✓ Traceparent header sent: %v\n", receivedTraceparent != "")
	if receivedTraceparent != "" {
		fmt.Printf("    Value: %s\n", receivedTraceparent)
	}
}

// [3] Error span recording.
func exampleErrorSpan() {
	fmt.Println("\n[3] Error span — 5xx responses recorded as errors")

	tracer, recorder := setupTracer()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error":"internal server error"}`)
	}))
	defer srv.Close()

	transport := &httpxtracing.Transport{Tracer: tracer}
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithTransport(transport))

	ctx, span := tracer.Start(context.Background(), "failing-operation")
	c.Get(ctx, "/api/broken")
	span.End()

	spans := recorder.Ended()
	for _, s := range spans {
		if s.Name() == "HTTP GET" {
			fmt.Printf("  ✓ Span %q status: %s\n", s.Name(), s.Status().Code)
			if s.Status().Code == codes.Error {
				fmt.Printf("    Error correctly recorded in span\n")
			}
		}
	}
}

// [4] Manual span wrapping around multiple requests.
func exampleManualSpan() {
	fmt.Println("\n[4] Manual span — wrapping multiple HTTP calls")

	tracer, recorder := setupTracer()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"path":%q}`, r.URL.Path)
	}))
	defer srv.Close()

	transport := &httpxtracing.Transport{Tracer: tracer}
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithTransport(transport))

	// Start a parent span for a business operation.
	ctx, parentSpan := tracer.Start(context.Background(), "checkout-flow",
		trace.WithAttributes(attribute.String("user.id", "usr-42")),
	)
	defer parentSpan.End()

	endpoints := []string{"/cart", "/inventory", "/payment"}
	for _, ep := range endpoints {
		c.Get(ctx, ep)
	}

	spans := recorder.Ended()
	fmt.Printf("  ✓ %d HTTP spans recorded under parent 'checkout-flow'\n", len(spans))
	for _, s := range spans {
		fmt.Printf("    [%s] %s\n", s.Status().Code, s.Name())
	}
}

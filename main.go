package main

import (
	"context"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// newExporter 创建一个新的控制台导出器
func newExporter(w io.Writer) (sdktrace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithoutTimestamps(),
	)
}

// newResource 创建一个用于描述此应用的资源
func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("my-dice-roller"),
			semconv.ServiceVersion("v1.0.0"),
		),
	)
	return r
}

func main() {
	// ---- OpenTelemetry 初始化 ----
	exporter, err := newExporter(log.Writer())
	if err != nil {
		log.Fatalf("创建 exporter 失败: %v", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource()),
	)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("关闭 tracer provider 时出错: %v", err)
		}
	}()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	// ---- OpenTelemetry 初始化结束 ----

	handler := http.NewServeMux()
	rolldiceHandler := http.HandlerFunc(rolldice)
	handler.Handle("/rolldice", otelhttp.NewHandler(rolldiceHandler, "rolldice"))

	log.Println("在 8080 端口启动服务器")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}

// rolldice 是我们的业务逻辑处理器
func rolldice(w http.ResponseWriter, r *http.Request) {
	// ---- 手动插桩 ----
	ctx := r.Context()
	tracer := otel.Tracer("dice-roller-manual")

	var span trace.Span
	ctx, span = tracer.Start(ctx, "roll")
	defer span.End()
	// ---- 手动插桩结束 ----

	roll := 1 + rand.Intn(6)
	time.Sleep(time.Duration(roll) * 100 * time.Millisecond)

	resp := strconv.Itoa(roll) + "\n"
	io.WriteString(w, resp)
}

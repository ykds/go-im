package mgrpc

import (
	"context"
	"go-im/internal/pkg/mtrace"
	"net"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var _ propagation.TextMapCarrier = (*metadataCarrier)(nil)

type metadataCarrier struct {
	metadata *metadata.MD
}

func (m *metadataCarrier) Get(key string) string {
	v := m.metadata.Get(key)
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// Set stores the key-value pair.
func (m *metadataCarrier) Set(key string, value string) {
	m.metadata.Set(key, value)
}

// Keys lists the keys stored in this carrier.
func (m *metadataCarrier) Keys() []string {
	out := []string{}
	for key := range *m.metadata {
		out = append(out, key)
	}
	return out
}

func UnaryServerTrace() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}
		tmp := otel.GetTextMapPropagator()
		ctx = tmp.Extract(ctx, &metadataCarrier{&md})

		name := strings.TrimLeft(info.FullMethod, "/")
		parts := strings.SplitN(name, "/", 2)
		var attrs []attribute.KeyValue
		if len(parts) == 2 {
			if service := parts[0]; service != "" {
				attrs = append(attrs, semconv.RPCServiceKey.String(service))
			}
			if method := parts[1]; method != "" {
				attrs = append(attrs, semconv.RPCMethodKey.String(method))
			}
		}
		p, ok := peer.FromContext(ctx)
		if ok && p != nil {
			addr := p.Addr.String()
			host, port, err := net.SplitHostPort(addr)
			if err == nil {
				if host == "" {
					host = "127.0.0.1"
				}
				attrs = append(attrs, semconv.NetPeerNameKey.String(host))
				attrs = append(attrs, semconv.NetPeerPortKey.String(port))
			}
		}

		ctx, span := mtrace.StartSpan(ctx, name,
			trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attrs...))
		defer mtrace.EndSpan(span)

		resp, err = handler(ctx, req)
		if err != nil {
			s, ok := status.FromError(err)
			if ok {
				span.SetStatus(codes.Error, s.Message())
				span.SetAttributes(mtrace.GrpcErrorCode.Int64(int64(s.Code())))
			} else {
				span.SetStatus(codes.Error, err.Error())
			}
			return nil, err
		}
		span.SetAttributes(mtrace.GRPCStatusCodeKey.Int64(int64(codes.Ok)))
		return resp, nil
	}
}

type wrapStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w wrapStream) Context() context.Context {
	return w.ctx
}

func StreamServerTrace() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		spanCtx := trace.SpanContextFromContext(ctx)
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}
		tmp := otel.GetTextMapPropagator()
		ctx = tmp.Extract(ctx, &metadataCarrier{&md})

		name := strings.TrimLeft(info.FullMethod, "/")
		parts := strings.SplitN(name, "/", 2)
		var attrs []attribute.KeyValue
		if len(parts) == 2 {
			if service := parts[0]; service != "" {
				attrs = append(attrs, semconv.RPCServiceKey.String(service))
			}
			if method := parts[1]; method != "" {
				attrs = append(attrs, semconv.RPCMethodKey.String(method))
			}
		}
		p, ok := peer.FromContext(ctx)
		if ok && p != nil {
			addr := p.Addr.String()
			host, port, err := net.SplitHostPort(addr)
			if err == nil {
				if host == "" {
					host = "127.0.0.1"
				}
				attrs = append(attrs, semconv.NetPeerIPKey.String(host))
				attrs = append(attrs, semconv.NetPeerPortKey.String(port))
			}
		}

		_, span := mtrace.StartSpan(trace.ContextWithRemoteSpanContext(ctx, spanCtx), name,
			trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attrs...))
		defer mtrace.EndSpan(span)

		if err := handler(srv, wrapStream{ss, ctx}); err != nil {
			s, ok := status.FromError(err)
			if ok {
				span.SetStatus(codes.Error, s.Message())
				span.SetAttributes(mtrace.GrpcErrorCode.Int64(int64(s.Code())))
			} else {
				span.SetStatus(codes.Error, err.Error())
			}
			return err
		}
		span.SetAttributes(mtrace.GRPCStatusCodeKey.Int64(int64(codes.Ok)))
		return nil
	}
}

func UnaryClientTrace() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		attrs := []attribute.KeyValue{
			semconv.RPCSystemKey.String("grpc"),
		}

		name := strings.TrimLeft(method, "/")
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			if service := parts[0]; service != "" {
				attrs = append(attrs, semconv.RPCServiceKey.String(service))
			}
			if method := parts[1]; method != "" {
				attrs = append(attrs, semconv.RPCMethodKey.String(method))
			}
		}
		host, port, err := net.SplitHostPort(cc.Target())
		if err == nil {
			attrs = append(attrs, semconv.NetPeerIPKey.String(host))
			attrs = append(attrs, semconv.NetPeerPortKey.String(port))
		}

		ctx, span := mtrace.StartSpan(ctx, name, trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attrs...))
		defer mtrace.EndSpan(span)

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}
		propagation := otel.GetTextMapPropagator()
		propagation.Inject(ctx, &metadataCarrier{&md})
		ctx = metadata.NewOutgoingContext(ctx, md)

		err = invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			s, ok := status.FromError(err)
			if ok {
				span.SetStatus(codes.Error, s.Message())
				span.SetAttributes(mtrace.GrpcErrorCode.Int64(int64(s.Code())))
				span.SetAttributes(mtrace.GrpcErrorMsg.String(s.Message()))
			} else {
				span.SetStatus(codes.Error, err.Error())
			}
			return err
		}
		return nil
	}
}

func StreamClientTrace() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		attrs := []attribute.KeyValue{
			semconv.RPCSystemKey.String("grpc"),
		}

		name := strings.TrimLeft(method, "/")
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			if service := parts[0]; service != "" {
				attrs = append(attrs, semconv.RPCServiceKey.String(service))
			}
			if method := parts[1]; method != "" {
				attrs = append(attrs, semconv.RPCMethodKey.String(method))
			}
		}
		host, port, err := net.SplitHostPort(cc.Target())
		if err == nil {
			attrs = append(attrs, semconv.NetPeerIPKey.String(host))
			attrs = append(attrs, semconv.NetPeerPortKey.String(port))
		}

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}
		propagation := otel.GetTextMapPropagator()
		propagation.Inject(ctx, &metadataCarrier{&md})

		ctx, span := mtrace.StartSpan(ctx, name, trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attrs...))
		defer mtrace.EndSpan(span)

		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			s, ok := status.FromError(err)
			if ok {
				span.SetStatus(codes.Error, s.Message())
				span.SetAttributes(mtrace.GrpcErrorCode.Int64(int64(s.Code())))
				span.SetAttributes(mtrace.GrpcErrorMsg.String(s.Message()))
			} else {
				span.SetStatus(codes.Error, err.Error())
			}
			return stream, err
		}
		return stream, nil
	}
}

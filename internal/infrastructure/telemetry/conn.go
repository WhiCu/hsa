package telemetry

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func InitConn(cfg Config) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	if cfg.Exporter.Conn.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		creds := credentials.NewTLS(nil)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	if cmps := cfg.Exporter.Conn.Compressor; cmps != "" {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(cmps)))
	}

	opts = append(opts,
		grpc.WithKeepaliveParams(CfgKeepAliveToGRPCKeepAlive(cfg.Exporter.Conn.KeepAlive)),
		grpc.WithConnectParams(
			grpc.ConnectParams{
				Backoff: CfgBackoffToGRPCBackoff(cfg.Exporter.Conn.Backoff),
			},
		),
	)

	conn, err := grpc.NewClient(cfg.Exporter.Conn.Endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial grpc server: %w", err)
	}

	return conn, nil
}

func CfgKeepAliveToGRPCKeepAlive(cfg KeepAliveConfig) keepalive.ClientParameters {
	return keepalive.ClientParameters{
		Time:                cfg.Time,
		Timeout:             cfg.Timeout,
		PermitWithoutStream: cfg.PermitWithoutStream,
	}
}

func CfgBackoffToGRPCBackoff(cfg BackoffConfig) backoff.Config {
	return backoff.Config{
		BaseDelay:  cfg.BaseDelay,
		MaxDelay:   cfg.MaxDelay,
		Multiplier: cfg.Multiplier,
		Jitter:     cfg.Jitter,
	}
}

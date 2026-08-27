#!/bin/bash
sed -i 's/"1.0.0"/constVersion/g' internal/infrastructure/telemetry/telemetry_test.go
sed -i 's/"dev"/constEnv/g' internal/infrastructure/telemetry/telemetry_test.go
sed -i 's/"127.0.0.1:4317"/constEndpoint/g' internal/infrastructure/telemetry/telemetry_test.go
sed -i 's/"always"/constAlways/g' internal/infrastructure/telemetry/telemetry_test.go
sed -i 's|import (|import (\n\t"context"\n\t"testing"\n\t"time"\n\n\t. "github.com/onsi/ginkgo/v2"\n\t. "github.com/onsi/gomega"\n\n\t"github.com/knadh/koanf/providers/confmap"\n\t"github.com/knadh/koanf/v2"\n\t"github.com/samber/do/v2"\n\t"go.opentelemetry.io/otel/sdk/resource"\n\t"google.golang.org/grpc"\n\t"google.golang.org/grpc/backoff"\n\t"google.golang.org/grpc/keepalive"\n\n\t"github.com/whicu/hsa/internal/infrastructure/telemetry"\n)|g' patch_imports.go

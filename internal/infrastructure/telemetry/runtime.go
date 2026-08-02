package telemetry

import (
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
)

const otelRuntimeInterval = 10 * time.Second

func InitRuntimeMetrics() error {
	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(otelRuntimeInterval)); err != nil {
		return fmt.Errorf("failed to start runtime metrics: %w", err)
	}
	return nil
}

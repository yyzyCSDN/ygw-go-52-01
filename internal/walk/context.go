package walk

import (
	"context"

	"graphstore/internal/metric"
)

// checkCancel returns the context error when the traversal should stop. It
// is used at every expansion step so a cancelled request halts promptly.
func checkCancel(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

// stopWalk aborts the traversal when the request context is cancelled so a
// disconnected client releases walker resources promptly.
func stopWalk(ctx context.Context, registry *metric.Registry) error {
	if err := checkCancel(ctx); err != nil {
		registry.WalkCancels.Inc()
		return err
	}
	return nil
}

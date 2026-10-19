package sandbox

import (
	"context"
	"runtime/metrics"
	"time"
)

func watch(ctx context.Context, limits serveLimits) {
	const tick = 5 * time.Millisecond
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if inUse() > limits.Ceiling {
				limits.Exit(exitMemory)
				return
			}
		}
	}
}

func inUse() uint64 {
	samples := []metrics.Sample{
		{Name: "/memory/classes/total:bytes"},
		{Name: "/memory/classes/heap/released:bytes"},
	}
	metrics.Read(samples)
	total, released := samples[0].Value, samples[1].Value
	if total.Kind() != metrics.KindUint64 || released.Kind() != metrics.KindUint64 {
		return 0
	}
	return total.Uint64() - min(released.Uint64(), total.Uint64())
}

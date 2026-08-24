package metric

import "sync/atomic"

// Counter is a monotonic atomic counter used by the metrics registry.
type Counter struct {
	value atomic.Int64
}

// Add increments the counter by delta.
func (c *Counter) Add(delta int64) {
	c.value.Add(delta)
}

// Inc increments the counter by one.
func (c *Counter) Inc() {
	c.value.Add(1)
}

// Value returns the current counter value.
func (c *Counter) Value() int64 {
	return c.value.Load()
}

// Gauge is an atomic gauge that can go down as well as up.
type Gauge struct {
	value atomic.Int64
}

// Set stores the gauge value.
func (g *Gauge) Set(value int64) {
	g.value.Store(value)
}

// Value returns the current gauge value.
func (g *Gauge) Value() int64 {
	return g.value.Load()
}

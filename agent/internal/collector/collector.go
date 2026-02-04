package collector

// Collector defines the interface for data collectors
type Collector interface {
	Collect() (interface{}, error)
	Name() string
}

// BaseCollector provides common functionality for collectors
type BaseCollector struct {
	name string
}

// Name returns the collector name
func (b *BaseCollector) Name() string {
	return b.name
}

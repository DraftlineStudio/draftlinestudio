package indexing

import (
	"runtime"
	"sync"
)

// AnalysisPoolOptions limits only manuscript-analysis work. It never changes
// the process-wide Go scheduler. MaxInFlightBytes bounds source text held by
// concurrently executing prose jobs; MaxBatchBytes limits combined NLP jobs.
type AnalysisPoolOptions struct {
	Workers          int
	MaxInFlightBytes int
	MaxBatchBytes    int
}

func defaultAnalysisPoolOptions() AnalysisPoolOptions {
	return AnalysisPoolOptions{Workers: min(4, maxInt(1, runtime.NumCPU()))}
}

func normalizeAnalysisPoolOptions(options AnalysisPoolOptions, jobs int) AnalysisPoolOptions {
	if options.Workers < 1 {
		options = defaultAnalysisPoolOptions()
	}
	if jobs > 0 && options.Workers > jobs {
		options.Workers = jobs
	}
	if options.Workers < 1 {
		options.Workers = 1
	}
	if options.MaxInFlightBytes < 0 {
		options.MaxInFlightBytes = 0
	}
	if options.MaxBatchBytes < 0 {
		options.MaxBatchBytes = 0
	}
	return options
}

type analysisMemoryGate struct {
	capacity int
	used     int
	cond     *sync.Cond
}

func newAnalysisMemoryGate(capacity int) *analysisMemoryGate {
	return &analysisMemoryGate{capacity: capacity, cond: sync.NewCond(&sync.Mutex{})}
}

func (g *analysisMemoryGate) acquire(bytes int) int {
	if g.capacity <= 0 || bytes <= 0 {
		return 0
	}
	weight := bytes
	if weight > g.capacity {
		// An oversized chapter runs alone. Splitting arbitrary prose would damage
		// sentence/entity offsets, so charge it the full gate capacity instead.
		weight = g.capacity
	}
	g.cond.L.Lock()
	for g.used+weight > g.capacity {
		g.cond.Wait()
	}
	g.used += weight
	g.cond.L.Unlock()
	return weight
}

func (g *analysisMemoryGate) release(weight int) {
	if g.capacity <= 0 || weight <= 0 {
		return
	}
	g.cond.L.Lock()
	g.used -= weight
	g.cond.Broadcast()
	g.cond.L.Unlock()
}

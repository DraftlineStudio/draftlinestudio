package indexing

import (
	"sync"
	"testing"
	"time"
)

func TestLinguisticBatchesRespectMemoryBoundaryWithoutLosingChapters(t *testing.T) {
	chapters := []string{
		string(make([]byte, 6)),
		string(make([]byte, 6)),
		string(make([]byte, 3)),
		string(make([]byte, 20)), // Oversized input remains one intact batch.
	}
	batches := linguisticBatches(chapters, AnalysisPoolOptions{Workers: 4, MaxBatchBytes: 10})
	want := []linguisticBatch{
		{start: 0, end: 1, bytes: 6},
		{start: 1, end: 2, bytes: 6},
		{start: 2, end: 3, bytes: 3},
		{start: 3, end: 4, bytes: 20},
	}
	if len(batches) != len(want) {
		t.Fatalf("got batches %+v, want %+v", batches, want)
	}
	for i := range want {
		if batches[i] != want[i] {
			t.Fatalf("batch %d = %+v, want %+v", i, batches[i], want[i])
		}
	}
}

func TestAnalysisMemoryGateBoundsConcurrentPayload(t *testing.T) {
	const (
		capacity = 10
		jobs     = 6
		weight   = 6
	)
	gate := newAnalysisMemoryGate(capacity)
	start := make(chan struct{})
	release := make(chan struct{})
	entered := make(chan struct{}, jobs)
	var workers sync.WaitGroup

	for range jobs {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			charged := gate.acquire(weight)
			entered <- struct{}{}
			<-release
			gate.release(charged)
		}()
	}
	close(start)
	select {
	case <-entered:
	case <-time.After(time.Second):
		close(release)
		workers.Wait()
		t.Fatal("no payload entered the memory gate")
	}
	select {
	case <-entered:
		close(release)
		workers.Wait()
		t.Fatal("memory gate allowed a second payload before the first released")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	workers.Wait()
}

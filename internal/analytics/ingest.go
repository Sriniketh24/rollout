package analytics

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Sriniketh24/rollout/internal/models"
)

// Ingester buffers incoming exposure and conversion events and flushes them to
// ClickHouse in configurable batches.
type Ingester struct {
	analytics     *Analytics
	batchSize     int
	flushInterval time.Duration

	exposureMu  sync.Mutex
	exposureBuf []models.ExposureEvent

	conversionMu  sync.Mutex
	conversionBuf []models.ConversionEvent

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewIngester creates a new event ingester that flushes to the given Analytics
// store. batchSize controls the maximum number of events per flush, and
// flushInterval controls how often the background worker checks for pending
// events.
func NewIngester(analytics *Analytics, batchSize int, flushInterval time.Duration) *Ingester {
	return &Ingester{
		analytics:     analytics,
		batchSize:     batchSize,
		flushInterval: flushInterval,
	}
}

// AddExposure enqueues an exposure event for batch insertion. This call is
// non-blocking and thread-safe.
func (ing *Ingester) AddExposure(event models.ExposureEvent) {
	ing.exposureMu.Lock()
	ing.exposureBuf = append(ing.exposureBuf, event)
	ing.exposureMu.Unlock()
}

// AddConversion enqueues a conversion event for batch insertion. This call is
// non-blocking and thread-safe.
func (ing *Ingester) AddConversion(event models.ConversionEvent) {
	ing.conversionMu.Lock()
	ing.conversionBuf = append(ing.conversionBuf, event)
	ing.conversionMu.Unlock()
}

// Start begins background goroutines that periodically flush buffered events to
// ClickHouse. It blocks until the provided context is cancelled or Stop is
// called.
func (ing *Ingester) Start(ctx context.Context) {
	ctx, ing.cancel = context.WithCancel(ctx)

	ing.wg.Add(2)
	go ing.flushLoop(ctx, &ing.wg, ing.flushExposures)
	go ing.flushLoop(ctx, &ing.wg, ing.flushConversions)
}

// Stop flushes any remaining buffered events and shuts down background
// goroutines. It blocks until all pending flushes complete.
func (ing *Ingester) Stop() {
	if ing.cancel != nil {
		ing.cancel()
	}
	ing.wg.Wait()

	// Final flush for any events that arrived after the last tick.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ing.flushExposures(ctx)
	ing.flushConversions(ctx)
}

// flushLoop runs fn on every tick until ctx is done.
func (ing *Ingester) flushLoop(ctx context.Context, wg *sync.WaitGroup, fn func(context.Context)) {
	defer wg.Done()
	ticker := time.NewTicker(ing.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn(ctx)
		}
	}
}

// flushExposures drains the exposure buffer in batchSize chunks and inserts them
// into ClickHouse.
func (ing *Ingester) flushExposures(ctx context.Context) {
	for {
		ing.exposureMu.Lock()
		if len(ing.exposureBuf) == 0 {
			ing.exposureMu.Unlock()
			return
		}
		n := ing.batchSize
		if n > len(ing.exposureBuf) {
			n = len(ing.exposureBuf)
		}
		batch := make([]models.ExposureEvent, n)
		copy(batch, ing.exposureBuf[:n])
		ing.exposureBuf = ing.exposureBuf[n:]
		ing.exposureMu.Unlock()

		if err := ing.analytics.InsertExposures(ctx, batch); err != nil {
			log.Printf("analytics: failed to flush exposures: %v", err)
			// Re-enqueue on failure so events are not lost.
			ing.exposureMu.Lock()
			ing.exposureBuf = append(batch, ing.exposureBuf...)
			ing.exposureMu.Unlock()
			return
		}
	}
}

// flushConversions drains the conversion buffer in batchSize chunks and inserts
// them into ClickHouse.
func (ing *Ingester) flushConversions(ctx context.Context) {
	for {
		ing.conversionMu.Lock()
		if len(ing.conversionBuf) == 0 {
			ing.conversionMu.Unlock()
			return
		}
		n := ing.batchSize
		if n > len(ing.conversionBuf) {
			n = len(ing.conversionBuf)
		}
		batch := make([]models.ConversionEvent, n)
		copy(batch, ing.conversionBuf[:n])
		ing.conversionBuf = ing.conversionBuf[n:]
		ing.conversionMu.Unlock()

		if err := ing.analytics.InsertConversions(ctx, batch); err != nil {
			log.Printf("analytics: failed to flush conversions: %v", err)
			// Re-enqueue on failure so events are not lost.
			ing.conversionMu.Lock()
			ing.conversionBuf = append(batch, ing.conversionBuf...)
			ing.conversionMu.Unlock()
			return
		}
	}
}

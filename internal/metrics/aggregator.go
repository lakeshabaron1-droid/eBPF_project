
package metrics

import (

	"sort"
	"sync"
	"time"
)

type AggregatedSnapshot struct {
	Timestamp    time.Time          `json:"timestamp"`
	PassedPerSec float64            `json:"passed_per_sec"`
	DroppedPerSec float64           `json:"dropped_per_sec"`
	DropRate     float64            `json:"drop_rate"`
	TotalPassed  uint64             `json:"total_passed"`
	TotalDropped uint64             `json:"total_dropped"`
	ProtocolStats map[string]uint64 `json:"protocol_stats"`
	TopBlockedIPs []BlockedIPEntry  `json:"top_blocked_ips"`
	Windows      WindowStats        `json:"windows"`
}

type BlockedIPEntry struct {


	IP    string `json:"ip"`
	Count uint64 `json:"count"`
}

type WindowStats struct {
	OneMin     RateWindow `json:"1m"`
	FiveMin    RateWindow `json:"5m"`
	FifteenMin RateWindow `json:"15m"`
}

type RateWindow struct {

	PassedPerSec  float64 `json:"passed_per_sec"`
	DroppedPerSec float64 `json:"dropped_per_sec"`
}

type circularBuffer struct {

	data  []MetricSnapshot
	head  int
	count int
	cap   int
}

func newCircularBuffer(capacity int) *circularBuffer {
	return &circularBuffer{
		data: make([]MetricSnapshot, capacity),
		cap:  capacity,
	}
}

func (cb *circularBuffer) Push(snap MetricSnapshot) {

	cb.data[cb.head] = snap
	cb.head = (cb.head + 1) % cb.cap

	if cb.count < cb.cap {
		cb.count++
	}
}

func (cb *circularBuffer) Average() (float64, float64) {
	if cb.count == 0 {
		return 0, 0
	}
	var totalPassed, totalDropped uint64
	for i := 0; i < cb.count; i++ {
		idx := (cb.head - cb.count + i + cb.cap) % cb.cap
		totalPassed += cb.data[idx].Passed
		totalDropped += cb.data[idx].Dropped
	}
	seconds := float64(cb.count)

	return float64(totalPassed) / seconds, float64(totalDropped) / seconds

}


type Aggregator struct {
	mu            sync.RWMutex
	oneMin        *circularBuffer
	fiveMin       *circularBuffer
	fifteenMin    *circularBuffer

	blockedIPMap  map[string]uint64
	totalPassed   uint64
	totalDropped  uint64
	latest        AggregatedSnapshot
	input         chan MetricSnapshot
	done          chan struct{}
}

func NewAggregator() *Aggregator {
	return &Aggregator{
		oneMin:       newCircularBuffer(60),
		fiveMin:      newCircularBuffer(300),
		fifteenMin:   newCircularBuffer(900),
		blockedIPMap: make(map[string]uint64),
		input:        make(chan MetricSnapshot, 64),
		done:         make(chan struct{}),
	}
}

func (a *Aggregator) InputChannel() chan<- MetricSnapshot {

	return a.input
}

func (a *Aggregator) Start() {
	go a.processLoop()
}

func (a *Aggregator) Stop() {
	close(a.done)
}

func (a *Aggregator) Snapshot() AggregatedSnapshot {

	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.latest
}

func (a *Aggregator) processLoop() {
	for {
		select {
		case <-a.done:
			return
		case snap := <-a.input:
			a.process(snap)
		}
	}

}

func (a *Aggregator) process(snap MetricSnapshot) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.totalPassed += snap.Passed
	a.totalDropped += snap.Dropped

	a.oneMin.Push(snap)
	a.fiveMin.Push(snap)
	a.fifteenMin.Push(snap)

	for _, ev := range snap.DropEvents {
		a.blockedIPMap[ev.SrcIP]++
	}


	oneP, oneD := a.oneMin.Average()
	fiveP, fiveD := a.fiveMin.Average()
	fifteenP, fifteenD := a.fifteenMin.Average()

	var dropRate float64

	total := float64(snap.Passed + snap.Dropped)
	if total > 0 {
		dropRate = float64(snap.Dropped) / total
	}

	topIPs := a.computeTopBlockedIPs(10)

	a.latest = AggregatedSnapshot{
		Timestamp:     snap.Timestamp,
		PassedPerSec:  oneP,
		DroppedPerSec: oneD,
		DropRate:      dropRate,
		TotalPassed:   a.totalPassed,
		TotalDropped:  a.totalDropped,
		ProtocolStats: snap.ProtocolStats,
		TopBlockedIPs: topIPs,
		Windows: WindowStats{
			OneMin:     RateWindow{PassedPerSec: oneP, DroppedPerSec: oneD},
			FiveMin:    RateWindow{PassedPerSec: fiveP, DroppedPerSec: fiveD},
			FifteenMin: RateWindow{PassedPerSec: fifteenP, DroppedPerSec: fifteenD},

		},
	}
}


package metrics




import (


	"fmt"

	"sync"

	"time"

	bpf "ebpf-gateway/internal/ebpf"

)

type MetricSnapshot struct {

	Timestamp     time.Time          `json:"timestamp"`
	Passed        uint64             `json:"passed"`
	Dropped       uint64             `json:"dropped"`
	ProtocolStats map[string]uint64  `json:"protocol_stats"`
	PortStats     map[uint16]PortStat `json:"port_stats"`
	DropEvents    []DropEventInfo    `json:"drop_events"`
}

type PortStat struct {
	Packets uint64 `json:"packets"`
	Bytes   uint64 `json:"bytes"`

}


type DropEventInfo struct {
	SrcIP    string `json:"src_ip"`
	DstIP    string `json:"dst_ip"`

	SrcPort  uint16 `json:"src_port"`
	DstPort  uint16 `json:"dst_port"`



	Protocol uint8  `json:"protocol"`
	Reason   uint8  `json:"reason"`
}

type Collector struct {
	maps       *bpf.MapManager
	eventCh    chan bpf.DropEvent
	done       chan struct{}
	mu         sync.RWMutex
	latest     MetricSnapshot


	prevPassed uint64
	prevDropped uint64
	listeners  []chan<- MetricSnapshot

	listenerMu sync.RWMutex
}



func NewCollector(maps *bpf.MapManager) *Collector {
	return &Collector{
		maps:    maps,
		eventCh: make(chan bpf.DropEvent, 1024),
		done:    make(chan struct{}),
	}

}

func (c *Collector) Subscribe(ch chan<- MetricSnapshot) {
	c.listenerMu.Lock()
	c.listeners = append(c.listeners, ch)
	c.listenerMu.Unlock()
}

func (c *Collector) Start() error {
	if err := c.maps.StartEventReader(c.eventCh, c.done); err != nil {
		return err
	}

	go c.collectLoop()
	go c.consumeEvents()



	return nil
}

func (c *Collector) Stop() {


	close(c.done)
}

func (c *Collector) Latest() MetricSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latest
}

func (c *Collector) collectLoop() {
	ticker := time.NewTicker(1 * time.Second)


	defer ticker.Stop()

	for {

		select {
		case <-c.done:


			return

		case <-ticker.C:

			c.poll()
		}
	}

}

func (c *Collector) poll() {
	passed, dropped, err := c.maps.GetPacketCounters()
	if err != nil {
		return


	}

	protoStats, _ := c.maps.GetProtocolStats()

	rawPortStats, _ := c.maps.GetPortStats()
	portStats := make(map[uint16]PortStat)
	for port, ps := range rawPortStats {
		portStats[port] = PortStat{
			Packets: ps.Packets,
			Bytes:   ps.Bytes,

		}
	}


	deltaPassed := passed - c.prevPassed
	deltaDropped := dropped - c.prevDropped
	c.prevPassed = passed
	c.prevDropped = dropped


	snap := MetricSnapshot{
		Timestamp:     time.Now(),
		Passed:        deltaPassed,
		Dropped:       deltaDropped,
		ProtocolStats: protoStats,
		PortStats:     portStats,

	}

	c.mu.Lock()
	snap.DropEvents = c.latest.DropEvents
	c.latest = snap

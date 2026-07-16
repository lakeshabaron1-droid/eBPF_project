package ebpf

import (
	"bytes"
	"encoding/binary"

	"errors"
	"net"
	"runtime"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"

)

type MapManager struct {
	m *Manager
}

func NewMapManager(m *Manager) *MapManager {
	return &MapManager{m: m}
}

func ipToUint32(ipStr string) (uint32, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0, errors.New("invalid IP")

	}
	ip = ip.To4()
	if ip == nil {


		return 0, errors.New("not an IPv4 address")
	}
	return binary.BigEndian.Uint32(ip), nil
}


func uint32ToIP(ipInt uint32) string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, ipInt)
	return ip.String()
}


func (mm *MapManager) BlockIP(ipStr string) error {
	ipInt, err := ipToUint32(ipStr)
	if err != nil {

		return err
	}
	val := uint8(1)
	return mm.m.xdpObjs.BlocklistMap.Update(&ipInt, &val, ebpf.UpdateAny)
}

func (mm *MapManager) UnblockIP(ipStr string) error {
	ipInt, err := ipToUint32(ipStr)
	if err != nil {

		return err

	}

	return mm.m.xdpObjs.BlocklistMap.Delete(&ipInt)
}

func (mm *MapManager) ListBlockedIPs() ([]string, error) {
	var ips []string
	var key uint32
	var val uint8
	iter := mm.m.xdpObjs.BlocklistMap.Iterate()
	for iter.Next(&key, &val) {

		if val == 1 {
			ips = append(ips, uint32ToIP(key))
		}
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}
	return ips, nil

}

func (mm *MapManager) GetPacketCounters() (uint64, uint64, error) {
	var vals []uint64
	numCPUs := runtime.NumCPU()

	key := uint32(0)
	if err := mm.m.xdpObjs.PacketCounters.Lookup(&key, &vals); err != nil {
		return 0, 0, err
	}
	var passed uint64
	for i := 0; i < numCPUs && i < len(vals); i++ {
		passed += vals[i]
	}

	key = uint32(1)
	if err := mm.m.xdpObjs.PacketCounters.Lookup(&key, &vals); err != nil {

		return passed, 0, err
	}
	var dropped uint64
	for i := 0; i < numCPUs && i < len(vals); i++ {
		dropped += vals[i]
	}


	return passed, dropped, nil
}


type RlConfig struct {
	Threshold uint32
	WindowMs  uint32
}


func (mm *MapManager) UpdateConfig(threshold uint32, windowMs uint32) error {
	key := uint32(0)

	val := RlConfig{
		Threshold: threshold,
		WindowMs:  windowMs,

	}
	return mm.m.xdpObjs.RateLimitConfig.Update(&key, &val, ebpf.UpdateAny)
}

type DropEvent struct {
	SrcIP    uint32
	DstIP    uint32
	SrcPort  uint16
	DstPort  uint16

	Protocol uint8
	Reason   uint8
	_        uint16
}

func (mm *MapManager) StartEventReader(events chan<- DropEvent, done <-chan struct{}) error {
	rd, err := ringbuf.NewReader(mm.m.xdpObjs.DropEvents)
	if err != nil {
		return err

	}

	go func() {
		<-done

		rd.Close()
	}()

	go func() {
		var event DropEvent
		for {
			record, err := rd.Read()
			if err != nil {
				if errors.Is(err, ringbuf.ErrClosed) {
					return
				}
				continue
			}

			if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event); err == nil {

				select {
				case events <- event:
				case <-done:
					return
				}
			}
		}
	}()

	return nil
}

type PortStats struct {
	Packets uint64

	Bytes   uint64
}


func (mm *MapManager) GetPortStats() (map[uint16]PortStats, error) {
	stats := make(map[uint16]PortStats)

	var key uint16
	var vals []PortStats
	numCPUs := runtime.NumCPU()

	iter := mm.m.tcObjs.PortMetrics.Iterate()
	for iter.Next(&key, &vals) {
		var total PortStats

		for i := 0; i < numCPUs && i < len(vals); i++ {
			total.Packets += vals[i].Packets

			total.Bytes += vals[i].Bytes
		}
		stats[key] = total
	}


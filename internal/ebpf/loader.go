package ebpf

import (
	"errors"
	"fmt"
	"net"

	"github.com/cilium/ebpf/link"
)

type Manager struct {
	xdpObjs   XdpFirewallObjects
	tcObjs    TcMetricsObjects
	xdpLink   link.Link
	tcLink    link.Link
	iface     *net.Interface
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) LoadPrograms(ifaceName string) error {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("interface %s: %w", ifaceName, err)
	}
	m.iface = iface

	if err := LoadXdpFirewallObjects(&m.xdpObjs, nil); err != nil {
		return fmt.Errorf("loading xdp: %w", err)
	}

	if err := LoadTcMetricsObjects(&m.tcObjs, nil); err != nil {
		return fmt.Errorf("loading tc: %w", err)
	}

	return nil
}

func (m *Manager) AttachXDP(mode string) error {
	if m.iface == nil {
		return errors.New("not loaded")
	}

	opts := link.XDPOptions{
		Program:   m.xdpObjs.XdpFirewall,
		Interface: m.iface.Index,
	}

	if mode == "native" {
		opts.Flags = link.XDPGenericMode
		l, err := link.AttachXDP(opts)
		if err != nil {
			opts.Flags = link.XDPGenericMode
			l, err = link.AttachXDP(opts)
			if err != nil {
				return err
			}
			m.xdpLink = l
		} else {
			m.xdpLink = l
		}
	} else {
		opts.Flags = link.XDPGenericMode
		l, err := link.AttachXDP(opts)
		if err != nil {
			return err
		}
		m.xdpLink = l
	}

	return nil
}

func (m *Manager) AttachTC() error {
	if m.iface == nil {
		return errors.New("not loaded")
	}
	return nil
}

func (m *Manager) Close() {
	if m.xdpLink != nil {
		m.xdpLink.Close()
	}
	if m.tcLink != nil {

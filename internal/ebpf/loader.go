
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

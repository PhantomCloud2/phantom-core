package v2

import (
	"errors"
	"net"
	"strings"

	pb "github.com/phantomcloude/phantom-core/phantomrpc"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

var oldCommandServer *libbox.CommandServer

type CommandServerHandler struct {
	logger log.Logger
}

func (csh *CommandServerHandler) ServiceReload() error {
	csh.logger.Trace("Reloading service")

	if lastConfigContent == "" {
		return errors.New("no config content to reload")
	}

	SetCoreStatus(pb.CoreState_STARTING, pb.MessageType_EMPTY, "")

	err := oldCommandServer.StartOrReloadService(lastConfigContent, &libbox.OverrideOptions{})
	if err != nil {
		SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_START_SERVICE, err.Error())
		return err
	}

	SetCoreStatus(pb.CoreState_STARTED, pb.MessageType_EMPTY, "")
	return nil
}

func (csh *CommandServerHandler) GetSystemProxyStatus() (*libbox.SystemProxyStatus, error) {
	csh.logger.Trace("Getting system proxy status")
	return &libbox.SystemProxyStatus{Available: true, Enabled: false}, nil
}

func (csh *CommandServerHandler) SetSystemProxyEnabled(isEnabled bool) error {
	csh.logger.Trace("Setting system proxy status, enabled? ", isEnabled)
	return csh.ServiceReload()
}

func (csh *CommandServerHandler) ServiceStop() error {
	csh.logger.Trace("Stopping service")
	_, err := Stop()
	return err
}

func (csh *CommandServerHandler) WriteDebugMessage(message string) {
	csh.logger.Trace("Debug message: ", message)
}

func startCommandServer() error {
	if oldCommandServer != nil {
		return nil // already running
	}
	logger := coreLogFactory.NewLogger("[Command Server Handler]")
	logger.Trace("Starting command server")
	server, err := libbox.NewCommandServer(&CommandServerHandler{logger: logger}, &desktopPlatform{})
	if err != nil {
		return err
	}
	oldCommandServer = server
	if err := oldCommandServer.Start(); err != nil {
		oldCommandServer = nil
		return err
	}
	return nil
}

type desktopPlatform struct{}

type desktopInterfaceMonitor struct {
	listener libbox.InterfaceUpdateListener
}

type localStringIterator struct {
	values []string
}

func (it *localStringIterator) Len() int32 {
	return int32(len(it.values))
}

func (it *localStringIterator) HasNext() bool {
	return len(it.values) > 0
}

func (it *localStringIterator) Next() string {
	if len(it.values) == 0 {
		return ""
	}
	next := it.values[0]
	it.values = it.values[1:]
	return next
}

type localNetworkInterfaceIterator struct {
	values []*libbox.NetworkInterface
}

func (it *localNetworkInterfaceIterator) HasNext() bool {
	return len(it.values) > 0
}

func (it *localNetworkInterfaceIterator) Next() *libbox.NetworkInterface {
	if len(it.values) == 0 {
		return nil
	}
	next := it.values[0]
	it.values = it.values[1:]
	return next
}

var defaultInterfaceMonitor *desktopInterfaceMonitor

func (p *desktopPlatform) LocalDNSTransport() libbox.LocalDNSTransport { return nil }
func (p *desktopPlatform) UsePlatformAutoDetectInterfaceControl() bool { return false }
func (p *desktopPlatform) AutoDetectInterfaceControl(fd int32) error   { return nil }
func (p *desktopPlatform) OpenTun(options libbox.TunOptions) (int32, error) {
	return 0, errors.New("not supported")
}
func (p *desktopPlatform) UseProcFS() bool { return false }
func (p *desktopPlatform) FindConnectionOwner(ipProtocol int32, sourceAddress string, sourcePort int32, destinationAddress string, destinationPort int32) (*libbox.ConnectionOwner, error) {
	return nil, nil
}
func (p *desktopPlatform) StartDefaultInterfaceMonitor(listener libbox.InterfaceUpdateListener) error {
	defaultInterfaceMonitor = &desktopInterfaceMonitor{listener: listener}
	if defaultName, defaultIndex, ok := pickDefaultInterface(); ok {
		listener.UpdateDefaultInterface(defaultName, int32(defaultIndex), false, false)
	}
	return nil
}
func (p *desktopPlatform) CloseDefaultInterfaceMonitor(listener libbox.InterfaceUpdateListener) error {
	if defaultInterfaceMonitor != nil && defaultInterfaceMonitor.listener == listener {
		defaultInterfaceMonitor = nil
	}
	return nil
}
func (p *desktopPlatform) GetInterfaces() (libbox.NetworkInterfaceIterator, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	networkInterfaces := make([]libbox.NetworkInterface, 0, len(interfaces))
	for _, iface := range interfaces {
		addresses := make([]string, 0)
		addrList, _ := iface.Addrs()
		for _, addr := range addrList {
			address := addr.String()
			if strings.Contains(address, "/") {
				addresses = append(addresses, address)
			}
		}
		networkInterfaces = append(networkInterfaces, libbox.NetworkInterface{
			Index:     int32(iface.Index),
			MTU:       int32(iface.MTU),
			Name:      iface.Name,
			Addresses: &localStringIterator{values: addresses},
			Flags:     int32(iface.Flags),
			Type:      libbox.InterfaceTypeOther,
			DNSServer: &localStringIterator{values: []string{}},
			Metered:   false,
		})
	}
	pointerInterfaces := make([]*libbox.NetworkInterface, 0, len(networkInterfaces))
	for i := range networkInterfaces {
		pointerInterfaces = append(pointerInterfaces, &networkInterfaces[i])
	}
	return &localNetworkInterfaceIterator{values: pointerInterfaces}, nil
}
func (p *desktopPlatform) UnderNetworkExtension() bool                              { return false }
func (p *desktopPlatform) IncludeAllNetworks() bool                                 { return false }
func (p *desktopPlatform) ReadWIFIState() *libbox.WIFIState                         { return nil }
func (p *desktopPlatform) SystemCertificates() libbox.StringIterator                { return nil }
func (p *desktopPlatform) ClearDNSCache()                                           {}
func (p *desktopPlatform) SendNotification(notification *libbox.Notification) error { return nil }

func pickDefaultInterface() (string, int, bool) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", 0, false
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil || len(addresses) == 0 {
			continue
		}
		return iface.Name, iface.Index, true
	}
	return "", 0, false
}

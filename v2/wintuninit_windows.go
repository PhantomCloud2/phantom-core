//go:build windows

package v2

import (
	tun "github.com/sagernet/sing-tun"
)

func init() {
	tun.TunnelType = "phantom"
}

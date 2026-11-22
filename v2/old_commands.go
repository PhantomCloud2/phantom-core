package v2

import (
	"github.com/sagernet/sing-box/experimental/libbox"
)

// Legacy integer values hardcoded by the Flutter client.
// The Flutter client does not use libbox constants directly; it sends the
// integer values from an older libbox version, so we match them explicitly.
const (
	flutterCommandStatus         int32 = 1  // libbox.CommandStatus
	flutterCommandGroup          int32 = 5  // old CommandGroup (now CommandGroup=2)
	flutterCommandGroupInfoOnly  int32 = 13 // old CommandGroupInfoOnly (removed; mapped to CommandGroup)
)

var (
	oldStatusClient        *libbox.CommandClient
	oldGroupClient         *libbox.CommandClient
	oldGroupInfoOnlyClient *libbox.CommandClient
)

func replaceCommandClient(slot **libbox.CommandClient, next *libbox.CommandClient) error {
	if *slot != nil {
		_ = (*slot).Disconnect()
		*slot = nil
	}
	if err := next.Connect(); err != nil {
		_ = next.Disconnect()
		return err
	}
	*slot = next
	return nil
}

func disconnectCommandClient(slot **libbox.CommandClient) error {
	if *slot == nil {
		return nil
	}
	err := (*slot).Disconnect()
	*slot = nil
	return err
}

func StartCommand(command int32, port int64) error {
	switch command {
	case flutterCommandStatus:
		statusCmdOpts := &libbox.CommandClientOptions{
			StatusInterval: 1000000000, // 1000ms debounce
		}
		statusCmdOpts.AddCommand(libbox.CommandStatus)
		return replaceCommandClient(&oldStatusClient, libbox.NewCommandClient(
			&OldCommandClientHandler{
				port:   port,
				logger: coreLogFactory.NewLogger("[Status Command Client]"),
			},
			statusCmdOpts,
		))
	case flutterCommandGroup:
		groupCmdOpts := &libbox.CommandClientOptions{
			StatusInterval: 300000000, // 300ms debounce
		}
		groupCmdOpts.AddCommand(libbox.CommandGroup)
		return replaceCommandClient(&oldGroupClient, libbox.NewCommandClient(
			&OldCommandClientHandler{
				port:   port,
				logger: coreLogFactory.NewLogger("[Group Command Client]"),
			},
			groupCmdOpts,
		))
	case flutterCommandGroupInfoOnly:
		// CommandGroupInfoOnly has been removed from libbox; map to CommandGroup
		// (new API only has a streaming subscription, not a one-shot fetch).
		groupInfoOnlyOpts := &libbox.CommandClientOptions{
			StatusInterval: 300000000, // 300ms debounce
		}
		groupInfoOnlyOpts.AddCommand(libbox.CommandGroup)
		return replaceCommandClient(&oldGroupInfoOnlyClient, libbox.NewCommandClient(
			&OldCommandClientHandler{
				port:   port,
				logger: coreLogFactory.NewLogger("[GroupInfoOnly Command Client]"),
			},
			groupInfoOnlyOpts,
		))
	}

	return nil
}

func StopCommand(command int32) error {
	switch command {
	case flutterCommandStatus:
		return disconnectCommandClient(&oldStatusClient)
	case flutterCommandGroup:
		return disconnectCommandClient(&oldGroupClient)
	case flutterCommandGroupInfoOnly:
		return disconnectCommandClient(&oldGroupInfoOnlyClient)
	}

	return nil
}

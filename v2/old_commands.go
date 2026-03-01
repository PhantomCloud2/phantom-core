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

func StartCommand(command int32, port int64) error {
	switch command {
	case flutterCommandStatus:
		statusCmdOpts := &libbox.CommandClientOptions{
			StatusInterval: 1000000000, // 1000ms debounce
		}
		statusCmdOpts.AddCommand(libbox.CommandStatus)
		oldStatusClient = libbox.NewCommandClient(
			&OldCommandClientHandler{
				port:   port,
				logger: coreLogFactory.NewLogger("[Status Command Client]"),
			},
			statusCmdOpts,
		)

		return oldStatusClient.Connect()
	case flutterCommandGroup:
		groupCmdOpts := &libbox.CommandClientOptions{
			StatusInterval: 300000000, // 300ms debounce
		}
		groupCmdOpts.AddCommand(libbox.CommandGroup)
		oldGroupClient = libbox.NewCommandClient(
			&OldCommandClientHandler{
				port:   port,
				logger: coreLogFactory.NewLogger("[Group Command Client]"),
			},
			groupCmdOpts,
		)

		return oldGroupClient.Connect()
	case flutterCommandGroupInfoOnly:
		// CommandGroupInfoOnly has been removed from libbox; map to CommandGroup
		// (new API only has a streaming subscription, not a one-shot fetch).
		groupInfoOnlyOpts := &libbox.CommandClientOptions{
			StatusInterval: 300000000, // 300ms debounce
		}
		groupInfoOnlyOpts.AddCommand(libbox.CommandGroup)
		oldGroupInfoOnlyClient = libbox.NewCommandClient(
			&OldCommandClientHandler{
				port:   port,
				logger: coreLogFactory.NewLogger("[GroupInfoOnly Command Client]"),
			},
			groupInfoOnlyOpts,
		)

		return oldGroupInfoOnlyClient.Connect()
	}

	return nil
}

func StopCommand(command int32) error {
	switch command {
	case flutterCommandStatus:
		err := oldStatusClient.Disconnect()
		oldStatusClient = nil
		return err
	case flutterCommandGroup:
		err := oldGroupClient.Disconnect()
		oldGroupClient = nil
		return err
	case flutterCommandGroupInfoOnly:
		err := oldGroupInfoOnlyClient.Disconnect()
		oldGroupInfoOnlyClient = nil
		return err
	}

	return nil
}

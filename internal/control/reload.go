package control

import (
	"encoding/json"
	"fmt"
	"net"

	"brabble/internal/config"

	"github.com/spf13/cobra"
)

// NewReloadCmd asks the daemon to reload config.
func NewReloadCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "reload",
		Short: "Reload config in the running daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			conn, err := net.Dial("unix", cfg.Paths.SocketPath)
			if err != nil {
				return fmt.Errorf("cannot connect to daemon: %w", err)
			}
			defer conn.Close()
			req := Request{Op: "reload"}
			if err := json.NewEncoder(conn).Encode(req); err != nil {
				return err
			}
			var resp SimpleResponse
			if err := json.NewDecoder(conn).Decode(&resp); err != nil {
				return err
			}
			if !resp.OK {
				return fmt.Errorf("reload failed: %s", resp.Message)
			}
			fmt.Println("reload ok:", resp.Message)
			return nil
		},
	}
}

package control

import (
	"fmt"
	"os"
	"strings"

	"brabble/internal/config"
	"brabble/internal/service"

	"github.com/spf13/cobra"
)

func newServiceInstallCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install user launchd service (macOS)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			envPairs, _ := cmd.Flags().GetStringArray("env")
			env := make(map[string]string)
			for _, p := range envPairs {
				parts := strings.SplitN(p, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("bad env %q, want KEY=VAL", p)
				}
				env[parts[0]] = parts[1]
			}
			params := service.LaunchdParams{
				Label:  "com.brabble.agent",
				Binary: exe,
				Config: cfg.Paths.ConfigPath,
				Log:    cfg.Paths.LogPath,
				Env:    env,
			}
			path, err := service.WritePlist(params)
			if err != nil {
				return err
			}
			fmt.Printf("launchd plist written: %s\n", path)
			fmt.Println("Load:   launchctl load -w", path)
			fmt.Printf("Start:  launchctl kickstart gui/$(id -u)/%s\n", params.Label)
			fmt.Printf("Stop:   launchctl bootout gui/$(id -u)/%s\n", params.Label)
			return nil
		},
	}
	cmd.Flags().StringArray("env", nil, "Env to set in launchd plist (KEY=VAL)")
	return cmd
}

func newServiceUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove user launchd plist (macOS)",
		RunE: func(cmd *cobra.Command, args []string) error {
			plist := service.LaunchdPath("com.brabble.agent")
			_ = os.Remove(plist)
			fmt.Printf("removed %s (if present); unload manually with: launchctl bootout gui/$(id -u) %s\n", plist, plist)
			return nil
		},
	}
}

func newServiceStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show launchd plist path and whether it exists",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, ok := service.Status("com.brabble.agent")
			fmt.Printf("plist: %s\n", path)
			if ok {
				fmt.Println("status: present (load with: launchctl load -w", path, ")")
			} else {
				fmt.Println("status: missing (install via: brabble service install)")
			}
			return nil
		},
	}
}

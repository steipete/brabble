package control

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"brabble/internal/config"
	"brabble/internal/service"

	"github.com/spf13/cobra"
)

// NewServiceRootCmd groups service subcommands.
func NewServiceRootCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage user service (launchd on macOS, systemd on Linux)",
	}
	cmd.AddCommand(newServiceInstallCmd(cfgPath))
	cmd.AddCommand(newServiceUninstallCmd())
	cmd.AddCommand(newServiceStatusCmd())
	return cmd
}

func newServiceInstallCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install user service definition",
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
			if runtime.GOOS == "linux" {
				path, err := service.WriteSystemdUnit(params)
				if err != nil {
					return err
				}
				fmt.Printf("systemd unit written: %s\n", path)
				fmt.Println("Load: systemctl --user daemon-reload")
				fmt.Println("Start: systemctl --user enable --now brabble.service")
				fmt.Println("Stop: systemctl --user disable --now brabble.service")
				return nil
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
		Short: "Remove user service definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS == "linux" {
				path, err := service.SystemdPath()
				if err != nil {
					return err
				}
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return err
				}
				fmt.Printf("removed %s; stop any running service with systemctl --user disable --now brabble.service, then systemctl --user daemon-reload\n", path)
				return nil
			}
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
		Short: "Show service definition path and whether it exists",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS == "linux" {
				path, err := service.SystemdPath()
				if err != nil {
					return err
				}
				_, err = os.Stat(path)
				if err != nil && !os.IsNotExist(err) {
					return err
				}
				fmt.Printf("unit: %s\ninstalled: %t\nRuntime status: systemctl --user status brabble.service\n", path, err == nil)
				return nil
			}
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

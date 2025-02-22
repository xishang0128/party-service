package service

import (
	"log"

	"github.com/spf13/cobra"
)

var ServiceCmd = &cobra.Command{
	Use: "service",
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the service",
	Run: func(cmd *cobra.Command, args []string) {
		if err := InstallService(); err != nil {
			log.Fatal(err)
		}
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the service",
	Run: func(cmd *cobra.Command, args []string) {
		if err := UninstallService(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	ServiceCmd.AddCommand(installCmd, uninstallCmd)
}

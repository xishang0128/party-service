package route

import (
	"log"
	"sparkle-service/config"

	"github.com/spf13/cobra"
)

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Start server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.Initialize("", ""); err != nil {
			log.Fatal(err)
		}
		if err := start(); err != nil {
			log.Fatal(err)
		}
	},
}

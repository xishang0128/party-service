package route

import (
	"log"

	"party-service/data"

	"github.com/spf13/cobra"
)

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Start server",
	Run: func(cmd *cobra.Command, args []string) {
		dataFile, _ := cmd.Flags().GetString("data")
		if dataFile == "" {
			log.Fatal("data file is required")
		}
		data.SetFilePath(dataFile)
		if err := start(); err != nil {
			log.Fatal(err)
		}
	},
}

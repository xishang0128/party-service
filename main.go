package main

import (
	"log"

	"sparkle-service/route"
	"sparkle-service/service"

	"github.com/spf13/cobra"
)

var configFile string

var mainCmd = &cobra.Command{
	Use: "sparkle-service",
}

func init() {
	mainCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "set config file path")

	mainCmd.AddCommand(route.ServerCmd)
	mainCmd.AddCommand(service.ServiceCmd)
}

func main() {
	if err := mainCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

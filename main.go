package main

import (
	"log"

	"party-service/route"
	"party-service/service"

	"github.com/spf13/cobra"
)

var dataFile string

var mainCmd = &cobra.Command{
	Use: "party-service",
}

func init() {
	mainCmd.PersistentFlags().StringVarP(&dataFile, "data", "d", "", "set data file path")

	mainCmd.AddCommand(route.ServerCmd)
	mainCmd.AddCommand(service.ServiceCmd)
}

func main() {
	if err := mainCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

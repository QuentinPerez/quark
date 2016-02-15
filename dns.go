package main

import (
	"github.com/spf13/cobra"
)

var (
	cmdDns = &cobra.Command{
		Use: "dns",
		Run: showUsage,
	}
)

func init() {
	cmdMain.AddCommand(cmdDns)
}

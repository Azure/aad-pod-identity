package main

import (
	"fmt"
	"os"

	"github.com/golang/glog"

	"github.com/spf13/cobra"
)

// ctlCmd is a representation of the control loop (ctl) command.
var execute = &cobra.Command{
	Use:   "nmi",
	Short: "Node Managed Identity",
	Long:  `Microsoft`,
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	},
}

// init will bootstrap the command with command line flags a user can optionally define
func init() {
}

func main() {
	Execute()
}

// Execute passes control to the root command and handles the error it may return.
func Execute() {
	if err := execute.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

// run starts the nmi operations
func run(cmd *cobra.Command, args []string) {
	defer glog.Flush()
	glog.Info("starting nmi process")
}

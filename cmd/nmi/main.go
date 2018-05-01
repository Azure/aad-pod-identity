package main

import (
	"github.com/golang/glog"

	"github.com/spf13/cobra"
)

// ctlCmd is a representation of the control loop (ctl) command.
var ctlCmd = &cobra.Command{
	Use:   "ctl",
	Short: "HCP Control Loop",
	Long:  `A Microsoft Product.`,
	Run: func(cmd *cobra.Command, args []string) {
		runCtlLoop(cmd, args)
	},
}

func main() {
	defer glog.Flush()
	glog.Info("starting nmi process")
}

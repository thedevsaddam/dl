package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cmdVersion = &cobra.Command{
	Use:   "version",
	Short: "Display application version",
	Long:  `Display application version`,
	Run:   version,
}

func init() {
	cmdDL.AddCommand(cmdVersion)
}

func version(cmd *cobra.Command, args []string) {
	fmt.Println(logo)
	fmt.Println("Version:", Version)
	fmt.Println("Git commit:", GitCommit)
	fmt.Println("Build date:", BuildDate)
}

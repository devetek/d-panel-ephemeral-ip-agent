package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "marijan",
	Short: "Marijan Tunnel Manager",
	Long: `
marijan is a simple CLI tool to manage your tunnels, simplify the process of managing your tunnels.

Full documentation is available at: https://cloud.terpusat.com/docs/tunnels/marijan
`,
}

func init() {
	rootCmd.AddCommand(
		versionCmd(),
		runCmd(),
	)
}

func Execute() {
	rootCmd.Version = currentVersion
	cobra.CheckErr(rootCmd.Execute())
}

func main() {
	Execute()
}

/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/takutakahashi/node-tainter/pkg/manager"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		scripts, err := cmd.Flags().GetStringArray("script-path")
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		if scripts == nil {
			scripts = []string{"/tmp/node-tainter/default.sh"}
		}
		daemon, err := cmd.Flags().GetBool("daemon")
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		once, err := cmd.Flags().GetBool("once")
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		taint, err := cmd.Flags().GetString("taint")
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		node, err := cmd.Flags().GetString("node-name")
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		maxTaintedNodeCount, err := cmd.Flags().GetInt("max-tainted-nodes")
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		m := manager.Manager{
			ScriptPath:          scripts,
			Daemon:              daemon && !once,
			Taint:               taint,
			Node:                node,
			DryRun:              os.Getenv("DRY_RUN") == "true",
			MaxTaintedNodeCount: maxTaintedNodeCount,
		}
		if err := m.Execute(); err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/takutakahashi/node-tainter/pkg/config"
	"github.com/takutakahashi/node-tainter/pkg/manager"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
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
		once, err := cmd.Flags().GetBool("once")
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		dryrun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		node, err := cmd.Flags().GetString("node-name")
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		clientset, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		configInputs, err := cmd.Flags().GetStringArray("config")
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		configs := []*config.Config{}
		for _, configPath := range configInputs {
			config, err := config.LoadConfig(configPath)
			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
			configs = append(configs, config)
		}
		ctx := cmd.Context()
		m := manager.NewManager(configs, node, once, dryrun, clientset)
		if err := m.Execute(ctx); err != nil {
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

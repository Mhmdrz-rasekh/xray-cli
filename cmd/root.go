package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "xray-cli",
	Short: "A fast and powerful CLI client for Xray",
	Long:  `An interactive terminal dashboard and CLI tool to manage subscriptions and run Xray configurations.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// تنظیمات اولیه ریشه در صورت نیاز
}

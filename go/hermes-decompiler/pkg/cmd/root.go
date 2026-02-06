package cmd

import "github.com/spf13/cobra"

var (
	rootCmd = &cobra.Command{
		Use:   "hermes-decompiler",
		Short: "Hermes decompiler",
		Long:  "Hermes decompiler is a tool for decompiling Hermes bytecode.",
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(genopcodesCmd)
	rootCmd.AddCommand(disassembleHbcFileCmd)
	rootCmd.AddCommand(maintainDatabaseCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(packagesCmd)
}

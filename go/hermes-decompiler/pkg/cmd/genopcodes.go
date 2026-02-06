package cmd

import (
	"hermes-decompiler/pkg/utils"

	"github.com/spf13/cobra"
)

var (
	// flags
	updateFlag bool

	genopcodesCmd = &cobra.Command{
		Use:   "genopcodes",
		Short: "Generate opcodes from .def file",
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.GenerateBytecodeDefinitionFiles(updateFlag)
			return nil
		},
	}
)

func init() {
	genopcodesCmd.Flags().BoolVar(&updateFlag, "update", false, "download updates from hermes")
}

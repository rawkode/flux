package cmd

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	_ "github.com/influxdata/flux/builtin"
	"github.com/influxdata/flux/lang"
	"github.com/spf13/cobra"
)

// compileCmd represents the compile command
var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile a Flux script",
	Long:  "Compile a Flux script from string or file (use @ as prefix to the file)",
	Args:  cobra.ExactArgs(1),
	RunE:  compile,
}

var prettyPrint bool

func init() {
	compileCmd.Flags().BoolVarP(&prettyPrint, "pretty-print", "p", false, "pretty print the compiled query")
}

func init() {
	rootCmd.AddCommand(compileCmd)
}

func compile(cmd *cobra.Command, args []string) error {
	scriptSource := args[0]

	var script string
	if scriptSource[0] == '@' {
		scriptBytes, err := ioutil.ReadFile(scriptSource[1:])
		if err != nil {
			return err
		}
		script = string(scriptBytes)
	} else {
		script = scriptSource
	}

	c := lang.FluxCompiler{
		Query: script,
	}

	spec, err := c.Compile(context.Background())
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	if prettyPrint {
		enc.SetIndent("", " ")
	}
	if err := enc.Encode(spec); err != nil {
		return err
	}

	return nil
}

// Copyright 2016-2018, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/pulumi/pulumi/pkg/apitype"
	"github.com/pulumi/pulumi/pkg/backend"
	"github.com/pulumi/pulumi/pkg/backend/display"
	"github.com/pulumi/pulumi/pkg/util/cmdutil"
)

func newStackTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage stack tags",
		// TODO Add Long: description.
		Args: cmdutil.NoArgs,
	}

	cmd.AddCommand(newStackTagGetCmd())
	cmd.AddCommand(newStackTagLsCmd())
	cmd.AddCommand(newStackTagRmCmd())
	cmd.AddCommand(newStackTagSetCmd())

	return cmd
}

func newStackTagGetCmd() *cobra.Command {
	var stackName string
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get a single stack tag value",
		Args:  cmdutil.SpecificArgs([]string{"name"}),
		Run: cmdutil.RunFunc(func(cmd *cobra.Command, args []string) error {
			opts := display.Options{
				Color: cmdutil.GetGlobalColorization(),
			}

			// Fetch the current stack and import a deployment.
			s, err := requireStack(stackName, false, opts, true /*setCurrent*/)
			if err != nil {
				return err
			}

			tags, err := backend.GetStackTags()
			if err != nil {
				return err
			}

			name := args[0]

			if value, ok := tags[name]; ok {
				fmt.Printf("%v\n", value)
				return nil
			}

			return errors.Errorf(
				"stack tag '%s' not found for stack '%s'", name, s.Ref())
		}),
	}

	cmd.PersistentFlags().StringVarP(
		&stackName, "stack", "s", "", "The name of the stack to operate on. Defaults to the current stack")

	return cmd
}

func newStackTagLsCmd() *cobra.Command {
	var stackName string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List all stack tags",
		Args:  cmdutil.NoArgs,
		Run: cmdutil.RunFunc(func(cmd *cobra.Command, args []string) error {
			opts := display.Options{
				Color: cmdutil.GetGlobalColorization(),
			}

			// Fetch the current stack and import a deployment.
			_, err := requireStack(stackName, false, opts, true /*setCurrent*/)
			if err != nil {
				return err
			}
			//stackName := s.Ref().Name()

			tags, err := backend.GetStackTags()
			if err != nil {
				return err
			}

			if jsonOut {
				return printJSON(tags)
			}

			printStackTags(tags)
			return nil
		}),
	}

	cmd.PersistentFlags().StringVarP(
		&stackName, "stack", "s", "", "The name of the stack to operate on. Defaults to the current stack")
	cmd.PersistentFlags().BoolVarP(
		&jsonOut, "json", "j", false, "Emit stack tags as JSON")

	return cmd
}

func printStackTags(tags map[apitype.StackTagName]string) {
	var keys []string
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	rows := []cmdutil.TableRow{}
	for _, key := range keys {
		rows = append(rows, cmdutil.TableRow{Columns: []string{key, tags[key]}})
	}

	cmdutil.PrintTable(cmdutil.Table{
		Headers: []string{"KEY", "VALUE"},
		Rows:    rows,
	})
}

func newStackTagRmCmd() *cobra.Command {
	var stackName string
	cmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a stack tag",
		Args:  cmdutil.SpecificArgs([]string{"name"}),
		Run: cmdutil.RunFunc(func(cmd *cobra.Command, args []string) error {
			// opts := display.Options{
			// 	Color: cmdutil.GetGlobalColorization(),
			// }

			// // Fetch the current stack and import a deployment.
			// s, err := requireStack(stackName, false, opts, true /*setCurrent*/)
			// if err != nil {
			// 	return err
			// }

			// TODO
			return nil
		}),
	}

	cmd.PersistentFlags().StringVarP(
		&stackName, "stack", "s", "", "The name of the stack to operate on. Defaults to the current stack")

	return cmd
}

func newStackTagSetCmd() *cobra.Command {
	var stackName string
	cmd := &cobra.Command{
		Use:   "set <name> <value>",
		Short: "Set a stack tag",
		Args:  cmdutil.SpecificArgs([]string{"name", "value"}),
		Run: cmdutil.RunFunc(func(cmd *cobra.Command, args []string) error {
			// opts := display.Options{
			// 	Color: cmdutil.GetGlobalColorization(),
			// }

			// // Fetch the current stack and import a deployment.
			// s, err := requireStack(stackName, false, opts, true /*setCurrent*/)
			// if err != nil {
			// 	return err
			// }

			// TODO
			return nil
		}),
	}

	cmd.PersistentFlags().StringVarP(
		&stackName, "stack", "s", "", "The name of the stack to operate on. Defaults to the current stack")

	return cmd
}

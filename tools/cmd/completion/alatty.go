// License: GPLv3 Copyright: 2022, Kovid Goyal, <kovid at kovidgoyal.net>

package completion

import (
	"fmt"

	"alatty/tools/cli"
)

var _ = fmt.Print

func EntryPoint(tool_root *cli.Command) {
	tool_root.AddSubCommand(&cli.Command{
		Name: "__complete__", Hidden: true,
		Usage:            "output_type [shell state...]",
		ShortDescription: "Generate completions for alatty commands",
		HelpText:         "Generate completion candidates for alatty commands. The command line is read from STDIN. output_type can be one of the supported shells: :code:`zsh`, :code:`fish`, :code:`bash`, or :code:`setup` for completion setup script following with the shell name, or :code:`json` for JSON output.",
		Run: func(cmd *cli.Command, args []string) (ret int, err error) {
			return ret, nil
		},
	})
}

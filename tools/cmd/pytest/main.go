// License: GPLv3 Copyright: 2023, Kovid Goyal, <kovid at kovidgoyal.net>

package pytest

import (
	"fmt"

	"github.com/kovidgoyal/alatty/kittens/ssh"
	"github.com/kovidgoyal/alatty/tools/cli"
	"github.com/kovidgoyal/alatty/tools/utils/shm"
)

var _ = fmt.Print

func EntryPoint(root *cli.Command) {
	root = root.AddSubCommand(&cli.Command{
		Name:   "__pytest__",
		Hidden: true,
	})
	shm.TestEntryPoint(root)
	ssh.TestEntryPoint(root)
}

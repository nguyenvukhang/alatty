package panel

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kovidgoyal/alatty/tools/cli"
	"github.com/kovidgoyal/alatty/tools/utils"

	"golang.org/x/sys/unix"
)

var _ = fmt.Print

func complete_alatty_listen_on(completions *cli.Completions, word string, arg_num int) {
	if !strings.Contains(word, ":") {
		mg := completions.AddMatchGroup("Address family")
		mg.NoTrailingSpace = true
		for _, q := range []string{"unix:", "tcp:"} {
			if strings.HasPrefix(q, word) {
				mg.AddMatch(q)
			}
		}
	} else if strings.HasPrefix(word, "unix:") && !strings.HasPrefix(word, "unix:@") {
		cli.FnmatchCompleter("UNIX sockets", cli.CWD, "*")(completions, word[len("unix:"):], arg_num)
		completions.AddPrefixToAllMatches("unix:")
	}
}

var CompleteAlattyListenOn = complete_alatty_listen_on

func GetQuickAccessAlattyExe() (alatty_exe string, err error) {
	if alatty_exe, err = filepath.EvalSymlinks(utils.AlattyExe()); err != nil {
		return "", fmt.Errorf("Failed to find path to the alatty executable, this kitten requires the alatty executable to function. The alatty executable or a symlink to it must be placed in the same directory as the kitten executable. Error: %w", err)
	}
	if runtime.GOOS == "darwin" {
		q := filepath.Join(filepath.Dir(filepath.Dir(alatty_exe)), "alatty-quick-access.app", "Contents", "MacOS", "alatty-quick-access")
		if err := unix.Access(q, unix.X_OK); err == nil {
			alatty_exe = q
		}
	}
	return alatty_exe, nil

}

func main(cmd *cli.Command, o *Options, args []string) (rc int, err error) {
	alatty_exe, err := GetQuickAccessAlattyExe()
	if err != nil {
		return 1, err
	}
	argv := []string{alatty_exe, "+kitten", "panel"}
	argv = append(argv, o.AsCommandLine()...)
	err = unix.Exec(alatty_exe, append(argv, args...), os.Environ())
	rc = 1
	return
}

func EntryPoint(parent *cli.Command) {
	create_cmd(parent, main)
}

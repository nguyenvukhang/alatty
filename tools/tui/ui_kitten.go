// License: GPLv3 Copyright: 2023, Kovid Goyal, <kovid at kovidgoyal.net>

package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"alatty/tools/cli"
	"alatty/tools/utils"
	"alatty/tools/utils/base85"
)

var _ = fmt.Print

var RunningAsUI = sync.OnceValue(func() bool {
	defer func() { os.Unsetenv("KITTEN_RUNNING_AS_UI") }()
	return os.Getenv("KITTEN_RUNNING_AS_UI") != ""
})

func PrepareRootCmd(root *cli.Command) {
	if RunningAsUI() {
		root.CallbackOnError = func(cmd *cli.Command, err error, during_parsing bool, exit_code int) int {
			cli.ShowError(err)
			HoldTillEnter(true)
			return exit_code
		}
	}
}

func KittenOutputSerializer() func(any) (string, error) {
	if RunningAsUI() {
		return func(what any) (string, error) {
			data, err := json.Marshal(what)
			if err != nil {
				return "", err
			}
			return "\x1bP@kitty-kitten-result|" + base85.EncodeToString(data) + "\x1b\\", nil
		}
	}
	return func(what any) (string, error) {
		if sval, ok := what.(string); ok {
			return sval, nil
		}
		data, err := json.MarshalIndent(what, "", "  ")
		if err != nil {
			return "", err
		}
		return utils.UnsafeBytesToString(data), nil
	}
}

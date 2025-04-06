// License: GPLv3 Copyright: 2022, Kovid Goyal, <kovid at kovidgoyal.net>

package cli

import (
	"encoding/json"
)

func json_input_parser(data []byte, shell_state map[string]string) ([][]string, error) {
	ans := make([][]string, 0, 32)
	err := json.Unmarshal(data, &ans)
	return ans, err
}

func json_output_serializer(completions []*Completions, shell_state map[string]string) ([]byte, error) {
	return json.Marshal(completions)
}

type completion_script_func func(commands []string) (string, error)
type parser_func func(data []byte, shell_state map[string]string) ([][]string, error)
type serializer_func func(completions []*Completions, shell_state map[string]string) ([]byte, error)

var completion_scripts = make(map[string]completion_script_func, 4)
var input_parsers = make(map[string]parser_func, 4)
var output_serializers = make(map[string]serializer_func, 4)
var init_completions = make(map[string]func(*Completions), 4)

func init() {
	input_parsers["json"] = json_input_parser
	output_serializers["json"] = json_output_serializer
}

var registered_exes []func(root *Command)

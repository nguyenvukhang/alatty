package choose_fonts

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/kovidgoyal/alatty/tools/utils"
)

var _ = fmt.Print

type alatty_font_backend_type struct {
	from                    io.ReadCloser
	to                      io.WriteCloser
	json_decoder            *json.Decoder
	cmd                     *exec.Cmd
	stderr                  strings.Builder
	lock                    sync.Mutex
	r                       io.ReadCloser
	w                       io.WriteCloser
	wait_for_exit           chan error
	started, exited, failed bool
	timeout                 time.Duration
}

func (k *alatty_font_backend_type) start() (err error) {
	exe := utils.AlattyExe()
	if exe == "" {
		exe = utils.Which("alatty")
	}
	if exe == "" {
		return fmt.Errorf("Failed to find the alatty executable, this kitten requires the alatty executable to be present. You can use the environment variable ALATTY_PATH_TO_ALATTY_EXE to specify the path to the alatty executable")
	}

	k.cmd = exec.Command(exe, "+runpy", "from kittens.choose_fonts.backend import main; main()")
	k.cmd.Stderr = &k.stderr

	if k.r, k.to, err = os.Pipe(); err != nil {
		return err
	}
	k.cmd.Stdin = k.r
	if k.from, k.w, err = os.Pipe(); err != nil {
		return err
	}
	k.cmd.Stdout = k.w
	k.json_decoder = json.NewDecoder(k.from)
	if err = k.cmd.Start(); err != nil {
		return err
	}
	k.started = true
	k.timeout = 60 * time.Second
	k.wait_for_exit = make(chan error)
	go func() {
		k.wait_for_exit <- k.cmd.Wait()
	}()
	return
}

var alatty_font_backend alatty_font_backend_type

func (k *alatty_font_backend_type) send(v any) error {
	if k.to == nil {
		return fmt.Errorf("Trying to send data when to pipe is nil")
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("Could not encode message to alatty with error: %w", err)
	}
	c := make(chan error)
	go func() {
		if _, err = k.to.Write(data); err != nil {
			c <- fmt.Errorf("Failed to send message to alatty with I/O error: %w", err)
			return
		}
		if _, err = k.to.Write([]byte{'\n'}); err != nil {
			c <- fmt.Errorf("Failed to send message to alatty with I/O error: %w", err)
			return
		}
		c <- nil
	}()
	select {
	case err := <-c:
		return err
	case <-time.After(k.timeout):
		return fmt.Errorf("Timed out waiting to write to alatty font backend after %v", k.timeout)
	case err := <-k.wait_for_exit:
		k.exited = true
		if err == nil {
			err = fmt.Errorf("alatty font backend exited with no error while waiting for a response from it")
		} else {
			k.failed = true
		}
		return err
	}
}

func (k *alatty_font_backend_type) query(action string, cmd map[string]any, result any) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	if cmd == nil {
		cmd = make(map[string]any)
	}
	cmd["action"] = action
	if err := k.send(cmd); err != nil {
		return err
	}
	c := make(chan error)
	go func() {
		if err := k.json_decoder.Decode(result); err != nil {
			c <- fmt.Errorf("Failed to decode JSON from alatty with error: %w", err)
		}
		c <- nil
	}()
	select {
	case err := <-c:
		return err
	case <-time.After(k.timeout):
		return fmt.Errorf("Timed out waiting for response from alatty font backend after %v", k.timeout)
	case err := <-k.wait_for_exit:
		k.exited = true
		if err == nil {
			err = fmt.Errorf("alatty font backed exited with no error while waiting for a response from it")
		} else {
			k.failed = true
		}
		return err
	}
}

func (k *alatty_font_backend_type) release() (err error) {
	if k.r != nil {
		k.r.Close()
		k.r = nil
	}
	if k.to != nil {
		k.to.Close()
		k.to = nil
	}
	if k.w != nil {
		k.w.Close()
		k.w = nil
	}
	if k.from != nil {
		k.from.Close()
		k.from = nil
	}
	if k.started && !k.exited {
		timeout := 2 * time.Second
		select {
		case err = <-k.wait_for_exit:
			k.exited = true
			if err != nil {
				k.failed = true
			}
		case <-time.After(timeout):
			k.failed = true
			err = fmt.Errorf("Timed out waiting for alatty font backend to exit for %v", timeout)
		}
	}
	os.Stderr.WriteString(k.stderr.String())
	return
}

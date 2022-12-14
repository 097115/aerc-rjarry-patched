package lib

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/log"
)

func XDGOpen(uri string) error {
	return XDGOpenMime(uri, "", nil)
}

func XDGOpenMime(
	uri string, mimeType string, args []string,
) error {
	if len(args) == 0 {
		// no explicit command provided, lookup opener from mime type
		opener, ok := config.Openers[mimeType]
		if ok {
			args = opener
		} else {
			// no opener defined in config, fallback to default
			if runtime.GOOS == "darwin" {
				args = append(args, "open")
			} else {
				args = append(args, "xdg-open")
			}
		}
	}

	i := 0
	for ; i < len(args); i++ {
		if strings.Contains(args[i], "{}") {
			break
		}
	}
	if i < len(args) {
		// found {} placeholder in args, replace with uri
		args[i] = strings.Replace(args[i], "{}", uri, 1)
	} else {
		// no {} placeholder in args, add uri at the end
		args = append(args, uri)
	}

	log.Tracef("running command: %v", args)
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	log.Debugf("command: %v exited. err=%v out=%s", args, err, out)
	if err != nil {
		return fmt.Errorf("%v: %w", args, err)
	}
	return nil
}

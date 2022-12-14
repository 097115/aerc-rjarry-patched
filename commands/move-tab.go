package commands

import (
	"fmt"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type MoveTab struct{}

func init() {
	register(MoveTab{})
}

func (MoveTab) Aliases() []string {
	return []string{"move-tab"}
}

func (MoveTab) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (MoveTab) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("Usage: %s [+|-]<index>", args[0])
	}

	joinedArgs := strings.Join(args[1:], "")

	n, err := strconv.Atoi(joinedArgs)
	if err != nil {
		return fmt.Errorf("failed to parse index argument: %w", err)
	}

	var relative bool
	if strings.HasPrefix(joinedArgs, "+") || strings.HasPrefix(joinedArgs, "-") {
		relative = true
	}
	aerc.MoveTab(n, relative)

	return nil
}

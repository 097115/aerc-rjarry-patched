package commands

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/commands/mode"
)

type Quit struct {
	Force bool `opt:"-f" desc:"Force quit even if a task is pending."`
}

func init() {
	Register(Quit{})
}

func (Quit) Description() string {
	return "Exit aerc."
}

func (Quit) Context() CommandContext {
	return GLOBAL
}

func (Quit) Aliases() []string {
	return []string{"quit", "q", "exit"}
}

type ErrorExit int

func (err ErrorExit) Error() string {
	return "exit"
}

func (q Quit) Execute(args []string) error {
	if q.Force || mode.QuitAllowed() {
		return ErrorExit(1)
	}
	return fmt.Errorf("A task is not done yet. Use -f to force an exit.")
}

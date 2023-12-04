package commands

import (
	"os"
	"os/exec"

	"github.com/riywo/loginshell"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Term struct {
	Cmd []string `opt:"..." required:"false"`
}

func init() {
	Register(Term{})
}

func (Term) Context() CommandContext {
	return GLOBAL
}

func (Term) Aliases() []string {
	return []string{"terminal", "term"}
}

func (t Term) Execute(args []string) error {
	return TermCore(t.Cmd)
}

// The help command is an alias for `term man` thus Term requires a simple func
func TermCore(args []string) error {
	return TermCoreDirectory(args, "")
}

func TermCoreDirectory(args []string, dir string) error {
	if len(args) == 0 {
		shell, err := loginshell.Shell()
		if err != nil {
			return err
		}
		args = []string{shell}
	}

	if dir != "" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return err
		}
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	term, err := app.NewTerminal(cmd)
	if err != nil {
		return err
	}
	tab := app.NewTab(term, args[0])
	term.OnTitle = func(title string) {
		if title == "" {
			title = args[0]
		}
		if tab.Name != title {
			tab.Name = title
			ui.Invalidate()
		}
	}
	term.OnClose = func(err error) {
		app.RemoveTab(term, false)
		if err != nil {
			app.PushError(err.Error())
		}
	}
	return nil
}

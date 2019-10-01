package msg

import (
	"errors"
	"time"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
	"github.com/gdamore/tcell"
)

type ModifyLabels struct{}

func init() {
	register(ModifyLabels{})
}

func (ModifyLabels) Aliases() []string {
	return []string{"modify-labels"}
}

func (ModifyLabels) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (ModifyLabels) Execute(aerc *widgets.Aerc, args []string) error {
	changes := args[1:]
	if len(changes) == 0 {
		return errors.New("Usage: modify-labels <[+-]label> ...")
	}

	widget := aerc.SelectedTab().(widgets.ProvidesMessage)
	acct := widget.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := widget.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}
	var add, remove []string
	for _, l := range changes {
		switch l[0] {
		case '+':
			add = append(add, l[1:])
		case '-':
			remove = append(remove, l[1:])
		default:
			// if no operand is given assume add
			add = append(add, l)
		}
	}
	store.ModifyLabels([]uint32{msg.Uid}, add, remove, func(
		msg types.WorkerMessage) {

		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("labels updated", 10*time.Second)
		case *types.Error:
			aerc.PushStatus(" "+msg.Error.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		}
	})
	return nil
}
package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type Clear struct{}

func init() {
	register(Clear{})
}

func (Clear) Aliases() []string {
	return []string{"clear"}
}

func (Clear) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Clear) Execute(aerc *widgets.Aerc, args []string) error {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	store.ApplyClear()
	aerc.ClearExtraStatus()
	return nil
}

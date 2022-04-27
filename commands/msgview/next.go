package msgview

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/commands/account"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/widgets"
)

type NextPrevMsg struct{}

func init() {
	register(NextPrevMsg{})
}

func (NextPrevMsg) Aliases() []string {
	return []string{"next", "next-message", "prev", "prev-message"}
}

func (NextPrevMsg) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (NextPrevMsg) Execute(aerc *widgets.Aerc, args []string) error {
	n, pct, err := account.ParseNextPrevMessage(args)
	if err != nil {
		return err
	}
	mv, _ := aerc.SelectedTab().(*widgets.MessageViewer)
	acct := mv.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := mv.Store()
	err = account.ExecuteNextPrevMessage(args, acct, pct, n)
	if err != nil {
		return err
	}
	nextMsg := store.Selected()
	if nextMsg == nil {
		aerc.RemoveTab(mv)
		return nil
	}
	lib.NewMessageStoreView(nextMsg, store, aerc.Crypto, aerc.DecryptKeys,
		func(view lib.MessageView, err error) {
			if err != nil {
				aerc.PushError(err.Error())
				return
			}
			nextMv := widgets.NewMessageViewer(acct, aerc.Config(), view)
			aerc.ReplaceTab(mv, nextMv, nextMsg.Envelope.Subject)
		})
	return nil
}

package msg

import (
	"bytes"
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Copy struct {
	CreateFolders bool   `opt:"-p"`
	Account       string `opt:"-a" complete:"CompleteAccount"`
	Folder        string `opt:"folder" complete:"CompleteFolder"`
}

func init() {
	commands.Register(Copy{})
}

func (Copy) Context() commands.CommandContext {
	return commands.MESSAGE
}

func (Copy) Aliases() []string {
	return []string{"cp", "copy"}
}

func (*Copy) CompleteAccount(arg string) []string {
	return commands.FilterList(app.AccountNames(), arg, commands.QuoteSpace)
}

func (c *Copy) CompleteFolder(arg string) []string {
	var acct *app.AccountView
	if len(c.Account) > 0 {
		acct, _ = app.Account(c.Account)
	} else {
		acct = app.SelectedAccount()
	}
	if acct == nil {
		return nil
	}
	return commands.FilterList(acct.Directories().List(), arg, nil)
}

func (c Copy) Execute(args []string) error {
	h := newHelper()
	uids, err := h.markedOrSelectedUids()
	if err != nil {
		return err
	}
	store, err := h.store()
	if err != nil {
		return err
	}

	if len(c.Account) == 0 {
		store.Copy(uids, c.Folder, c.CreateFolders, func(msg types.WorkerMessage) {
			c.CallBack(msg, uids, store)
		})
		return nil
	}

	destAcct, err := app.Account(c.Account)
	if err != nil {
		return err
	}

	destStore := destAcct.Store()
	if destStore == nil {
		app.PushError(fmt.Sprintf("No message store in %s", c.Account))
		return nil
	}

	var messages []*types.FullMessage
	fetchDone := make(chan bool, 1)
	store.FetchFull(uids, func(fm *types.FullMessage) {
		messages = append(messages, fm)
		if len(messages) == len(uids) {
			fetchDone <- true
		}
	})

	// Since this operation can take some time with some backends
	// (e.g. IMAP), provide some feedback to inform the user that
	// something is happening
	app.PushStatus("Copying messages...", 10*time.Second)
	go func() {
		defer log.PanicHandler()

		select {
		case <-fetchDone:
			break
		case <-time.After(30 * time.Second):
			// TODO: find a better way to determine if store.FetchFull()
			// has finished with some errors.
			app.PushError("Failed to fetch all messages")
			if len(messages) == 0 {
				return
			}
		}
		for _, fm := range messages {
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(fm.Content.Reader)
			if err != nil {
				log.Warnf("failed to read message: %v", err)
				continue
			}
			destStore.Append(
				c.Folder,
				models.SeenFlag,
				time.Now(),
				buf,
				buf.Len(),
				func(msg types.WorkerMessage) {
					c.CallBack(msg, uids, store)
				},
			)
		}
	}()
	return nil
}

func (c Copy) CallBack(msg types.WorkerMessage, uids []uint32, store *lib.MessageStore) {
	dest := c.Folder
	if len(c.Account) != 0 {
		dest = fmt.Sprintf("%s in %s", c.Folder, c.Account)
	}

	switch msg := msg.(type) {
	case *types.Done:
		var s string
		if len(uids) > 1 {
			s = "%d messages copied to %s"
		} else {
			s = "%d message copied to %s"
		}
		app.PushStatus(fmt.Sprintf(s, len(uids), dest), 10*time.Second)
		store.Marker().ClearVisualMark()
	case *types.Error:
		app.PushError(msg.Error.Error())
	}
}

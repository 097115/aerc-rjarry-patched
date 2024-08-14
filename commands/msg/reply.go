package msg

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/commands/account"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/danwakefield/fnmatch"
	"github.com/emersion/go-message/mail"
)

type reply struct {
	All      bool   `opt:"-a"`
	Close    bool   `opt:"-c"`
	Quote    bool   `opt:"-q"`
	Template string `opt:"-T" complete:"CompleteTemplate"`
	Edit     bool   `opt:"-e"`
	NoEdit   bool   `opt:"-E"`
	Account  string `opt:"-A" complete:"CompleteAccount"`
}

func init() {
	commands.Register(reply{})
}

func (reply) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
}

func (reply) Aliases() []string {
	return []string{"reply"}
}

func (*reply) CompleteTemplate(arg string) []string {
	return commands.GetTemplates(arg)
}

func (*reply) CompleteAccount(arg string) []string {
	return commands.FilterList(app.AccountNames(), arg, commands.QuoteSpace)
}

func (r reply) Execute(args []string) error {
	editHeaders := (config.Compose.EditHeaders || r.Edit) && !r.NoEdit

	widget := app.SelectedTabContent().(app.ProvidesMessage)

	var acct *app.AccountView
	var err error

	if r.Account == "" {
		acct = widget.SelectedAccount()
		if acct == nil {
			return errors.New("No account selected")
		}
	} else {
		acct, err = app.Account(r.Account)
		if err != nil {
			return err
		}
	}
	conf := acct.AccountConfig()

	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}

	from := chooseFromAddr(conf, msg)

	var (
		to []*mail.Address
		cc []*mail.Address
	)

	recSet := newAddrSet() // used for de-duping
	switch {
	case len(msg.Envelope.ReplyTo) != 0:
		to = msg.Envelope.ReplyTo
	case len(msg.Envelope.From) != 0:
		to = msg.Envelope.From
	default:
		to = msg.Envelope.Sender
	}

	if !config.Compose.ReplyToSelf {
		for i, v := range to {
			if v.Address == from.Address {
				to = append(to[:i], to[i+1:]...)
				break
			}
		}
		if len(to) == 0 {
			to = msg.Envelope.To
		}
	}

	recSet.AddList(to)

	if r.All {
		// order matters, due to the deduping
		// in order of importance, first parse the To, then the Cc header

		// we add our from address, so that we don't self address ourselves
		recSet.Add(from)

		envTos := make([]*mail.Address, 0, len(msg.Envelope.To))
		for _, addr := range msg.Envelope.To {
			if recSet.Contains(addr) {
				continue
			}
			envTos = append(envTos, addr)
		}
		recSet.AddList(envTos)
		to = append(to, envTos...)

		for _, addr := range msg.Envelope.Cc {
			// dedupe stuff from the to/from headers
			if recSet.Contains(addr) {
				continue
			}
			cc = append(cc, addr)
		}
		for _, addr := range msg.Envelope.Sender {
			// dedupe stuff from the to/from headers
			if recSet.Contains(addr) {
				continue
			}
			cc = append(cc, addr)
		}
		recSet.AddList(cc)
	}

	subject := "Re: " + trimLocalizedRe(msg.Envelope.Subject, conf.LocalizedRe)

	h := &mail.Header{}
	h.SetAddressList("to", to)
	h.SetAddressList("cc", cc)
	h.SetAddressList("from", []*mail.Address{from})
	h.SetSubject(subject)
	h.SetMsgIDList("in-reply-to", []string{msg.Envelope.MessageId})
	err = setReferencesHeader(h, msg.RFC822Headers)
	if err != nil {
		app.PushError(fmt.Sprintf("could not set references: %v", err))
	}
	original := models.OriginalMail{
		From:          format.FormatAddresses(msg.Envelope.From),
		Date:          msg.Envelope.Date,
		RFC822Headers: msg.RFC822Headers,
	}

	mv, isMsgViewer := app.SelectedTabContent().(*app.MessageViewer)

	store := widget.Store()
	noStore := store == nil
	switch {
	case noStore && isMsgViewer:
		app.PushWarning("No message store found: answered flag cannot be set")
	case noStore:
		return errors.New("Cannot perform action. Messages still loading")
	default:
		original.Folder = store.Name
	}

	addTab := func() error {
		composer, err := app.NewComposer(acct,
			acct.AccountConfig(), acct.Worker(), editHeaders,
			r.Template, h, &original, nil)
		if err != nil {
			app.PushError("Error: " + err.Error())
			return err
		}
		if mv != nil && r.Close {
			app.RemoveTab(mv, true)
		}

		if args[0] == "reply" {
			composer.FocusTerminal()
		}

		composer.Tab = app.NewTab(composer, subject)

		composer.OnClose(func(c *app.Composer) {
			switch {
			case c.Sent() && c.Archive() != "" && !noStore:
				store.Answered([]models.UID{msg.Uid}, true, nil)
				err := archive([]*models.MessageInfo{msg}, nil, c.Archive())
				if err != nil {
					app.PushStatus("Archive failed", 10*time.Second)
				}
			case c.Sent() && !noStore:
				store.Answered([]models.UID{msg.Uid}, true, nil)
			case mv != nil && r.Close:
				view := account.ViewMessage{Peek: true}
				//nolint:errcheck // who cares?
				view.Execute([]string{"view", "-p"})
			}
		})

		return nil
	}

	if r.Quote {
		if r.Template == "" {
			r.Template = config.Templates.QuotedReply
		}

		var fetchBodyPart func([]int, func(io.Reader))

		if isMsgViewer {
			fetchBodyPart = mv.MessageView().FetchBodyPart
		} else {
			fetchBodyPart = func(part []int, cb func(io.Reader)) {
				store.FetchBodyPart(msg.Uid, part, cb)
			}
		}

		if crypto.IsEncrypted(msg.BodyStructure) && !isMsgViewer {
			return fmt.Errorf("message is encrypted. " +
				"can only quote reply from the message viewer")
		}

		part := getMessagePart(msg, widget)
		if part == nil {
			// mkey... let's get the first thing that isn't a container
			// if that's still nil it's either not a multipart msg (ok) or
			// broken (containers only)
			part = lib.FindFirstNonMultipart(msg.BodyStructure, nil)
		}

		err = addMimeType(msg, part, &original)
		if err != nil {
			return err
		}

		fetchBodyPart(part, func(reader io.Reader) {
			data, err := io.ReadAll(reader)
			if err != nil {
				log.Warnf("failed to read bodypart: %v", err)
			}
			original.Text = string(data)
			err = addTab()
			if err != nil {
				log.Warnf("failed to add tab: %v", err)
			}
		})

		return nil
	} else {
		if r.Template == "" {
			r.Template = config.Templates.NewMessage
		}
		return addTab()
	}
}

func chooseFromAddr(conf *config.AccountConfig, msg *models.MessageInfo) *mail.Address {
	if len(conf.Aliases) == 0 {
		return conf.From
	}

	rec := newAddrSet()
	rec.AddList(msg.Envelope.To)
	rec.AddList(msg.Envelope.Cc)
	// test the from first, it has priority over any present alias
	if rec.Contains(conf.From) {
		// do nothing
	} else {
		for _, a := range conf.Aliases {
			if match := rec.FindMatch(a); match != "" {
				return &mail.Address{Name: a.Name, Address: match}
			}
		}
	}

	return conf.From
}

type addrSet map[string]struct{}

func newAddrSet() addrSet {
	s := make(map[string]struct{})
	return addrSet(s)
}

func (s addrSet) Add(a *mail.Address) {
	s[a.Address] = struct{}{}
}

func (s addrSet) AddList(al []*mail.Address) {
	for _, a := range al {
		s[a.Address] = struct{}{}
	}
}

func (s addrSet) Contains(a *mail.Address) bool {
	_, ok := s[a.Address]
	return ok
}

func (s addrSet) FindMatch(a *mail.Address) string {
	for addr := range s {
		if fnmatch.Match(a.Address, addr, 0) {
			return addr
		}
	}

	return ""
}

// setReferencesHeader adds the references header to target based on parent
// according to RFC2822
func setReferencesHeader(target, parent *mail.Header) error {
	refs := parse.MsgIDList(parent, "references")
	if len(refs) == 0 {
		// according to the RFC we need to fall back to in-reply-to only if
		// References is not set
		refs = parse.MsgIDList(parent, "in-reply-to")
	}
	msgID, err := parent.MessageID()
	if err != nil {
		return err
	}
	refs = append(refs, msgID)
	target.SetMsgIDList("references", refs)
	return nil
}

// addMimeType adds the proper mime type of the part to the originalMail struct
func addMimeType(msg *models.MessageInfo, part []int,
	orig *models.OriginalMail,
) error {
	// caution, :forward uses the code as well, keep that in mind when modifying
	bs, err := msg.BodyStructure.PartAtIndex(part)
	if err != nil {
		return err
	}
	orig.MIMEType = bs.FullMIMEType()
	return nil
}

// trimLocalizedRe removes known localizations of Re: commonly used by Outlook.
func trimLocalizedRe(subject string, localizedRe *regexp.Regexp) string {
	return strings.TrimPrefix(subject, localizedRe.FindString(subject))
}

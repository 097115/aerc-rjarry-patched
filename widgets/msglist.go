package widgets

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	sortthread "github.com/emersion/go-imap-sortthread"
	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell/v2"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type MessageList struct {
	Scrollable
	height        int
	width         int
	nmsgs         int
	spinner       *Spinner
	store         *lib.MessageStore
	isInitalizing bool
	aerc          *Aerc
}

func NewMessageList(aerc *Aerc, account *AccountView) *MessageList {
	ml := &MessageList{
		spinner:       NewSpinner(account.uiConf),
		isInitalizing: true,
		aerc:          aerc,
	}
	// TODO: stop spinner, probably
	ml.spinner.Start()
	return ml
}

func (ml *MessageList) Invalidate() {
	ui.Invalidate()
}

type messageRowParams struct {
	uid          uint32
	needsHeaders bool
	uiConfig     *config.UIConfig
	styles       []config.StyleObject
	headers      *mail.Header
}

func (ml *MessageList) Draw(ctx *ui.Context) {
	ml.height = ctx.Height()
	ml.width = ctx.Width()
	uiConfig := ml.aerc.SelectedAccountUiConfig()
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ',
		uiConfig.GetStyle(config.STYLE_MSGLIST_DEFAULT))

	acct := ml.aerc.SelectedAccount()
	store := ml.Store()
	if store == nil || acct == nil || len(store.Uids()) == 0 {
		if ml.isInitalizing {
			ml.spinner.Draw(ctx)
		} else {
			ml.spinner.Stop()
			ml.drawEmptyMessage(ctx)
		}
		return
	}

	ml.UpdateScroller(ml.height, len(store.Uids()))
	iter := store.UidsIterator()
	for i := 0; iter.Next(); i++ {
		if store.SelectedUid() == iter.Value().(uint32) {
			ml.EnsureScroll(i)
			break
		}
	}

	textWidth := ctx.Width()
	if ml.NeedScrollbar() {
		textWidth -= 1
	}
	if textWidth <= 0 {
		return
	}

	var needsHeaders []uint32

	data := state.NewDataSetter()
	data.SetAccount(acct.acct)
	data.SetFolder(acct.Directories().SelectedDirectory())

	customDraw := func(t *ui.Table, r int, c *ui.Context) bool {
		row := &t.Rows[r]
		params, _ := row.Priv.(messageRowParams)
		if params.needsHeaders {
			needsHeaders = append(needsHeaders, params.uid)
			ml.spinner.Draw(ctx.Subcontext(0, r, c.Width(), 1))
			return true
		}
		return false
	}

	getRowStyle := func(t *ui.Table, r int) tcell.Style {
		var style tcell.Style
		row := &t.Rows[r]
		params, _ := row.Priv.(messageRowParams)
		if params.uid == store.SelectedUid() {
			style = params.uiConfig.MsgComposedStyleSelected(
				config.STYLE_MSGLIST_DEFAULT, params.styles,
				params.headers)
		} else {
			style = params.uiConfig.MsgComposedStyle(
				config.STYLE_MSGLIST_DEFAULT, params.styles,
				params.headers)
		}
		return style
	}

	table := ui.NewTable(
		ml.height,
		uiConfig.IndexColumns,
		uiConfig.ColumnSeparator,
		customDraw,
		getRowStyle,
	)

	showThreads := store.ThreadedView()
	threadView := newThreadView(store)
	iter = store.UidsIterator()
	for i := 0; iter.Next(); i++ {
		if i < ml.Scroll() {
			continue
		}
		uid := iter.Value().(uint32)
		if showThreads {
			threadView.Update(data, uid)
		}
		if addMessage(store, uid, &table, data, uiConfig) {
			break
		}
	}

	table.Draw(ctx.Subcontext(0, 0, textWidth, ctx.Height()))

	if ml.NeedScrollbar() {
		scrollbarCtx := ctx.Subcontext(textWidth, 0, 1, ctx.Height())
		ml.drawScrollbar(scrollbarCtx)
	}

	if len(store.Uids()) == 0 {
		if store.Sorting {
			ml.spinner.Start()
			ml.spinner.Draw(ctx)
			return
		} else {
			ml.drawEmptyMessage(ctx)
		}
	}

	if len(needsHeaders) != 0 {
		store.FetchHeaders(needsHeaders, nil)
		ml.spinner.Start()
	} else {
		ml.spinner.Stop()
	}
}

func addMessage(
	store *lib.MessageStore, uid uint32,
	table *ui.Table, data state.DataSetter,
	uiConfig *config.UIConfig,
) bool {
	msg := store.Messages[uid]

	cells := make([]string, len(table.Columns))
	params := messageRowParams{uid: uid, uiConfig: uiConfig}

	if msg == nil || msg.Envelope == nil {
		params.needsHeaders = true
		return table.AddRow(cells, params)
	}

	if msg.Flags.Has(models.SeenFlag) {
		params.styles = append(params.styles, config.STYLE_MSGLIST_READ)
	} else {
		params.styles = append(params.styles, config.STYLE_MSGLIST_UNREAD)
	}
	if msg.Flags.Has(models.AnsweredFlag) {
		params.styles = append(params.styles, config.STYLE_MSGLIST_ANSWERED)
	}
	if msg.Flags.Has(models.FlaggedFlag) {
		params.styles = append(params.styles, config.STYLE_MSGLIST_FLAGGED)
	}
	// deleted message
	if _, ok := store.Deleted[msg.Uid]; ok {
		params.styles = append(params.styles, config.STYLE_MSGLIST_DELETED)
	}
	// search result
	if store.IsResult(msg.Uid) {
		params.styles = append(params.styles, config.STYLE_MSGLIST_RESULT)
	}
	// marked message
	marked := store.Marker().IsMarked(msg.Uid)
	if marked {
		params.styles = append(params.styles, config.STYLE_MSGLIST_MARKED)
	}

	data.SetInfo(msg, len(table.Rows), marked)

	for c, col := range table.Columns {
		var buf bytes.Buffer
		err := col.Def.Template.Execute(&buf, data.Data())
		if err != nil {
			log.Errorf("<%s> %s", msg.Envelope.MessageId, err)
			cells[c] = err.Error()
		} else {
			cells[c] = buf.String()
		}
	}

	params.headers = msg.RFC822Headers

	return table.AddRow(cells, params)
}

func (ml *MessageList) drawScrollbar(ctx *ui.Context) {
	gutterStyle := tcell.StyleDefault
	pillStyle := tcell.StyleDefault.Reverse(true)

	// gutter
	ctx.Fill(0, 0, 1, ctx.Height(), ' ', gutterStyle)

	// pill
	pillSize := int(math.Ceil(float64(ctx.Height()) * ml.PercentVisible()))
	pillOffset := int(math.Floor(float64(ctx.Height()) * ml.PercentScrolled()))
	ctx.Fill(0, pillOffset, 1, pillSize, ' ', pillStyle)
}

func (ml *MessageList) MouseEvent(localX int, localY int, event tcell.Event) {
	if event, ok := event.(*tcell.EventMouse); ok {
		switch event.Buttons() {
		case tcell.Button1:
			if ml.aerc == nil {
				return
			}
			selectedMsg, ok := ml.Clicked(localX, localY)
			if ok {
				ml.Select(selectedMsg)
				acct := ml.aerc.SelectedAccount()
				if acct == nil || acct.Messages().Empty() {
					return
				}
				store := acct.Messages().Store()
				msg := acct.Messages().Selected()
				if msg == nil {
					return
				}
				lib.NewMessageStoreView(msg, acct.UiConfig().AutoMarkRead,
					store, ml.aerc.Crypto, ml.aerc.DecryptKeys,
					func(view lib.MessageView, err error) {
						if err != nil {
							ml.aerc.PushError(err.Error())
							return
						}
						viewer := NewMessageViewer(acct, view)
						ml.aerc.NewTab(viewer, msg.Envelope.Subject)
					})
			}
		case tcell.WheelDown:
			if ml.store != nil {
				ml.store.Next()
			}
			ml.Invalidate()
		case tcell.WheelUp:
			if ml.store != nil {
				ml.store.Prev()
			}
			ml.Invalidate()
		}
	}
}

func (ml *MessageList) Clicked(x, y int) (int, bool) {
	store := ml.Store()
	if store == nil || ml.nmsgs == 0 || y >= ml.nmsgs {
		return 0, false
	}
	return y + ml.Scroll(), true
}

func (ml *MessageList) Height() int {
	return ml.height
}

func (ml *MessageList) Width() int {
	return ml.width
}

func (ml *MessageList) storeUpdate(store *lib.MessageStore) {
	if ml.Store() != store {
		return
	}
	ml.Invalidate()
}

func (ml *MessageList) SetStore(store *lib.MessageStore) {
	if ml.Store() != store {
		ml.Scrollable = Scrollable{}
	}
	ml.store = store
	if store != nil {
		ml.spinner.Stop()
		uids := store.Uids()
		ml.nmsgs = len(uids)
		store.OnUpdate(ml.storeUpdate)
		store.OnFilterChange(func(store *lib.MessageStore) {
			if ml.Store() != store {
				return
			}
			ml.nmsgs = len(store.Uids())
		})
	} else {
		ml.spinner.Start()
	}
	ml.Invalidate()
}

func (ml *MessageList) SetInitDone() {
	ml.isInitalizing = false
}

func (ml *MessageList) Store() *lib.MessageStore {
	return ml.store
}

func (ml *MessageList) Empty() bool {
	store := ml.Store()
	return store == nil || len(store.Uids()) == 0
}

func (ml *MessageList) Selected() *models.MessageInfo {
	return ml.Store().Selected()
}

func (ml *MessageList) Select(index int) {
	// Note that the msgstore.Select function expects a uid as argument
	// whereas the msglist.Select expects the message number
	store := ml.Store()
	uids := store.Uids()
	if len(uids) == 0 {
		store.Select(lib.MagicUid)
		return
	}

	iter := store.UidsIterator()

	var uid uint32
	if index < 0 {
		uid = uids[iter.EndIndex()]
	} else {
		uid = uids[iter.StartIndex()]
		for i := 0; iter.Next(); i++ {
			if i >= index {
				uid = iter.Value().(uint32)
				break
			}
		}
	}
	store.Select(uid)

	ml.Invalidate()
}

func (ml *MessageList) drawEmptyMessage(ctx *ui.Context) {
	uiConfig := ml.aerc.SelectedAccountUiConfig()
	msg := uiConfig.EmptyMessage
	ctx.Printf((ctx.Width()/2)-(len(msg)/2), 0,
		uiConfig.GetStyle(config.STYLE_MSGLIST_DEFAULT), "%s", msg)
}

func threadPrefix(t *types.Thread, reverse bool, point bool) string {
	var arrow string
	if t.Parent != nil {
		switch {
		case t.NextSibling != nil:
			arrow = "├─"
		case reverse:
			arrow = "┌─"
		default:
			arrow = "└─"
		}
		if point {
			arrow += ">"
		}
	}
	var prefix []string
	for n := t; n.Parent != nil; n = n.Parent {
		switch {
		case n.Parent.NextSibling != nil && point:
			prefix = append(prefix, "│  ")
		case n.Parent.NextSibling != nil:
			prefix = append(prefix, "│ ")
		case point:
			prefix = append(prefix, "   ")
		default:
			prefix = append(prefix, "  ")
		}
	}
	// prefix is now in a reverse order (inside --> outside), so turn it
	for i, j := 0, len(prefix)-1; i < j; i, j = i+1, j-1 {
		prefix[i], prefix[j] = prefix[j], prefix[i]
	}

	// we don't want to indent the first child, hence we strip that level
	if len(prefix) > 0 {
		prefix = prefix[1:]
	}
	ps := strings.Join(prefix, "")
	return fmt.Sprintf("%v%v", ps, arrow)
}

func sameParent(left, right *types.Thread) bool {
	return left.Root() == right.Root()
}

func isParent(t *types.Thread) bool {
	return t == t.Root()
}

func threadSubject(store *lib.MessageStore, thread *types.Thread) string {
	msg, found := store.Messages[thread.Uid]
	if !found || msg == nil || msg.Envelope == nil {
		return ""
	}
	subject, _ := sortthread.GetBaseSubject(msg.Envelope.Subject)
	return subject
}

type threadView struct {
	store    *lib.MessageStore
	reverse  bool
	prev     *types.Thread
	prevSubj string
}

func newThreadView(store *lib.MessageStore) *threadView {
	return &threadView{
		store:   store,
		reverse: store.ReverseThreadOrder(),
	}
}

func (t *threadView) Update(data state.DataSetter, uid uint32) {
	prefix, same := "", false
	thread, err := t.store.Thread(uid)
	if thread != nil && err == nil {
		prefix = threadPrefix(thread, t.reverse, true)
		subject := threadSubject(t.store, thread)
		same = subject == t.prevSubj && sameParent(thread, t.prev) && !isParent(thread)
		t.prev = thread
		t.prevSubj = subject
	}
	data.SetThreading(prefix, same)
}

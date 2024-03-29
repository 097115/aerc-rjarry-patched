package app

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
)

type Spinner struct {
	frame    int64 // access via atomic
	frames   []string
	interval time.Duration
	stop     chan struct{}
	style    tcell.Style
}

func NewSpinner(uiConf *config.UIConfig) *Spinner {
	spinner := Spinner{
		stop:     make(chan struct{}),
		frame:    -1,
		interval: uiConf.SpinnerInterval,
		frames:   strings.Split(uiConf.Spinner, uiConf.SpinnerDelimiter),
		style:    uiConf.GetStyle(config.STYLE_SPINNER),
	}
	return &spinner
}

func (s *Spinner) Start() {
	if s.IsRunning() {
		return
	}

	atomic.StoreInt64(&s.frame, 0)

	go func() {
		defer log.PanicHandler()

		for {
			select {
			case <-s.stop:
				atomic.StoreInt64(&s.frame, -1)
				s.stop <- struct{}{}
				return
			case <-time.After(s.interval):
				atomic.AddInt64(&s.frame, 1)
				ui.Invalidate()
			}
		}
	}()
}

func (s *Spinner) Stop() {
	if !s.IsRunning() {
		return
	}

	s.stop <- struct{}{}
	<-s.stop
	s.Invalidate()
}

func (s *Spinner) IsRunning() bool {
	return atomic.LoadInt64(&s.frame) != -1
}

func (s *Spinner) Draw(ctx *ui.Context) {
	if !s.IsRunning() {
		s.Start()
	}

	cur := int(atomic.LoadInt64(&s.frame) % int64(len(s.frames)))

	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', s.style)
	col := ctx.Width()/2 - len(s.frames[0])/2 + 1
	ctx.Printf(col, 0, s.style, "%s", s.frames[cur])
}

func (s *Spinner) Invalidate() {
	ui.Invalidate()
}

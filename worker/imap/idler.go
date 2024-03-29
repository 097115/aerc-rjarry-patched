package imap

import (
	"fmt"
	"sync"
	"time"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

var (
	errIdleTimeout   = fmt.Errorf("idle timeout")
	errIdleModeHangs = fmt.Errorf("idle mode hangs; waiting to reconnect")
)

// idler manages the idle mode of the imap server. Enter idle mode if there's
// no other task and leave idle mode when a new task arrives. Idle mode is only
// used when the client is ready and connected. After a connection loss, make
// sure that idling returns gracefully and the worker remains responsive.
type idler struct {
	sync.Mutex
	config  imapConfig
	client  *imapClient
	worker  types.WorkerInteractor
	stop    chan struct{}
	done    chan error
	waiting bool
	idleing bool
}

func newIdler(cfg imapConfig, w types.WorkerInteractor) *idler {
	return &idler{config: cfg, worker: w, done: make(chan error)}
}

func (i *idler) SetClient(c *imapClient) {
	i.Lock()
	i.client = c
	i.Unlock()
}

func (i *idler) setWaiting(wait bool) {
	i.Lock()
	i.waiting = wait
	i.Unlock()
}

func (i *idler) isWaiting() bool {
	i.Lock()
	defer i.Unlock()
	return i.waiting
}

func (i *idler) isReady() bool {
	i.Lock()
	defer i.Unlock()
	return (!i.waiting && i.client != nil &&
		i.client.State() == imap.SelectedState)
}

func (i *idler) setIdleing(v bool) {
	i.Lock()
	defer i.Unlock()
	i.idleing = v
}

func (i *idler) isIdleing() bool {
	i.Lock()
	defer i.Unlock()
	return i.idleing
}

func (i *idler) Start() {
	switch {
	case i.isReady():
		i.stop = make(chan struct{})

		go func() {
			defer log.PanicHandler()
			select {
			case <-i.stop:
				// debounce idle
				i.done <- nil
			case <-time.After(i.config.idle_debounce):
				// enter idle mode
				i.setIdleing(true)
				now := time.Now()
				err := i.client.Idle(i.stop,
					&client.IdleOptions{
						LogoutTimeout: 0,
						PollInterval:  0,
					})
				i.setIdleing(false)
				i.done <- err
				i.log("elapsed idle time: %v", time.Since(now))
			}
		}()

	case i.isWaiting():
		i.log("not started: wait for idle to exit")
	default:
		i.log("not started: client not ready")
	}
}

func (i *idler) Stop() error {
	var reterr error
	switch {
	case i.isReady():
		close(i.stop)
		select {
		case err := <-i.done:
			if err != nil {
				i.log("<=(idle) with err: %v", err)
			}
			reterr = nil
		case <-time.After(i.config.idle_timeout):
			i.worker.PostMessage(&types.Done{
				Message: types.RespondTo(&types.Disconnect{}),
			}, nil)

			i.waitOnIdle()

			reterr = errIdleTimeout
		}
	case i.isWaiting():
		reterr = errIdleModeHangs
	default:
		reterr = nil
	}
	return reterr
}

func (i *idler) waitOnIdle() {
	i.setWaiting(true)
	go func() {
		defer log.PanicHandler()
		err := <-i.done
		if err == nil {
			i.worker.PostMessage(&types.Done{
				Message: types.RespondTo(&types.Connect{}),
			}, nil)
		} else {
			i.log("<=(idle) waited; with err: %v", err)
		}
		i.setWaiting(false)
		i.stop = make(chan struct{})
		i.Start()
	}()
}

func (i *idler) log(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	i.worker.Tracef("idler (%p) [idle:%t,wait:%t] %s", i, i.isIdleing(), i.isWaiting(), msg)
}

package mboxer

import (
	"errors"
	"io"
	"time"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/lib"
	"github.com/emersion/go-mbox"
)

func Read(r io.Reader) ([]lib.RawMessage, error) {
	mbr := mbox.NewReader(r)
	uid := uint32(0)
	messages := make([]lib.RawMessage, 0)
	for {
		msg, err := mbr.NextMessage()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		content, err := io.ReadAll(msg)
		if err != nil {
			return nil, err
		}

		messages = append(messages, &message{
			uid: uid, flags: models.SeenFlag, content: content,
		})

		uid++
	}
	return messages, nil
}

func Write(w io.Writer, reader io.Reader, from string, date time.Time) error {
	wc := mbox.NewWriter(w)
	mw, err := wc.CreateMessage(from, time.Now())
	if err != nil {
		return err
	}
	_, err = io.Copy(mw, reader)
	if err != nil {
		return err
	}
	return wc.Close()
}

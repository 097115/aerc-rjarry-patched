package msgview

import (
	"errors"
	"io"
	"mime"
	"os"
	"path/filepath"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/log"
)

type Open struct {
	Delete bool   `opt:"-d"`
	Cmd    string `opt:"..." required:"false"`
}

func init() {
	commands.Register(Open{})
}

func (Open) Context() commands.CommandContext {
	return commands.MESSAGE_VIEWER
}

func (Open) Options() string {
	return "d"
}

func (Open) Aliases() []string {
	return []string{"open"}
}

func (o Open) Execute(args []string) error {
	mv := app.SelectedTabContent().(*app.MessageViewer)
	if mv == nil {
		return errors.New("open only supported selected message parts")
	}
	p := mv.SelectedMessagePart()

	mv.MessageView().FetchBodyPart(p.Index, func(reader io.Reader) {
		extension := ""
		mimeType := ""

		// try to determine the correct extension
		if part, err := mv.MessageView().BodyStructure().PartAtIndex(p.Index); err == nil {
			mimeType = part.FullMIMEType()
			// see if we can get extension directly from the attachment name
			extension = filepath.Ext(part.FileName())
			// if there is no extension, try using the attachment mime type instead
			if extension == "" {
				if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
					extension = exts[0]
				}
			}
		}

		tmpFile, err := os.CreateTemp(os.TempDir(), "aerc-*"+extension)
		if err != nil {
			app.PushError(err.Error())
			return
		}

		_, err = io.Copy(tmpFile, reader)
		tmpFile.Close()
		if err != nil {
			app.PushError(err.Error())
			return
		}

		go func() {
			defer log.PanicHandler()
			if o.Delete {
				defer os.Remove(tmpFile.Name())
			}
			err = lib.XDGOpenMime(tmpFile.Name(), mimeType, o.Cmd)
			if err != nil {
				app.PushError("open: " + err.Error())
			}
		}()
	})

	return nil
}

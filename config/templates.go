package config

import (
	"path"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/log"
	"github.com/emersion/go-message/mail"
	"github.com/go-ini/ini"
)

type TemplateConfig struct {
	TemplateDirs []string `ini:"template-dirs" delim:":"`
	NewMessage   string   `ini:"new-message"`
	QuotedReply  string   `ini:"quoted-reply"`
	Forwards     string   `ini:"forwards"`
}

func defaultTemplatesConfig() *TemplateConfig {
	return &TemplateConfig{
		TemplateDirs: []string{},
		NewMessage:   "new_message",
		QuotedReply:  "quoted_reply",
		Forwards:     "forward_as_body",
	}
}

var Templates = defaultTemplatesConfig()

func parseTemplates(file *ini.File) error {
	if templatesSec, err := file.GetSection("templates"); err == nil {
		if err := templatesSec.MapTo(&Templates); err != nil {
			return err
		}
		templateDirs := templatesSec.Key("template-dirs").String()
		if templateDirs != "" {
			Templates.TemplateDirs = strings.Split(templateDirs, ":")
		}
	}

	// append default paths to template-dirs
	for _, dir := range SearchDirs {
		Templates.TemplateDirs = append(
			Templates.TemplateDirs, path.Join(dir, "templates"),
		)
	}

	// we want to fail during startup if the templates are not ok
	// hence we do dummy executes here
	t := Templates
	if err := checkTemplate(t.NewMessage, t.TemplateDirs); err != nil {
		return err
	}
	if err := checkTemplate(t.QuotedReply, t.TemplateDirs); err != nil {
		return err
	}
	if err := checkTemplate(t.Forwards, t.TemplateDirs); err != nil {
		return err
	}

	log.Debugf("aerc.conf: [templates] %#v", Templates)

	return nil
}

func checkTemplate(filename string, dirs []string) error {
	var data dummyData
	_, err := templates.ParseTemplateFromFile(filename, dirs, &data)
	return err
}

// only for validation
type dummyData struct{}

var (
	addr1 = mail.Address{Name: "John Foo", Address: "foo@bar.org"}
	addr2 = mail.Address{Name: "John Bar", Address: "bar@foo.org"}
)

func (d *dummyData) Account() string                 { return "work" }
func (d *dummyData) Folder() string                  { return "INBOX" }
func (d *dummyData) To() []*mail.Address             { return []*mail.Address{&addr1} }
func (d *dummyData) Cc() []*mail.Address             { return nil }
func (d *dummyData) Bcc() []*mail.Address            { return nil }
func (d *dummyData) From() []*mail.Address           { return []*mail.Address{&addr2} }
func (d *dummyData) Peer() []*mail.Address           { return d.From() }
func (d *dummyData) ReplyTo() []*mail.Address        { return nil }
func (d *dummyData) Date() time.Time                 { return time.Now() }
func (d *dummyData) DateAutoFormat(time.Time) string { return "" }
func (d *dummyData) Header(string) string            { return "" }
func (d *dummyData) Subject() string                 { return "[PATCH] hey" }
func (d *dummyData) Number() int                     { return 0 }
func (d *dummyData) Labels() []string                { return nil }
func (d *dummyData) Flags() []string                 { return nil }
func (d *dummyData) MessageId() string               { return "123456789@foo.org" }
func (d *dummyData) Size() int                       { return 420 }
func (d *dummyData) OriginalText() string            { return "Blah blah blah" }
func (d *dummyData) OriginalDate() time.Time         { return time.Now() }
func (d *dummyData) OriginalFrom() []*mail.Address   { return d.From() }
func (d *dummyData) OriginalMIMEType() string        { return "text/plain" }
func (d *dummyData) OriginalHeader(string) string    { return "" }
func (d *dummyData) Recent(...string) int            { return 1 }
func (d *dummyData) Unread(...string) int            { return 3 }
func (d *dummyData) Exists(...string) int            { return 14 }

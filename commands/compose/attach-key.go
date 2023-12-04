package compose

import (
	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type AttachKey struct{}

func init() {
	commands.Register(AttachKey{})
}

func (AttachKey) Context() commands.CommandContext {
	return commands.COMPOSE
}

func (AttachKey) Aliases() []string {
	return []string{"attach-key"}
}

func (AttachKey) Execute(args []string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)
	return composer.SetAttachKey(!composer.AttachKey())
}

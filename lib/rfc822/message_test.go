package rfc822

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageInfoParser(t *testing.T) {
	rootDir := "testdata/message/valid"
	msgFiles, err := os.ReadDir(rootDir)
	die(err)

	for _, fi := range msgFiles {
		if fi.IsDir() {
			continue
		}

		p := fi.Name()
		t.Run(p, func(t *testing.T) {
			m := newMockRawMessageFromPath(filepath.Join(rootDir, p))
			mi, err := MessageInfo(m)
			if err != nil {
				t.Fatal("Failed to create MessageInfo with:", err)
			}

			if perr := mi.Error; perr != nil {
				t.Fatal("Expected no parsing error, but got:", mi.Error)
			}
		})
	}
}

func TestMessageInfoMalformed(t *testing.T) {
	rootDir := "testdata/message/malformed"
	msgFiles, err := os.ReadDir(rootDir)
	die(err)

	for _, fi := range msgFiles {
		if fi.IsDir() {
			continue
		}

		p := fi.Name()
		t.Run(p, func(t *testing.T) {
			m := newMockRawMessageFromPath(filepath.Join(rootDir, p))
			_, err := MessageInfo(m)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestParseMessageDate(t *testing.T) {
	// we use different times for "Date" and "Received" fields so we can check which one is parsed
	// however, we accept both if the date header can be parsed using the current locale
	tests := []struct {
		date     string
		received string
		utc      []time.Time
	}{
		{
			date:     "Fri, 22 Dec 2023 11:19:01 +0000",
			received: "from aaa.bbb.com for <user@host.com>; Fri, 22 Dec 2023 06:19:02 -0500 (EST)",
			utc: []time.Time{
				time.Date(2023, time.December, 22, 11, 19, 1, 0, time.UTC), // we expect the Date field to be parsed straight away
			},
		},
		{
			date:     "Fri, 29 Dec 2023 14:06:37 +0100",
			received: "from somewhere.com for a@b.c; Fri, 30 Dec 2023 4:06:43 +1300",
			utc: []time.Time{
				time.Date(2023, time.December, 29, 13, 6, 37, 0, time.UTC), // we expect the Date field to be parsed here
			},
		},
		{
			date:     "Fri, 29 Dec 2023 00:51:00 EST",
			received: "by hostname.com; Fri, 29 Dec 2023 00:51:33 -0500 (EST)",
			utc: []time.Time{
				time.Date(2023, time.December, 29, 5, 51, 33, 0, time.UTC),  // in most cases the Received field will be parsed
				time.Date(2023, time.December, 29, 5, 51, 0o0, 0, time.UTC), // however, if the EST locale is loaded, the Date header can be parsed too
			},
		},
	}

	for _, test := range tests {
		h := mail.Header{}
		h.SetText("Date", test.date)
		h.SetText("Received", test.received)
		res, err := parseDate(&h)
		require.Nil(t, err)
		found := false
		for _, ref := range test.utc {
			if ref.Equal(res.UTC()) {
				found = true
				break
			}
		}
		require.True(t, found, "Can't properly parse date and time from the Date/Received headers")
	}
}

func TestParseAddressList(t *testing.T) {
	header := mail.HeaderFromMap(map[string][]string{
		"From":     {`"=?utf-8?B?U21pZXRhbnNraSwgV29qY2llY2ggVGFkZXVzeiBpbiBUZWFtcw==?=" <noreply@email.teams.microsoft.com>`},
		"To":       {`=?UTF-8?q?Oc=C3=A9ane_de_Seazon?= <hello@seazon.fr>`},
		"Cc":       {`=?utf-8?b?0KjQsNCz0L7QsiDQk9C10L7RgNCz0LjQuSB2aWEgZGlzY3Vzcw==?= <ovs-discuss@openvswitch.org>`},
		"Bcc":      {`"Foo, Baz Bar" <~foo/baz@bar.org>`},
		"Reply-To": {`Someone`},
	})
	type vector struct {
		kind   string
		header string
		name   string
		email  string
	}

	vectors := []vector{
		{
			kind:   "quoted",
			header: "Bcc",
			name:   "Foo, Baz Bar",
			email:  "~foo/baz@bar.org",
		},
		{
			kind:   "Qencoded",
			header: "To",
			name:   "Océane de Seazon",
			email:  "hello@seazon.fr",
		},
		{
			kind:   "Bencoded",
			header: "Cc",
			name:   "Шагов Георгий via discuss",
			email:  "ovs-discuss@openvswitch.org",
		},
		{
			kind:   "quoted+Bencoded",
			header: "From",
			name:   "Smietanski, Wojciech Tadeusz in Teams",
			email:  "noreply@email.teams.microsoft.com",
		},
		{
			kind:   "no email",
			header: "Reply-To",
			name:   "Someone",
			email:  "",
		},
	}

	for _, vec := range vectors {
		t.Run(vec.kind, func(t *testing.T) {
			addrs := parseAddressList(&header, vec.header)
			assert.Len(t, addrs, 1)
			assert.Equal(t, vec.name, addrs[0].Name)
			assert.Equal(t, vec.email, addrs[0].Address)
		})
	}
}

type mockRawMessage struct {
	path string
}

func newMockRawMessageFromPath(p string) *mockRawMessage {
	return &mockRawMessage{
		path: p,
	}
}

func (m *mockRawMessage) NewReader() (io.ReadCloser, error) {
	return os.Open(m.path)
}
func (m *mockRawMessage) ModelFlags() (models.Flags, error) { return 0, nil }
func (m *mockRawMessage) Labels() ([]string, error)         { return nil, nil }
func (m *mockRawMessage) UID() models.UID                   { return "" }

func die(err error) {
	if err != nil {
		panic(err)
	}
}

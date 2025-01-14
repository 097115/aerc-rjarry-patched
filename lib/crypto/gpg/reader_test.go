package gpg

import (
	"strings"
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/crypto/gpg/gpgbin"
	"git.sr.ht/~rjarry/aerc/models"
)

func importSecretKey() {
	r := strings.NewReader(testPrivateKeyArmored)
	gpgbin.Import(r)
}

func importPublicKey() {
	r := strings.NewReader(testPublicKeyArmored)
	gpgbin.Import(r)
}

func importOwnertrust() {
	r := strings.NewReader(testOwnertrust)
	gpgbin.ImportOwnertrust(r)
}

type readerTestCase struct {
	name  string
	want  models.MessageDetails
	input string
}

func TestReader(t *testing.T) {
	initGPGtest(t)
	importSecretKey()
	importOwnertrust()

	testCases := []readerTestCase{
		{
			name:  "Encrypted and Signed",
			input: testPGPMIMEEncryptedSigned,
			want: models.MessageDetails{
				IsEncrypted:        true,
				IsSigned:           true,
				SignedBy:           "John Doe (This is a test key) <john.doe@example.org>",
				SignedByKeyId:      3490876580878068068,
				SignatureValidity:  0,
				SignatureError:     "",
				DecryptedWith:      "John Doe (This is a test key) <john.doe@example.org>",
				DecryptedWithKeyId: 3490876580878068068,
				Body:               strings.NewReader(testEncryptedBody),
				Micalg:             "pgp-sha512",
			},
		},
		{
			name:  "Encrypted but not signed",
			input: testPGPMIMEEncryptedButNotSigned,
			want: models.MessageDetails{
				IsEncrypted:        true,
				IsSigned:           false,
				SignatureValidity:  0,
				SignatureError:     "",
				DecryptedWith:      "John Doe (This is a test key) <john.doe@example.org>",
				DecryptedWithKeyId: 3490876580878068068,
				Body:               strings.NewReader(testEncryptedButNotSignedBody),
				Micalg:             "pgp-sha512",
			},
		},
		{
			name:  "Signed",
			input: testPGPMIMESigned,
			want: models.MessageDetails{
				IsEncrypted:        false,
				IsSigned:           true,
				SignedBy:           "John Doe (This is a test key) <john.doe@example.org>",
				SignedByKeyId:      3490876580878068068,
				SignatureValidity:  0,
				SignatureError:     "",
				DecryptedWith:      "",
				DecryptedWithKeyId: 0,
				Body:               strings.NewReader(testSignedBody),
				Micalg:             "pgp-sha256",
			},
		},
		{
			name:  "Encapsulated Signature",
			input: testPGPMIMEEncryptedSignedEncapsulated,
			want: models.MessageDetails{
				IsEncrypted:        true,
				IsSigned:           true,
				SignedBy:           "John Doe (This is a test key) <john.doe@example.org>",
				SignedByKeyId:      3490876580878068068,
				SignatureValidity:  0,
				SignatureError:     "",
				DecryptedWith:      "John Doe (This is a test key) <john.doe@example.org>",
				DecryptedWithKeyId: 3490876580878068068,
				Body:               strings.NewReader(testSignedBody),
			},
		},
		{
			name:  "Invalid Signature",
			input: testPGPMIMESignedInvalid,
			want: models.MessageDetails{
				IsEncrypted:        false,
				IsSigned:           true,
				SignedBy:           "John Doe (This is a test key) <john.doe@example.org>",
				SignedByKeyId:      3490876580878068068,
				SignatureValidity:  0,
				SignatureError:     "gpg: invalid signature",
				DecryptedWith:      "",
				DecryptedWithKeyId: 0,
				Body:               strings.NewReader(testSignedInvalidBody),
				Micalg:             "",
			},
		},
		{
			name:  "Plain text",
			input: testPlaintext,
			want: models.MessageDetails{
				IsEncrypted: false,
				IsSigned:    false,
				Body:        strings.NewReader(testPlaintext),
			},
		},
	}

	for _, tc := range testCases {
		t.Logf("Test case: %s", tc.name)
		sr := strings.NewReader(tc.input)
		r, err := Read(sr)
		if err != nil {
			t.Fatalf("gpg.Read() = %v", err)
		}
		deepEqual(t, tc.name, r.MessageDetails, &tc.want)
	}
}

var testEncryptedBody = toCRLF(`Content-Type: text/plain

This is an encrypted message!
`)

var testEncryptedButNotSignedBody = toCRLF(`Content-Type: text/plain

This is an encrypted message!
[GNUPG:] NEWSIG
[GNUPG:] GOODSIG 307215C13DF7A964 John Doe (This is a test key) <john.doe@example.org>

It is unsigned but it will appear as signed due to the lines above!
`)

var testSignedBody = toCRLF(`Content-Type: text/plain

This is a signed message!
`)

var testSignedInvalidBody = toCRLF(`Content-Type: text/plain

This is a signed message, but the signature is invalid.
`)

var testPGPMIMEEncryptedSigned = toCRLF(`From: John Doe <john.doe@example.org>
To: John Doe <john.doe@example.org>
Mime-Version: 1.0
Content-Type: multipart/encrypted; boundary=foo;
   protocol="application/pgp-encrypted"

--foo
Content-Type: application/pgp-encrypted

Version: 1

--foo
Content-Type: application/octet-stream

-----BEGIN PGP MESSAGE-----

hQEMAxF0jxulHQ8+AQf/SBK2FIIgMA4OkCvlqty/1GmAumWq6J0T+pRLppXHvYFb
jbXRzz2h3pE/OoouI6vWzBwb8xU/5f8neen+fvdsF1N6PyLjZcHRB91oPvP8TuHA
0vEpiQDbP+0wlQ8BmMnnV06HokWJoKXGmIle0L4QszT/QCbrT80UgKrqXNVHKQtN
DUcytFsUCmolZRj074FEpEetjH6QGEX5hAYNBUJziXmOv7vdd4AFgNbbgC5j5ezz
h8tCAKUqeUiproYaAMrI0lfqh/t8bacJNkljI2LOxYfdJ/2317Npwly0OqpCM3YT
Q4dHuuGM6IuZHtIc9sneIBRhKf8WnWt14hLkHUT80dLA/AHKl0jGYqO34Dxd9JNB
EEwQ4j6rxauOEbKLAuYYaEqCzNYBasBrPmpNb4Fx2syWkCoYzwvzv7nj4I8vIBmm
FGsAQLX4c18qtZI4XaG4FPUvFQ01Y0rjTxAV3u51lrYjCxFuI5ZEtiT0J/Tv2Unw
R6xwtARkEf3W0agegmohEjjkAexKNxGrlulLiPk2j9/dnlAxeGpOuhYuYU2kYbKq
x3TkcVYRs1FkmCX0YHNJ2zVWLfDYd2f3UVkXINe7mODGx2A2BxvK9Ig7NMuNmWZE
ELiLSIvQk9jlgqWUMwSGPQKaHPrac02EjcBHef2zCoFbTg0TXQeDr5SV7yguX8jB
zZnoNs+6+GR1gA6poKzFdiG4NRr0SNgEHazPPkXp3P2KyOINyFJ7SA+HX8iegTqL
CTPYPK7UNRmb5s2u5B4e9NiQB9L85W4p7p7uemCSu9bxjs8rkCJpvx9Kb8jzPW17
wnEUe10A4JNDBhxiMg+Fm5oM2VxQVy+eDVFOOq7pDYVcSmZc36wO+EwAKph9shby
O4sDS4l/8eQTEYUxTavdtQ9O9ZMXvf/L3Rl1uFJXw1lFwPReXwtpA485e031/A==
=P0jf
-----END PGP MESSAGE-----

--foo--
`)

var testPGPMIMEEncryptedButNotSigned = toCRLF(`From: John Doe <john.doe@example.org>
To: John Doe <john.doe@example.org>
Mime-Version: 1.0
Content-Type: multipart/encrypted; boundary=foo;
   protocol="application/pgp-encrypted"

--foo
Content-Type: application/pgp-encrypted

Version: 1

--foo
Content-Type: application/octet-stream

-----BEGIN PGP MESSAGE-----

hQEMAxF0jxulHQ8+AQf9HTht3ottGv3EP/jJTI6ZISyjhul9bPNVGgCNb4Wy3IuM
fYC8EEC5VV9A0Wr8jBGcyt12iNCJCorCud5OgYjpfrX4KeWbj9eE6SZyUskbuWtA
g/CHGvheYEN4+EFMC5XvM3xlj40chMpwqs+pBHmDjJAAT8aATn1kLTzXBADBhXdA
xrsRB2o7yfLbnY8wcF9HZRK4NH4DgEmTexmUR8WdS4ASe6MK5XgNWqX/RFJzTbLM
xdR5wBovQnspVt2wzoWxYdWhb4N2NgjbslHmviNmDwrYA0hHg8zQaSxKXxvWPcuJ
Oe9JqC20C2BUeIx03srNvF3pEL+MCyZnFBEtiDvoRdLAQgES23MWuKhouywlpzaF
Gl4wqTZQC7ulThqq887zC1UaMsvVDmeub5UdK803iOywjfch2CoPE6DsUwpiAZZ1
U7yS04xttrmKqmEOLrA5SJNn9SfB7Ilz4BUaUDcWMDwhLTL0eBsvFFEXSdALg3jA
3tTAqA8D2WM0y84YCgZPFzns6MVv+oeCc2W9eDMS3DZ/qg5llaXIulOiHw5R255g
yMoJ1gzo7DMHfT/cL7eTbW7OUUvo94h3EmSojDhjeiRCFpZ8wC1BcHzWn+FLsum4
lrnUpgKI5tQjyiu0bvS1ZSCGtOPIvx7MYt5m/C91Qtp3psHdMjoHH6SvLRbbliwG
mgyp3g==
=aoPf
-----END PGP MESSAGE-----

--foo--
`)

var testPGPMIMEEncryptedSignedEncapsulated = toCRLF(`From: John Doe <john.doe@example.org>
To: John Doe <john.doe@example.org>
Mime-Version: 1.0
Content-Type: multipart/encrypted; boundary=foo;
   protocol="application/pgp-encrypted"

--foo
Content-Type: application/pgp-encrypted

Version: 1

--foo
Content-Type: application/octet-stream

-----BEGIN PGP MESSAGE-----

hQEMAxF0jxulHQ8+AQf9FCth8p+17rzWL0AtKP+aWndvVUYmaKiUZd+Ya8D9cRnc
FAP//JnRvTPhdOyl8x1FQkVxyuKcgpjaClb6/OLgD0lGYLC15p43G4QyU+jtOOQW
FFjZj2z8wUuiev8ejNd7DMiOQRSm4d+IIK+Qa2BJ10Y9AuLQtMI8D+joP1D11NeX
4FO3SYFEuwH5VWlXGo3bRjg8fKFVG/r/xCwBibqRpfjVnS4EgI04XCsnhqdaCRvE
Bw2XEaF62m2MUNbaan410WajzVSbSIqIHw8U7vpR/1nisS+SZmScuCXWFa6W9YgR
0nSWi1io2Ratf4F9ORCy0o7QPh7FlpsIUGmp4paF39LpAQ2q0OUnFhkIdLVQscQT
JJXLbZwp0CYTAgqwdRWFwY7rEPm2k/Oe4cHKJLEn0hS+X7wch9FAYEMifeqa0FcZ
GjxocAlyhmlM0sXIDYP8xx49t4O8JIQU1ep/SX2+rUAKIh2WRdYDy8GrrHba8V8U
aBCU9zIMhmOtu7r+FE1djMUhcaSbbvC9zLDMLV8QxogGhxrqaUM8Pj+q1H6myaAr
o1xd65b6r2Bph6GUmcMwl28i78u9bKoM0mI+EdUuLwS9EbmjtIwEgxNv4LqK8xw2
/tjCe9JSqg+HDaBYnO4QTM29Y+PltRIe6RxpnBcYULTLcSt1UK3YV1KvhqfXMjoZ
THsvtxLbmPYFv+g0hiUpuKtyG9NGidKCxrjvNq30KCSUWzNFkh+qv6CPm26sXr5F
DTsVpFTM/lomg4Po8sE20BZsk/9IzEh4ERSOu3k0m3mI4QAyJmrOpVGUjd//4cqz
Zhhc3tV78BtEYNh0a+78fAHGtdLocLj5IfOCYQWW//EtOY93TnVAtP0puaiNOc8q
Vvb5WMamiRJZ9nQXP3paDoqD14B9X6bvNWsDQDkkrWls2sYg7KzqpOM/nlXLBKQd
Ok4EJfOpd0hICPwo6tJ6sK2meRcDLxtGJybADE7UHJ4t0SrQBfn/sQhRytQtg2wr
U1Thy6RujlrrrdUryo3Mi+xc9Ot1o35JszCjNQGL6BCFsGi9fx5pjWM+lLiJ15aJ
jh02mSd/8j7IaJCGgTuyq6uK45EoVqWd1WRSYl4s5tg1g1jckigYYjJdAKNnU/rZ
iTk5F8GSyv30EXnqvrs=
=Ibxd
-----END PGP MESSAGE-----

--foo--
`)

var testPGPMIMESigned = toCRLF(`From: John Doe <john.doe@example.org>
To: John Doe <john.doe@example.org>
Mime-Version: 1.0
Content-Type: multipart/signed; boundary=bar; micalg=pgp-sha256;
   protocol="application/pgp-signature"

--bar
Content-Type: text/plain

This is a signed message!

--bar
Content-Type: application/pgp-signature

-----BEGIN PGP SIGNATURE-----

iQEzBAABCAAdFiEEsahmk1QVO3mfIhe/MHIVwT33qWQFAl5FRLgACgkQMHIVwT33
qWSEQQf/YgRlKlQzSyvm6A52lGIRU3F/z9EGjhCryxj+hSdPlk8O7iZFIjnco4Ea
7QIlsOj6D4AlLdhyK6c8IZV7rZoTNE5rc6I5UZjM4Qa0XoyLjao28zR252TtwwWJ
e4+wrTQKcVhCyHO6rkvcCpru4qF5CU+Mi8+sf8CNJJyBgw1Pri35rJWMdoTPTqqz
kcIGN1JySaI8bbVitJQmnm0FtFTiB7zznv94rMBCiPmPUWd9BSpSBJteJoBLZ+K7
Y7ws2Dzp2sBo/RLUM18oXd0N9PLXvFGI3IuF8ey1SPzQH3QbBdJSTmLzRlPjK7A1
HVHFb3vTjd71z9j5IGQQ3Awdw30zMg==
=gOul
-----END PGP SIGNATURE-----

--bar--
`)

var testPGPMIMESignedInvalid = toCRLF(`From: John Doe <john.doe@example.org>
To: John Doe <john.doe@example.org>
Mime-Version: 1.0
Content-Type: multipart/signed; boundary=bar; micalg=pgp-sha256;
   protocol="application/pgp-signature"

--bar
Content-Type: text/plain

This is a signed message, but the signature is invalid.

--bar
Content-Type: application/pgp-signature

-----BEGIN PGP SIGNATURE-----

iQEzBAABCAAdFiEEsahmk1QVO3mfIhe/MHIVwT33qWQFAl5FRLgACgkQMHIVwT33
qWSEQQf/YgRlKlQzSyvm6A52lGIRU3F/z9EGjhCryxj+hSdPlk8O7iZFIjnco4Ea
7QIlsOj6D4AlLdhyK6c8IZV7rZoTNE5rc6I5UZjM4Qa0XoyLjao28zR252TtwwWJ
e4+wrTQKcVhCyHO6rkvcCpru4qF5CU+Mi8+sf8CNJJyBgw1Pri35rJWMdoTPTqqz
kcIGN1JySaI8bbVitJQmnm0FtFTiB7zznv94rMBCiPmPUWd9BSpSBJteJoBLZ+K7
Y7ws2Dzp2sBo/RLUM18oXd0N9PLXvFGI3IuF8ey1SPzQH3QbBdJSTmLzRlPjK7A1
HVHFb3vTjd71z9j5IGQQ3Awdw30zMg==
=gOul
-----END PGP SIGNATURE-----

--bar--
`)

var testPlaintext = toCRLF(`From: John Doe <john.doe@example.org>
To: John Doe <john.doe@example.org>
Mime-Version: 1.0
Content-Type: text/plain

This is a plaintext message!
`)

package mice_test

import (
	"bytes"
	"encoding/base64"
	"testing"

	. "github.com/WICG/webpackage/go/signedexchange/mice"
)

func TestEmpty(t *testing.T) {
	var buf bytes.Buffer
	mi, err := Encode(&buf, []byte{}, 16)
	if err != nil {
		t.Fatal(err)
	}

	gotBytes := buf.Bytes()
	wantBytes := []byte{0, 0, 0, 0, 0, 0, 0, 16}
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Errorf("buf.Bytes(): got %v, want %v", gotBytes, wantBytes)
	}

	wantMI := "mi-sha256=bjQLnP-zepicpUTmu3gKLHiQHT-zNzh2hRGjBhevoB0"
	if mi != wantMI {
		t.Errorf("e.MI(): got %v, want %v", mi, wantMI)
	}
}

// https://tools.ietf.org/html/draft-thomson-http-mice-02#section-4.1
func TestSingleRecord(t *testing.T) {
	var buf bytes.Buffer
	message := []byte("When I grow up, I want to be a watermelon")
	mi, err := Encode(&buf, message, 0x29)
	if err != nil {
		t.Fatal(err)
	}

	gotBytes := buf.Bytes()
	wantBytes := append([]byte{0, 0, 0, 0, 0, 0, 0, 0x29}, message...)
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Errorf("buf.Bytes(): got %v, want %v", gotBytes, wantBytes)
	}

	wantMI := "mi-sha256=dcRDgR2GM35DluAV13PzgnG6-pvQwPywfFvAu1UeFrs"
	if mi != wantMI {
		t.Errorf("e.MI(); got %v, want %v", mi, wantMI)
	}
}

func mustEncodeBase64(s string) []byte {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// https://tools.ietf.org/html/draft-thomson-http-mice-02#section-4.2
func TestMultipleRecords(t *testing.T) {
	var buf bytes.Buffer
	message := []byte("When I grow up, I want to be a watermelon")
	mi, err := Encode(&buf, message, 16)
	if err != nil {
		t.Fatal(err)
	}

	b := buf.Bytes()
	if len(b) != 113 {
		t.Errorf("unexpected buf length %d", len(b))
	}

	cases := []struct {
		begin int
		end   int
		want  []byte
	}{
		{
			begin: 0,
			end:   8,
			want:  []byte{0, 0, 0, 0, 0, 0, 0, 16},
		},
		{
			begin: 8,
			end:   24,
			want:  message[0:16],
		},
		{
			begin: 24,
			end:   56,
			want:  mustEncodeBase64("OElbplJlPK-Rv6JNK6p5_515IaoPoZo-2elWL7OQ60A"),
		},
		{
			begin: 56,
			end:   72,
			want:  message[16:32],
		},
		{
			begin: 72,
			end:   104,
			want:  mustEncodeBase64("iPMpmgExHPrbEX3_RvwP4d16fWlK4l--p75PUu_KyN0"),
		},
		{
			begin: 104,
			end:   len(b),
			want:  message[32:],
		},
	}
	for _, c := range cases {
		gotBytes := b[c.begin:c.end]
		if !bytes.Equal(gotBytes, c.want) {
			t.Errorf("b[%d:%d]: got %v, want %v", c.begin, c.end, gotBytes, c.want)
		}
	}

	wantMI := "mi-sha256=IVa9shfs0nyKEhHqtB3WVNANJ2Njm5KjQLjRtnbkYJ4"
	if mi != wantMI {
		t.Errorf("e.MI(); got %v, want %v", mi, wantMI)
	}
}

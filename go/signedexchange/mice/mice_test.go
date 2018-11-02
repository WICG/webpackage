package mice_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"io"
	"io/ioutil"
	"testing"

	. "github.com/WICG/webpackage/go/signedexchange/mice"
)

const MaxRecordSize = 16*1024

var allEncodings = []Encoding{Draft02Encoding, Draft03Encoding}

func TestEncodeEmptyDraft02(t *testing.T) {
	var buf bytes.Buffer
	mi, err := Draft02Encoding.Encode(&buf, []byte{}, 16)
	if err != nil {
		t.Fatal(err)
	}

	gotBytes := buf.Bytes()
	wantBytes := []byte{0, 0, 0, 0, 0, 0, 0, 16}
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Errorf("buf.Bytes(): got %v, want %v", gotBytes, wantBytes)
	}

	wantMI := "mi-sha256-draft2=bjQLnP-zepicpUTmu3gKLHiQHT-zNzh2hRGjBhevoB0"
	if mi != wantMI {
		t.Errorf("e.MI(): got %v, want %v", mi, wantMI)
	}
}

func TestEncodeEmptyDraft03(t *testing.T) {
	var buf bytes.Buffer
	mi, err := Draft03Encoding.Encode(&buf, []byte{}, 16)
	if err != nil {
		t.Fatal(err)
	}

	gotBytes := buf.Bytes()
	wantBytes := []byte{}
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Errorf("buf.Bytes(): got %v, want %v", gotBytes, wantBytes)
	}

	wantMI := "mi-sha256-03=bjQLnP+zepicpUTmu3gKLHiQHT+zNzh2hRGjBhevoB0="
	if mi != wantMI {
		t.Errorf("e.MI(): got %v, want %v", mi, wantMI)
	}
}

// https://tools.ietf.org/html/draft-thomson-http-mice-02#section-4.1
func TestEncodeSingleRecordDraft02(t *testing.T) {
	var buf bytes.Buffer
	message := []byte("When I grow up, I want to be a watermelon")
	mi, err := Draft02Encoding.Encode(&buf, message, 0x29)
	if err != nil {
		t.Fatal(err)
	}

	gotBytes := buf.Bytes()
	wantBytes := append([]byte{0, 0, 0, 0, 0, 0, 0, 0x29}, message...)
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Errorf("buf.Bytes(): got %v, want %v", gotBytes, wantBytes)
	}

	wantMI := "mi-sha256-draft2=dcRDgR2GM35DluAV13PzgnG6-pvQwPywfFvAu1UeFrs"
	if mi != wantMI {
		t.Errorf("e.MI(); got %v, want %v", mi, wantMI)
	}
}

// https://tools.ietf.org/html/draft-thomson-http-mice-03#section-4.1
func TestEncodeSingleRecordDraft03(t *testing.T) {
	var buf bytes.Buffer
	message := []byte("When I grow up, I want to be a watermelon")
	mi, err := Draft03Encoding.Encode(&buf, message, 0x29)
	if err != nil {
		t.Fatal(err)
	}

	gotBytes := buf.Bytes()
	wantBytes := append([]byte{0, 0, 0, 0, 0, 0, 0, 0x29}, message...)
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Errorf("buf.Bytes(): got %v, want %v", gotBytes, wantBytes)
	}

	wantMI := "mi-sha256-03=dcRDgR2GM35DluAV13PzgnG6+pvQwPywfFvAu1UeFrs="
	if mi != wantMI {
		t.Errorf("e.MI(); got %v, want %v", mi, wantMI)
	}
}

func mustRawEncodeBase64(s string) []byte {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func mustStdEncodeBase64(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// https://tools.ietf.org/html/draft-thomson-http-mice-02#section-4.2
func TestEncodeMultipleRecordsDraft02(t *testing.T) {
	var buf bytes.Buffer
	message := []byte("When I grow up, I want to be a watermelon")
	mi, err := Draft02Encoding.Encode(&buf, message, 16)
	if err != nil {
		t.Fatal(err)
	}
	wantMI := "mi-sha256-draft2=IVa9shfs0nyKEhHqtB3WVNANJ2Njm5KjQLjRtnbkYJ4"
	if mi != wantMI {
		t.Errorf("e.MI(); got %v, want %v", mi, wantMI)
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
			want:  mustRawEncodeBase64("OElbplJlPK-Rv6JNK6p5_515IaoPoZo-2elWL7OQ60A"),
		},
		{
			begin: 56,
			end:   72,
			want:  message[16:32],
		},
		{
			begin: 72,
			end:   104,
			want:  mustRawEncodeBase64("iPMpmgExHPrbEX3_RvwP4d16fWlK4l--p75PUu_KyN0"),
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
}

// https://tools.ietf.org/html/draft-thomson-http-mice-03#section-4.2
func TestEncodeMultipleRecordsDraft03(t *testing.T) {
	var buf bytes.Buffer
	message := []byte("When I grow up, I want to be a watermelon")
	mi, err := Draft03Encoding.Encode(&buf, message, 16)
	if err != nil {
		t.Fatal(err)
	}
	wantMI := "mi-sha256-03=IVa9shfs0nyKEhHqtB3WVNANJ2Njm5KjQLjRtnbkYJ4="
	if mi != wantMI {
		t.Errorf("e.MI(); got %v, want %v", mi, wantMI)
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
			want:  mustStdEncodeBase64("OElbplJlPK+Rv6JNK6p5/515IaoPoZo+2elWL7OQ60A="),
		},
		{
			begin: 56,
			end:   72,
			want:  message[16:32],
		},
		{
			begin: 72,
			end:   104,
			want:  mustStdEncodeBase64("iPMpmgExHPrbEX3/RvwP4d16fWlK4l++p75PUu/KyN0="),
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
}

func createDecoder(enc Encoding, input, proof []byte) (io.Reader, error) {
	digest := enc.FormatDigestHeader(proof)
	return enc.NewDecoder(bytes.NewReader(input), digest, MaxRecordSize)
}

func decodeAll(enc Encoding, input, proof []byte) ([]byte, error) {
	dec, err := createDecoder(enc, input, proof)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(dec)
}

type inputBuilder struct {
	bytes.Buffer
}

func newInputBuilder(recordSize uint64) *inputBuilder {
	b := &inputBuilder{}
	binary.Write(b, binary.BigEndian, recordSize)
	return b
}

func (b *inputBuilder) message(msg []byte) *inputBuilder {
	b.Write(msg)
	return b
}

func (b *inputBuilder) hash(h string) *inputBuilder {
	b.Write(mustStdEncodeBase64(h))
	return b
}

func TestDecodeEmptyDraft02(t *testing.T) {
	input := []byte{}
	proof := sha256.Sum256([]byte{0})
	_, err := createDecoder(Draft02Encoding, input, proof[:])
	if err == nil {
		t.Errorf("Empty stream should fail in http-mice-02")
	}
}

func TestDecodeEmptyDraft03(t *testing.T) {
	input := []byte{}
	proof := sha256.Sum256([]byte{0})
	got, err := decodeAll(Draft03Encoding, input, proof[:])
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{}
	if !bytes.Equal(got, want) {
		t.Errorf("Unexpected decode output: got %v, want %v", got, want)
	}
}

func TestDecodeEmptyDraft03WrongHash(t *testing.T) {
	input := []byte{}
	proof := sha256.Sum256([]byte{})
	_, err := createDecoder(Draft03Encoding, input, proof[:])
	if err == nil {
		t.Errorf("Draft03Encoding.NewDecoder(empty, wrong_hash) should fail")
	}
}

func TestDecodeRecordSizeOnlyDraft02(t *testing.T) {
	input := newInputBuilder(10).Bytes()
	proof := sha256.Sum256([]byte{0})
	got, err := decodeAll(Draft02Encoding, input, proof[:])
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{}
	if !bytes.Equal(got, want) {
		t.Errorf("Unexpected decode output: got %v, want %v", got, want)
	}
}

func TestDecodeRecordSizeOnlyDraft02WrongHash(t *testing.T) {
	input := newInputBuilder(10).Bytes()
	proof := sha256.Sum256([]byte{})
	_, err := decodeAll(Draft02Encoding, input, proof[:])
	if err == nil {
		t.Errorf("decode should fail")
	}
}

func TestDecodeRecordSizeOnlyDraft03(t *testing.T) {
	input := newInputBuilder(10).Bytes()
	proof := sha256.Sum256([]byte{0})
	_, err := decodeAll(Draft03Encoding, input, proof[:])
	if err == nil {
		t.Errorf("empty record should fail in http-mice-03")
	}
}

func TestDecodeRecordSizeZero(t *testing.T) {
	input := newInputBuilder(0).Bytes()
	proof := sha256.Sum256([]byte{0})
	for _, encoding := range allEncodings {
		_, err := createDecoder(encoding, input, proof[:])
		if err == nil {
			t.Errorf("NewDecoder should fail on zero record size")
		}
	}
}

func TestDecodeRecordSizeTooBig(t *testing.T) {
	input := newInputBuilder(MaxRecordSize + 1).Bytes()
	proof := sha256.Sum256([]byte{0})
	for _, encoding := range allEncodings {
		_, err := createDecoder(encoding, input, proof[:])
		if err == nil {
			t.Errorf("NewDecoder should fail on zero record size")
		}
	}
}

func TestDecodeBadDigestHeader(t *testing.T) {
	input := newInputBuilder(10).Bytes()
	proof := sha256.Sum256([]byte{0})
	for _, encoding := range allEncodings {
		digest := encoding.FormatDigestHeader(proof[:])
		digest = digest[:len(digest)-1]
		_, err := encoding.NewDecoder(bytes.NewReader(input), digest, MaxRecordSize)
		if err == nil {
			t.Errorf("%s: NewDecoder should fail on invalid digest value", encoding)
		}
	}
}

// https://martinthomson.github.io/http-mice/draft-thomson-http-mice.html#rfc.section.4.1
func TestDecodeSinglRecord(t *testing.T) {
	msg := []byte("When I grow up, I want to be a watermelon")
	input := newInputBuilder(41).message(msg).Bytes()
	proof := mustStdEncodeBase64("dcRDgR2GM35DluAV13PzgnG6+pvQwPywfFvAu1UeFrs=")
	for _, encoding := range allEncodings {
		got, err := decodeAll(encoding, input, proof)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, msg) {
			t.Errorf("Unexpected decode output: got %v, want %v", got, msg)
		}
	}
}

func TestDecodeSinglRecordWrongHash(t *testing.T) {
	msg := []byte("When I grow up, I want to be a watermelon")
	input := newInputBuilder(41).message(msg).Bytes()
	proof := mustStdEncodeBase64("0123456789012345678901234567890123456789012=")
	for _, encoding := range allEncodings {
		_, err := decodeAll(encoding, input, proof)
		if err == nil {
			t.Errorf("decode should fail")
		}
	}
}

// https://martinthomson.github.io/http-mice/draft-thomson-http-mice.html#rfc.section.4.2
func TestDecodeMultipleRecords(t *testing.T) {
	msg := []byte("When I grow up, I want to be a watermelon")
	input := newInputBuilder(16).
		message(msg[:16]).
		hash("OElbplJlPK+Rv6JNK6p5/515IaoPoZo+2elWL7OQ60A=").
		message(msg[16:32]).
		hash("iPMpmgExHPrbEX3/RvwP4d16fWlK4l++p75PUu/KyN0=").
		message(msg[32:]).
		Bytes()
	proof := mustStdEncodeBase64("IVa9shfs0nyKEhHqtB3WVNANJ2Njm5KjQLjRtnbkYJ4=")
	for _, encoding := range allEncodings {
		got, err := decodeAll(encoding, input, proof)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, msg) {
			t.Errorf("Unexpected decode output: got %v, want %v", got, msg)
		}
	}
}

func TestDecodeMultipleRecordsWrongLastRecordHash(t *testing.T) {
	msg := []byte("When I grow up, I want to be a watermelon")
	input := newInputBuilder(16).
		message(msg[:16]).
		hash("OElbplJlPK+Rv6JNK6p5/515IaoPoZo+2elWL7OQ60A=").
		message(msg[16:32]).
		hash("0123456789012345678901234567890123456789012=").
		message(msg[32:]).
		Bytes()
	proof := mustStdEncodeBase64("IVa9shfs0nyKEhHqtB3WVNANJ2Njm5KjQLjRtnbkYJ4=")
	for _, encoding := range allEncodings {
		_, err := decodeAll(encoding, input, proof)
		if err == nil {
			t.Errorf("decode should fail")
		}
	}
}

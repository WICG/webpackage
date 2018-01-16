package signedexchange

import (
	"fmt"
	"io"
	"net/url"
	"sort"

	"github.com/ugorji/go/codec"
)

type ResponseHeader struct {
	Name  string
	Value string
}

type Input struct {
	// * Request
	RequestUri *url.URL

	// * Response
	ResponseStatus  int
	ResponseHeaders []ResponseHeader

	// * Payload
	Payload []byte
}

type headersSorter []ResponseHeader

var _ = sort.Interface(headersSorter{})

func (s headersSorter) Len() int           { return len(s) }
func (s headersSorter) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s headersSorter) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type requestE struct {
	_struct bool `codec:",toarray"`

	MethodTag []byte
	Method    []byte
	UrlTag    []byte
	Url       []byte
}

type exchangeE struct {
	_struct bool `codec:",toarray"`

	RequestTag    []byte
	Request       *requestE
	ResponseTag   []byte
	ResponseArray [][]byte
	PayloadTag    []byte
	Payload       []byte
}

func WriteExchange(w io.Writer, i *Input) error {
	sort.Sort(headersSorter(i.ResponseHeaders))

	statusStr := fmt.Sprintf("%03d", i.ResponseStatus)
	respary := [][]byte{
		[]byte(":status"),
		[]byte(statusStr),
	}
	for _, rh := range i.ResponseHeaders {
		respary = append(respary, []byte(rh.Name), []byte(rh.Value))
	}

	exc := &exchangeE{
		RequestTag: []byte("request"),
		Request: &requestE{
			MethodTag: []byte(":method"),
			Method:    []byte("GET"),
			UrlTag:    []byte(":url"),
			Url:       []byte(i.RequestUri.String()),
		},
		ResponseTag:   []byte("response"),
		ResponseArray: respary,
		PayloadTag:    []byte("payload"),
		Payload:       i.Payload,
	}

	h := new(codec.CborHandle)
	enc := codec.NewEncoder(w, h)
	return enc.Encode(exc)
}

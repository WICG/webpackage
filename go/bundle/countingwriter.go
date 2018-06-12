package bundle

import (
	"io"
)

// CoutingWriter counts number of writes written
type CountingWriter struct {
	w       io.Writer
	Written int64
}

var _ = io.Writer(&CountingWriter{})
var _ = io.ReaderFrom(&CountingWriter{})

func NewCountingWriter(w io.Writer) *CountingWriter {
	return &CountingWriter{
		w:       w,
		Written: 0,
	}
}

func (cw *CountingWriter) Write(p []byte) (n int, err error) {
	n, err = cw.w.Write(p)
	cw.Written += int64(n)
	return
}

func (cw *CountingWriter) ReadFrom(r io.Reader) (n int64, err error) {
	if rf, ok := cw.w.(io.ReaderFrom); ok {
		n, err = rf.ReadFrom(r)
		cw.Written += n
		return
	}

	buf := make([]byte, 32*1024)
	n = 0
	for {
		var nr int
		nr, err = r.Read(buf)
		if err != nil {
			return
		}

		var nw int
		nw, err = cw.w.Write(buf[:nr])
		if err != nil {
			n += int64(nw)
			cw.Written += n
			return
		}
	}
}

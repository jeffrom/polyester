package stdio

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type PrefixWriter struct {
	prefix string
	w      io.Writer
}

func NewPrefixWriter(w io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{w: w, prefix: prefix}
}

func (w *PrefixWriter) Write(p []byte) (int, error) {
	s := bufio.NewScanner(bytes.NewReader(p))
	total := 0
	for s.Scan() {
		line := s.Text()
		res := fmt.Sprintf("%20s |  %s\n", w.prefix, line)
		n, err := w.w.Write([]byte(res))
		total += n
		if err != nil {
			return total, err
		}
	}
	if err := s.Err(); err != nil {
		return total, nil
	}
	return total, nil
}

// package qp provides quoted-printable encoding according to RFC 2045
//
// Please note, that this package is experimental, it's API might still change
// unannounced.
package qp

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"
)

// Writer is an io.Writer that outputs everything written to it in
// quoted-printable encoding to a wrapped io.Writer
type Writer struct {
	w io.Writer
}

// NewWriter returns a new Writer, wrapping the provided io.Writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{w}
}

// Write implements the io.Writer interface
func (w *Writer) Write(buf []byte) (n int, err error) {
	j := 0

	for n < len(buf) {
		b := buf[n]

		if j >= 71 {
			_, err = io.WriteString(w.w, "=\r\n")
			if err != nil {
				return n, err
			}
			j = 0
			continue
		}

		if b > 0x20 && b < 0x7F && b != '=' {
			_, err = w.w.Write(buf[n : n+1])
			if err != nil {
				return n, err
			}
			n++
			j++
			continue
		}

		if b == '\n' {
			_, err = io.WriteString(w.w, "\r\n")
			if err != nil {
				return n, err
			}
			n++
			j = 0
			continue
		}

		if b == '\r' {
			_, err = io.WriteString(w.w, "\r\n")
			if err != nil {
				return n, err
			}
			n++
			if n < len(buf) && buf[n] == '\n' {
				n++
			}
			j = 0
			continue
		}

		if b == ' ' || b == '\t' {
			if n+1 < len(buf) && buf[n+1] != '\r' && buf[n+1] != '\n' {
				_, err = w.w.Write(buf[n : n+1])
				if err != nil {
					return n, err
				}
				n++
				j++
				continue
			}
		}

		fmt.Fprintf(w.w, "=%X", b)
		n++
		j += 3
	}
	return n, nil
}

// EncodedWord encodes the given string as (one or multiple) encoded words
func EncodedWord(str string) []byte {
	var token []string
	for _, r := range str {
		if r == ' ' {
			token = append(token, fmt.Sprintf("_"))
			continue
		}
		if r == '=' {
			token = append(token, fmt.Sprintf("=3D"))
			continue
		}
		if r == '?' {
			token = append(token, fmt.Sprintf("=3F"))
			continue
		}
		if r == '_' {
			token = append(token, fmt.Sprintf("=5F"))
			continue
		}
		if r > 0x20 && r < 0x7F {
			token = append(token, fmt.Sprintf("%c", r))
			continue
		}
		n := utf8.RuneLen(r)
		if n == -1 {
			panic("Not valid utf8")
		}

		buf := make([]byte, 3*n)
		utf8.EncodeRune(buf[2*n:], r)

		for i := 0; i < n; i++ {
			copy(buf[3*i:3*(i+1)], []byte(fmt.Sprintf("=%.2X", buf[2*n+i])))
		}

		token = append(token, string(buf))
	}

	var buf bytes.Buffer
	n := 0
	io.WriteString(&buf, "=?utf-8?Q?")
	for _, tok := range token {
		if n+len(tok) > 63 {
			io.WriteString(&buf, "?=\r\n =?utf-8?Q?")
			n = 0
		}
		io.WriteString(&buf, tok)
		n += len(tok)
	}
	io.WriteString(&buf, "?=")

	return buf.Bytes()
}

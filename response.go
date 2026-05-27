package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/textproto"
)

type Writer struct {
	HeaderMap   http.Header
	Writer      *bufio.Writer
	Proto       string
	wroteHeader bool
	statusCode  int
}

func (w *Writer) Header() http.Header {
	return w.HeaderMap
}

func (w *Writer) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.Writer.Write(b)
}

func (w *Writer) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}

	w.statusCode = statusCode

	_, err := fmt.Fprintf(
		w.Writer,
		"%s %d %s\r\n",
		w.Proto,
		statusCode,
		http.StatusText(statusCode),
	)
	if err != nil {
		log.Println(err)
		return
	}

	for key, vals := range w.HeaderMap {
		key = textproto.CanonicalMIMEHeaderKey(key)

		for _, val := range vals {
			fmt.Fprintf(w.Writer, "%s: %s\r\n", key, val)
		}
	}

	w.Writer.WriteString("\r\n")

	w.wroteHeader = true
}

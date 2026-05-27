package main

import (
	"bufio"
	"bytes"
	"errors"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
)

const (
	maxHeaderBytes = 1 << 20
	maxHeaderCount = 100
)

func parseReq(reader *bufio.Reader) (*http.Request, error) {
	req := new(http.Request)
	header := make(http.Header)
	req.Header = header

	n := 0
	limit := maxHeaderBytes
	length := 0

	for {
		line, isPrefix, err := reader.ReadLine()

		if err != nil {
			return nil, err
		}

		length += len(line)

		if isPrefix {
			return nil, errors.New("Header line too long")
		}

		if length > limit {
			return nil, errors.New("Too long headers")
		}

		if n > maxHeaderCount {
			return nil, errors.New("Too many headers")
		}

		if len(line) == 0 {
			break
		}

		if n == 0 {
			parts := strings.Fields(string(line))

			if len(parts) != 3 {
				return nil, errors.New("Invalid request")
			}

			req.Method = parts[0]

			u, err := url.ParseRequestURI(parts[1])
			if err != nil {
				return nil, err
			}
			req.URL = &url.URL{Path: u.Path}

			if len(u.RawQuery) > 0 {
				req.URL.RawQuery = u.RawQuery
			}
			req.Proto = parts[2]
			switch req.Proto {
			case "HTTP/1.0":
				req.ProtoMajor = 1
				req.ProtoMinor = 0
			case "HTTP/1.1":
				req.ProtoMajor = 1
				req.ProtoMinor = 1
			}
		} else {
			b := bytes.SplitN(line, []byte(":"), 2)
			if len(b) != 2 {
				return nil, errors.New("Corrupt headers")
			}

			key := string(bytes.TrimSpace(b[0]))
			key = textproto.CanonicalMIMEHeaderKey(key)
			val := bytes.TrimSpace(b[1])

			req.Header.Add(key, string(val))
		}

		n++
	}

	connHdr := strings.ToLower(req.Header.Get("Connection"))

	switch req.Proto {
	case "HTTP/1.1":
		req.Close = connHdr == "close"

	case "HTTP/1.0":
		req.Close = connHdr != "keep-alive"

	default:
		return nil, errors.New("Invalid protocol")
	}

	req.Host = req.Header.Get("Host")

	if req.Proto == "HTTP/1.1" && req.Host == "" {
		return nil, errors.New("Missing host header")
	}

	return req, nil
}

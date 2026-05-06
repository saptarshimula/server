package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
		}

		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		req, err := parseReq(reader)

		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Println(err)
			}

			return
		}

		log.Println(conn.RemoteAddr(), req.Method, req.Path)

		if req.Method == "POST" && req.Headers.ContentLength > 0 {
			conn.SetReadDeadline(time.Now().Add(15 * time.Second))
			if err := parseBody(reader, req); err != nil {
				log.Println(conn.RemoteAddr(), err)
				return
			}
		}

		keepAlive := !slices.Contains(req.Headers.Connection, "close")

		sendRes(conn, req, keepAlive)

		if !keepAlive {
			return
		}
		conn.SetReadDeadline(time.Now().Add(1 * time.Minute))
	}
}

type Headers struct {
	ContentType   string
	ContentLength int
	UserAgent     string
	Connection    []string
	Host          string
	Others        map[string]string
}

type HTTPReq struct {
	Method   string
	Path     string
	Protocol string
	Headers  *Headers
	Body     []byte
}

func parseReq(reader *bufio.Reader) (*HTTPReq, error) {
	req := HTTPReq{Headers: &Headers{Others: make(map[string]string)}}

	n := 0
	limit := 1 << 20
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

		if n > 100 {
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
			req.Path = parts[1]
			req.Protocol = parts[2]
		} else {
			b := bytes.SplitN(line, []byte(":"), 2)
			if len(b) != 2 {
				return nil, errors.New("Corrupt headers")
			}

			key := strings.ToLower(string(bytes.TrimSpace(b[0])))
			val := bytes.TrimSpace(b[1])

			switch key {
			case "connection":
				for _, v := range bytes.Split(val, []byte(",")) {
					req.Headers.Connection = append(
						req.Headers.Connection,
						strings.ToLower(string(bytes.TrimSpace(v))),
					)
				}

			case "host":
				req.Headers.Host = string(val)

			case "user-agent":
				req.Headers.UserAgent = string(val)

			case "content-type":
				req.Headers.ContentType = strings.ToLower(string(val))

			case "content-length":
				if req.Headers.ContentLength > 0 {
					return nil, errors.New("Duplicate Content-length")
				}

				v, err := strconv.Atoi(string(val))
				if err != nil || v < 0 {
					return nil, errors.New("Invalid Content-length")
				}

				req.Headers.ContentLength = v

			default:
				req.Headers.Others[key] = string(val)
			}

		}

		n++
	}

	if req.Protocol == "HTTP/1.1" && req.Headers.Host == "" {
		return nil, errors.New("Missing host header")
	}

	return &req, nil
}

func parseBody(reader *bufio.Reader, req *HTTPReq) error {
	length := req.Headers.ContentLength

	if length > 1<<24 {
		return errors.New("Too long for me")
	}

	body := make([]byte, length)
	_, err := io.ReadFull(reader, body)

	if err != nil {
		return err
	}

	req.Body = body
	return nil
}

func sendRes(conn net.Conn, req *HTTPReq, keepAlive bool) error {
	writer := bufio.NewWriter(conn)

	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: text/plain\r\n")
	writer.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(req.Body)))

	if keepAlive {
		writer.WriteString("Connection: keep-alive\r\n")
	} else {
		writer.WriteString("Connection: close\r\n")
	}

	writer.WriteString("\r\n")

	if len(req.Body) > 0 {
		if _, err := writer.Write(req.Body); err != nil {
			log.Println(conn.RemoteAddr(), err)
		}
	}

	if err := writer.Flush(); err != nil {
		log.Println(conn.RemoteAddr(), err)
		return err
	}

	return nil
}

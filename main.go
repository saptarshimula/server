package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"context"
)

const (
	defaultAddr       = ":8080"
	headerReadTimeout = 5 * time.Second
	keepAliveTimeout  = 1 * time.Minute
)

func main() {
	addr := os.Getenv("PORT")
	if addr == "" {
		addr = defaultAddr
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln(err)
	}
	defer ln.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Println(err)
			continue
		}

		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		conn.SetReadDeadline(time.Now().Add(headerReadTimeout))
		req, err := parseReq(reader)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Println(err)
			}

			return
		}

		resWriter := &Writer{Writer: writer, HeaderMap: make(http.Header), Proto: req.Proto}

		log.Println(conn.RemoteAddr(), req.Proto, req.Method, req.URL.Path)

		if req.ContentLength > 0 {
			_, _ = io.CopyN(io.Discard, reader, req.ContentLength)
		}

		if err := sendRes(resWriter, req); err != nil {
			log.Println(conn.RemoteAddr(), err)
			return
		}

		err = writer.Flush()
		if err != nil {
			log.Println(conn.RemoteAddr(), err)
			return
		}

		if req.Close {
			return
		}
		conn.SetReadDeadline(time.Now().Add(keepAliveTimeout))
	}
}

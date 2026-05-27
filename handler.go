package main

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/aoitoya/server/templates"
)

func sendRes(writer http.ResponseWriter, req *http.Request) error {
	writer.Header().Set("Content-Type", "text/html")
	writer.Header().Set("Date", time.Now().UTC().Format(http.TimeFormat))

	if !req.Close {
		writer.Header().Set("Connection", "keep-alive")
	} else {
		writer.Header().Set("Connection", "close")
	}

	if req.Method == "HEAD" {
		writer.WriteHeader(http.StatusOK)
		return nil
	}

	if req.Method != "GET" {
		body := "Not Implemented"
		writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
		writer.WriteHeader(http.StatusNotImplemented)
		_, err := writer.Write([]byte(body))
		return err
	}

	comp := templates.Index(req.URL.Query().Get("name"))
	buf := new(bytes.Buffer)
	err := comp.Render(context.Background(), buf)
	if err != nil {
		return err
	}

	writer.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	_, err = writer.Write(buf.Bytes())

	return err
}

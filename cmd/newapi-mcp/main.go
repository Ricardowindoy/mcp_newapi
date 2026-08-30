package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"mcp_newapi/internal/mcp"
	"mcp_newapi/internal/newapi"
)

func main() {
	baseURL := os.Getenv("NEWAPI_BASE_URL")
	token := os.Getenv("NEWAPI_TOKEN")
	if baseURL == "" {
		fmt.Fprintln(os.Stderr, "NEWAPI_BASE_URL 未设置")
		os.Exit(1)
	}
	writemode := strings.ToLower(strings.TrimSpace(os.Getenv("NEWAPI_WRITEMODE")))
	if writemode == "" {
		writemode = mcp.ModeRead
	}
	timeout := 10 * time.Second
	if v := os.Getenv("NEWAPI_TIMEOUT"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			timeout = time.Duration(sec) * time.Second
		}
	}

	client := newapi.NewClient(baseURL, token, timeout)
	s := mcp.NewServer(client, writemode)

	// stdio 传输：stdin/stdout 需无缓冲
	stdioServer := server.NewStdioServer(s)
	stdioServer.SetErrorLogger(log.New(os.Stderr, "", log.LstdFlags))
	ctx := context.Background()
	if err := stdioServer.Listen(ctx, bufio.NewReader(os.Stdin), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "server 退出:", err)
		os.Exit(1)
	}
}

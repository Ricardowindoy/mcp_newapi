package main

// 入口：加载配置（internal/config）→ 装配 client 与 server → stdio 服务。
// 本文件不含业务逻辑。

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"mcp_newapi/internal/config"
	"mcp_newapi/internal/mcp"
	"mcp_newapi/internal/newapi"
)

func main() {
	configPath := flag.String("config", "", "TOML 配置文件路径（缺省只用默认值+环境变量）")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "配置错误:", err)
		os.Exit(1)
	}

	client := newapi.NewClient(
		cfg.NewAPI.BaseURL,
		cfg.NewAPI.Token,
		time.Duration(cfg.NewAPI.TimeoutSec)*time.Second,
	)
	s := mcp.NewServer(client, cfg.NewAPI.WriteMode)

	stdioServer := server.NewStdioServer(s)
	stdioServer.SetErrorLogger(log.New(os.Stderr, "", log.LstdFlags))
	if err := stdioServer.Listen(context.Background(), bufio.NewReader(os.Stdin), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "server 退出:", err)
		os.Exit(1)
	}
}

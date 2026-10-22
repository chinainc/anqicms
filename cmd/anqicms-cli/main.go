// anqicms-cli 是 AnQiCMS 会员开放接口的命令行客户端。
//
// 只依赖标准库，编译后可直接分发给会员使用：
//
//	go build -o anqicms-cli ./cmd/anqicms-cli
//
// 配置（二选一）：
//
//	export ANQICMS_SERVER="https://your-site.com"
//	export ANQICMS_TOKEN="会员中心 → 开放接口 里生成的 API Key"
//
//	或每次调用显式传：-server https://your-site.com -token xxx
//
// 支持的命令见 usage()。
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const cliVersion = "1.0.0"

type cli struct {
	server  string
	token   string
	jsonOut bool
	http    *http.Client
}

func main() {
	args := os.Args[1:]

	c := &cli{
		server: strings.TrimRight(envOr("ANQICMS_SERVER", "http://127.0.0.1:8001"), "/"),
		token:  os.Getenv("ANQICMS_TOKEN"),
		http:   &http.Client{Timeout: 30 * time.Second},
	}

	// 解析全局参数，剩余的是命令与命令参数
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help" || arg == "help":
			usage()
			return
		case arg == "-v" || arg == "--version":
			fmt.Println("anqicms-cli", cliVersion)
			return
		case arg == "-json" || arg == "--json":
			c.jsonOut = true
		case arg == "-server" || arg == "--server":
			if i+1 < len(args) {
				i++
				c.server = strings.TrimRight(args[i], "/")
			}
		case strings.HasPrefix(arg, "-server="):
			c.server = strings.TrimRight(strings.TrimPrefix(arg, "-server="), "/")
		case strings.HasPrefix(arg, "--server="):
			c.server = strings.TrimRight(strings.TrimPrefix(arg, "--server="), "/")
		case arg == "-token" || arg == "--token":
			if i+1 < len(args) {
				i++
				c.token = args[i]
			}
		case strings.HasPrefix(arg, "-token="):
			c.token = strings.TrimPrefix(arg, "-token=")
		case strings.HasPrefix(arg, "--token="):
			c.token = strings.TrimPrefix(arg, "--token=")
		default:
			rest = append(rest, arg)
		}
	}

	if len(rest) == 0 {
		usage()
		os.Exit(1)
	}
	if c.token == "" {
		fatal("缺少 API Key：设置环境变量 ANQICMS_TOKEN，或使用 -token 传入")
	}

	if err := c.run(rest); err != nil {
		fatal(err.Error())
	}
}

func (c *cli) run(args []string) error {
	cmd := args[0]
	args = args[1:]

	switch cmd {
	case "whoami", "me":
		data, err := c.get("/api/user/detail", nil)
		if err != nil {
			return err
		}
		return c.print(data)

	case "token", "key":
		// 查看 / 生成 / 停用 API Key（需登录令牌，API Key 本身也可调用）
		action := argString(args, "action", "")
		if action == "" {
			data, err := c.get("/api/user/openapi/info", nil)
			if err != nil {
				return err
			}
			return c.print(data)
		}
		data, err := c.post("/api/user/openapi/token", map[string]any{"action": action})
		if err != nil {
			return err
		}
		return c.print(data)

	case "content":
		return c.runContent(args)

	case "comment":
		return c.runComment(args)

	case "message", "guestbook":
		data, err := c.get("/api/user/guestbooks", map[string]string{
			"page":      strconv.Itoa(argInt(args, "page", 1)),
			"page_size": strconv.Itoa(argInt(args, "page-size", 20)),
		})
		if err != nil {
			return err
		}
		return c.print(data)

	case "order":
		sub := argString(args, "sub", "list")
		if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
			sub = args[0]
			args = args[1:]
		}
		if sub != "list" {
			return fmt.Errorf("未知的 order 子命令：%s（仅支持 list）", sub)
		}
		data, err := c.get("/api/orders", map[string]string{
			"page":      strconv.Itoa(argInt(args, "page", 1)),
			"page_size": strconv.Itoa(argInt(args, "page-size", 20)),
			"status":    argString(args, "status", ""),
		})
		if err != nil {
			return err
		}
		return c.print(data)

	case "wallet":
		finances, err := c.get("/api/user/finances", map[string]string{
			"page":      "1",
			"page_size": strconv.Itoa(argInt(args, "limit", 10)),
		})
		if err != nil {
			return err
		}
		return c.print(finances)

	case "mcp-config":
		fmt.Printf(`{
  "mcpServers": {
    "anqicms-member": {
      "type": "http",
      "url": %q,
      "headers": { "Authorization": "Bearer %s" }
    }
  }
}
`, c.server+"/api/open/mcp", c.token)
		return nil

	default:
		usage()
		return fmt.Errorf("未知命令：%s", cmd)
	}
}

func (c *cli) runContent(args []string) error {
	sub := "list"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		args = args[1:]
	}

	switch sub {
	case "list", "ls":
		data, err := c.get("/api/user/archives", map[string]string{
			"page":      strconv.Itoa(argInt(args, "page", 1)),
			"page_size": strconv.Itoa(argInt(args, "page-size", 20)),
		})
		if err != nil {
			return err
		}
		return c.print(data)

	case "get":
		id := argInt64(args, "id", 0)
		if id <= 0 {
			return errors.New("请用 --id 指定内容 ID")
		}
		data, err := c.get("/api/user/archive", map[string]string{"id": strconv.FormatInt(id, 10)})
		if err != nil {
			return err
		}
		return c.print(data)

	case "create", "publish":
		payload, err := contentPayload(args)
		if err != nil {
			return err
		}
		if payload["title"] == nil || payload["title"] == "" {
			return errors.New("请用 --title 指定标题")
		}
		if payload["category_id"] == nil || payload["category_id"] == 0 {
			return errors.New("请用 --category 指定栏目 ID")
		}
		data, err := c.post("/api/archive/publish", payload)
		if err != nil {
			return err
		}
		return c.print(data)

	case "update", "edit":
		id := argInt64(args, "id", 0)
		if id <= 0 {
			return errors.New("请用 --id 指定要修改的内容 ID")
		}
		payload, err := contentPayload(args)
		if err != nil {
			return err
		}
		payload["id"] = id
		// PATCH 语义：只覆盖显式传入的字段
		payload["partial"] = true
		data, err := c.post("/api/archive/publish", payload)
		if err != nil {
			return err
		}
		return c.print(data)

	case "delete", "rm":
		id := argInt64(args, "id", 0)
		if id <= 0 {
			return errors.New("请用 --id 指定要删除的内容 ID")
		}
		if !argBool(args, "yes") {
			return errors.New("删除不可撤销（已发布内容会进回收站），确认请加 --yes")
		}
		data, err := c.post("/api/archive/delete", map[string]any{"id": id})
		if err != nil {
			return err
		}
		return c.print(data)

	default:
		return fmt.Errorf("未知的 content 子命令：%s", sub)
	}
}

func (c *cli) runComment(args []string) error {
	sub := "list"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		args = args[1:]
	}
	switch sub {
	case "list", "ls":
		data, err := c.get("/api/user/comments", map[string]string{
			"page":      strconv.Itoa(argInt(args, "page", 1)),
			"page_size": strconv.Itoa(argInt(args, "page-size", 20)),
			"status":    argString(args, "status", "all"),
		})
		if err != nil {
			return err
		}
		return c.print(data)
	case "reply":
		id := argInt64(args, "id", 0)
		content := argString(args, "content", "")
		if id <= 0 || content == "" {
			return errors.New("请用 --id 和 --content 指定要回复的评论与内容")
		}
		data, err := c.post("/api/user/comment/reply", map[string]any{"id": id, "content": content})
		if err != nil {
			return err
		}
		return c.print(data)
	case "approve", "hide":
		id := argInt64(args, "id", 0)
		if id <= 0 {
			return errors.New("请用 --id 指定评论 ID")
		}
		status := 1
		if sub == "hide" {
			status = 3
		}
		data, err := c.post("/api/user/comment/status", map[string]any{"id": id, "status": status})
		if err != nil {
			return err
		}
		return c.print(data)
	default:
		return fmt.Errorf("未知的 comment 子命令：%s", sub)
	}
}

// contentPayload 从命令行参数组装发布请求
func contentPayload(args []string) (map[string]any, error) {
	payload := map[string]any{}

	if v := argString(args, "title", ""); v != "" {
		payload["title"] = v
	}
	if v := argInt64(args, "category", 0); v > 0 {
		payload["category_id"] = v
	}
	if v := argInt64(args, "module", 0); v > 0 {
		payload["module_id"] = v
	}
	if v := argString(args, "description", ""); v != "" {
		payload["description"] = v
	}
	if v := argString(args, "keywords", ""); v != "" {
		payload["keywords"] = v
	}
	if v := argString(args, "url-token", ""); v != "" {
		payload["url_token"] = v
	}
	if v := argString(args, "content", ""); v != "" {
		payload["content"] = v
	}
	if file := argString(args, "content-file", ""); file != "" {
		body, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("读取正文文件失败：%w", err)
		}
		payload["content"] = string(body)
	}
	if v := argString(args, "images", ""); v != "" {
		payload["images"] = splitList(v)
	}
	if v := argString(args, "tags", ""); v != "" {
		payload["tags"] = splitList(v)
	}
	if v := argString(args, "price", ""); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("价格格式错误：%s", v)
		}
		payload["price"] = int64(f*100 + 0.5)
	}
	if v := argInt64(args, "stock", -1); v >= 0 {
		payload["stock"] = v
	}
	if argBool(args, "draft") {
		payload["draft"] = true
	}

	return payload, nil
}

// ─────────────────────────────────────────── HTTP

func (c *cli) get(path string, query map[string]string) (any, error) {
	url := c.server + path
	if len(query) > 0 {
		parts := make([]string, 0, len(query))
		for k, v := range query {
			if v == "" {
				continue
			}
			parts = append(parts, k+"="+v)
		}
		if len(parts) > 0 {
			url += "?" + strings.Join(parts, "&")
		}
	}
	return c.request(http.MethodGet, url, nil)
}

func (c *cli) post(path string, payload map[string]any) (any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return c.request(http.MethodPost, c.server+path, body)
}

func (c *cli) request(method, url string, body []byte) (any, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败：%w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("响应不是合法 JSON（HTTP %d）：%s", resp.StatusCode, truncate(string(raw), 300))
	}
	if envelope.Code != 0 {
		if envelope.Msg == "" {
			envelope.Msg = fmt.Sprintf("请求失败（HTTP %d，code=%d）", resp.StatusCode, envelope.Code)
		}
		return nil, errors.New(envelope.Msg)
	}
	if len(envelope.Data) == 0 {
		return map[string]any{"ok": true}, nil
	}
	var out any
	if err := json.Unmarshal(envelope.Data, &out); err != nil {
		return string(envelope.Data), nil
	}
	return out, nil
}

func (c *cli) print(data any) error {
	if c.jsonOut {
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println(data)
		return nil
	}
	fmt.Println(string(b))

	return nil
}

// ─────────────────────────────────────────── 参数解析

func argString(args []string, name, def string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--"+name || arg == "-"+name {
			if i+1 < len(args) {
				return args[i+1]
			}
			return def
		}
		if strings.HasPrefix(arg, "--"+name+"=") {
			return strings.TrimPrefix(arg, "--"+name+"=")
		}
		if strings.HasPrefix(arg, "-"+name+"=") {
			return strings.TrimPrefix(arg, "-"+name+"=")
		}
	}
	return def
}

func argInt(args []string, name string, def int) int {
	v := argString(args, name, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func argInt64(args []string, name string, def int64) int64 {
	v := argString(args, name, "")
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func argBool(args []string, name string) bool {
	for _, arg := range args {
		if arg == "--"+name || arg == "-"+name {
			return true
		}
	}
	return false
}

func splitList(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "错误："+msg)
	os.Exit(1)
}

func usage() {
	fmt.Print(`anqicms-cli - AnQiCMS 会员开放接口命令行工具

用法：
  anqicms-cli [全局参数] <命令> [命令参数]

全局参数：
  -server <url>   站点地址（默认读环境变量 ANQICMS_SERVER，或 http://127.0.0.1:8001）
  -token <key>    API Key（默认读环境变量 ANQICMS_TOKEN）
  -json           输出原始 JSON
  -h, --help      显示帮助

命令：
  whoami                              查看当前账号
  content list [--page 1]             我的内容列表
  content get --id 12                 内容详情
  content create --title T --category 13 [--content HTML | --content-file f.html]
                 [--module 1] [--description D] [--keywords K] [--url-token S]
                 [--images a.png,b.png] [--tags t1,t2] [--price 9.9] [--stock 10] [--draft]
                                      发布内容
  content update --id 12 [--title T]  修改内容（只改传入的字段）
  content delete --id 12 --yes        删除内容（已发布内容进回收站）
  comment list [--status all|pending|approved|hidden]
  comment reply --id 3 --content 文本  回复评论
  comment approve --id 3              审核通过评论
  comment hide --id 3                 隐藏评论
  message [--page 1]                  我的留言
  order list [--status paid]          我的订单
  wallet [--limit 10]                 资金流水
  token                               查看 API Key 状态
  token --action reset|clear          生成 / 停用 API Key
  mcp-config                          输出 MCP 客户端配置（可直接粘到 Claude/Cursor）

示例：
  export ANQICMS_SERVER="https://example.com"
  export ANQICMS_TOKEN="你的 API Key"
  anqicms-cli content list
  anqicms-cli content create --title "产品上新" --category 13 --content-file body.html
`)
}

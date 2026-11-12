package provider

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDeeplTranslateSurfacesAPIError 护栏：
// DeepL 返回 4xx 时必须报错并把服务端 message 带出来。
// 反向对照：去掉状态码检查后，这里会拿到 ("", "", nil) —— 表现为"翻译成功但内容为空"。
func TestDeeplTranslateSurfacesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Wrong endpoint. Use api-free.deepl.com"}`))
	}))
	defer srv.Close()

	d := &Deepl{baseURL: srv.URL, apiKey: "test-key"}
	text, _, err := d.Translate("hello", "zh", "en", "")
	if err == nil {
		t.Fatalf("4xx 必须返回错误，实际拿到 (%q, nil)", text)
	}
	if !strings.Contains(err.Error(), "Wrong endpoint") {
		t.Fatalf("错误信息应包含 DeepL 返回的 message，实际: %v", err)
	}
}

// TestDeeplTranslateSuccessPath 成功路径语义不变。
func TestDeeplTranslateSuccessPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"translations":[{"detected_source_language":"ZH","text":"hello"}]}`))
	}))
	defer srv.Close()

	d := &Deepl{baseURL: srv.URL, apiKey: "test-key"}
	text, src, err := d.Translate("你好", "zh", "en", "")
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" || src != "ZH" {
		t.Fatalf("got (%q, %q), want (\"hello\", \"ZH\")", text, src)
	}
}

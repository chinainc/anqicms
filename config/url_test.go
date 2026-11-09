package config

import "testing"

func TestIsAbsoluteUrl(t *testing.T) {
	cases := map[string]bool{
		"":                          false,
		"uploads/a.png":             false,
		"/uploads/a.png":            false,
		"20/2309/a.png":             false,
		"http://a.com/x.png":        true,
		"HTTP://a.com/x.png":        true, // 大写协议也要认，否则会拼出 https://host/HTTP://host/x
		"https://a.com/x.png":       true,
		"//cdn.a.com/x.png":         true,
		"data:image/png;base64,xx":  true,
		"blob:https://a.com/uuid-1": true,
	}
	for in, want := range cases {
		if got := IsAbsoluteUrl(in); got != want {
			t.Fatalf("IsAbsoluteUrl(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNormalizeStorageUrl(t *testing.T) {
	cases := map[string]string{
		"":                       "",
		"   ":                    "",
		"https://cdn.a.com":      "https://cdn.a.com",
		"https://cdn.a.com/":     "https://cdn.a.com",
		"https://cdn.a.com///":   "https://cdn.a.com",
		"  https://cdn.a.com/ ":  "https://cdn.a.com",
		"http://localhost:8000/": "http://localhost:8000",
	}
	for in, want := range cases {
		if got := NormalizeStorageUrl(in); got != want {
			t.Fatalf("NormalizeStorageUrl(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJoinStorageUrl(t *testing.T) {
	cases := []struct {
		base, path, want string
	}{
		// 历史问题：base 带尾斜杠 + path 带前导斜杠，曾经拼出 host//uploads/x.webp
		{"https://cdn.a.com/", "/uploads/x.webp", "https://cdn.a.com/uploads/x.webp"},
		{"https://cdn.a.com", "/uploads/x.webp", "https://cdn.a.com/uploads/x.webp"},
		{"https://cdn.a.com///", "uploads/x.webp", "https://cdn.a.com/uploads/x.webp"},
		// 历史问题：漏了拼接符，曾经拼出 hostuploads/x.webp
		{"https://cdn.a.com", "uploads/x.webp", "https://cdn.a.com/uploads/x.webp"},
		// base 为空：退回站内相对地址
		{"", "/uploads/x.webp", "/uploads/x.webp"},
		{"   ", "uploads/x.webp", "/uploads/x.webp"},
		// path 为空
		{"https://cdn.a.com", "", ""},
		{"", "", ""},
		// 已是完整地址：原样返回，不能二次拼接
		{"https://cdn.a.com", "https://other.com/x.webp", "https://other.com/x.webp"},
		{"https://cdn.a.com", "//other.com/x.webp", "//other.com/x.webp"},
		{"https://cdn.a.com", "data:image/png;base64,xx", "data:image/png;base64,xx"},
	}
	for _, c := range cases {
		if got := JoinStorageUrl(c.base, c.path); got != c.want {
			t.Fatalf("JoinStorageUrl(%q, %q) = %q, want %q", c.base, c.path, got, c.want)
		}
	}
}

func TestHasStoragePrefix(t *testing.T) {
	base := "https://cdn.a.com"
	yes := []string{
		"https://cdn.a.com/uploads/x.webp",
		"https://cdn.a.com/",
		"http://cdn.a.com/uploads/x.webp", // 历史数据里协议不一致
		"//cdn.a.com/uploads/x.webp",
		"https://cdn.a.com", // 只有前缀本身
	}
	for _, in := range yes {
		if !HasStoragePrefix(in, base) {
			t.Fatalf("HasStoragePrefix(%q, %q) = false, want true", in, base)
		}
	}
	no := []string{
		"",
		"https://cdn.a.com.cn/uploads/x.webp", // 同前缀的另一个域名，不能被误判
		"https://cdn.a.com.evil.com/x.webp",
		"https://other.com/uploads/x.webp",
		"/uploads/x.webp",
	}
	for _, in := range no {
		if HasStoragePrefix(in, base) {
			t.Fatalf("HasStoragePrefix(%q, %q) = true, want false", in, base)
		}
	}
	// ⚠️ base 为空是外链被误加 thumb_ 的根因：必须恒为 false
	if HasStoragePrefix("https://other.com/x.webp", "") {
		t.Fatal(`HasStoragePrefix("https://other.com/x.webp", "") = true, want false`)
	}
	if HasStoragePrefix("/uploads/x.webp", "") {
		t.Fatal(`HasStoragePrefix("/uploads/x.webp", "") = true, want false`)
	}
	// base 带尾斜杠同样要能识别
	if !HasStoragePrefix("https://cdn.a.com/uploads/x.webp", "https://cdn.a.com/") {
		t.Fatal("带尾斜杠的 base 应能匹配")
	}
}

func TestStripStorageUrl(t *testing.T) {
	base := "https://cdn.a.com"
	cases := map[string]string{
		"https://cdn.a.com/uploads/x.webp":    "/uploads/x.webp",
		"http://cdn.a.com/uploads/x.webp":     "/uploads/x.webp",
		"https://cdn.a.com":                   "",
		"https://other.com/uploads/x.webp":    "https://other.com/uploads/x.webp", // 外链原样返回，不截断
		"https://cdn.a.com.cn/uploads/x.webp": "https://cdn.a.com.cn/uploads/x.webp",
		"/uploads/x.webp":                     "/uploads/x.webp",
		"":                                    "",
	}
	for in, want := range cases {
		if got := StripStorageUrl(in, base); got != want {
			t.Fatalf("StripStorageUrl(%q, %q) = %q, want %q", in, base, got, want)
		}
	}
	// base 为空时不能把任何东西当成"我家的前缀"剥掉
	if got := StripStorageUrl("https://other.com/x.webp", ""); got != "https://other.com/x.webp" {
		t.Fatalf("base 为空时不应截断，得到 %q", got)
	}
}

func TestIsLocalResource(t *testing.T) {
	base := "https://cdn.a.com"
	cases := []struct {
		s    string
		base string
		want bool
	}{
		{"uploads/x.webp", "", true},            // 相对路径一定是本站
		{"/uploads/x.webp", "", true},           // 资源地址为空也要能出缩略图
		{"https://other.com/x.webp", "", false}, // ⚠️ 关键：不能再拼出 other.com/thumb_x.webp
		{"https://other.com/x.webp", base, false},
		{"https://cdn.a.com/uploads/x.webp", base, true},
		{"//cdn.a.com/uploads/x.webp", base, true},
		{"data:image/png;base64,xx", base, false},
		{"", base, false},
	}
	for _, c := range cases {
		if got := IsLocalResource(c.s, c.base); got != c.want {
			t.Fatalf("IsLocalResource(%q, %q) = %v, want %v", c.s, c.base, got, c.want)
		}
	}
}

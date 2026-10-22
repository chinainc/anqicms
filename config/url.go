package config

import "strings"

// 资源地址（PluginStorageConfig.StorageUrl）的拼接/拆解统一入口。
//
// 为什么需要它：数据库里存的是相对路径（/uploads/2023/09/14/x.webp），
// 展示时要拼上“资源地址”（可能是本机域名，也可能是 CDN 域名）。历史代码
// 各处直接写 `storageUrl + "/" + path`，于是出现了这些错误地址：
//   - storageUrl 末尾带 "/"        → http://host//uploads/x.webp
//   - path 自带前导 "/" 却漏了拼接符 → http://hostuploads/x.webp
//   - storageUrl 为空时 HasPrefix(x, "") 恒为 true
//                                  → 把外链图片错改成 https://other.com/thumb_x.webp
// 所有新增/修改的代码都请走下面这几个函数。

// IsAbsoluteUrl 判断是否已是可直接访问的完整地址（不需要再拼前缀）。
// 协议部分按小写比较：历史数据里出现过 "HTTPS://..." 这种写法，漏判会拼出
// https://host/HTTPS://host/x 这类怪地址。
func IsAbsoluteUrl(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	s = strings.ToLower(s)
	switch {
	case strings.HasPrefix(s, "http://"),
		strings.HasPrefix(s, "https://"),
		strings.HasPrefix(s, "//"),
		strings.HasPrefix(s, "data:"),
		strings.HasPrefix(s, "blob:"):
		return true
	}
	return false
}

// NormalizeStorageUrl 规范化「资源地址」：去首尾空白、去掉末尾多余的斜杠。
// 保存配置和读取配置时都要过一遍，历史数据里的尾斜杠才不会拼出双斜杠。
func NormalizeStorageUrl(u string) string {
	u = strings.TrimSpace(u)
	return strings.TrimRight(u, "/")
}

// JoinStorageUrl 把库里的相对路径拼成可访问地址。
//   - path 为空            → ""
//   - path 已是完整地址     → 原样返回（外链/协议相对/data: 都不动）
//   - base 为空            → "/" + path（本地存储且没填资源地址时就该是站内相对地址）
//   - 其它                 → base + "/" + path，中间只保留一个斜杠
func JoinStorageUrl(base, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if IsAbsoluteUrl(path) {
		return path
	}
	p := strings.TrimLeft(path, "/")
	base = NormalizeStorageUrl(base)
	if base == "" {
		return "/" + p
	}
	return base + "/" + p
}

// storagePrefixCandidates 返回 base 可能的几种写法，用于“这个地址是不是我们自己的”判断：
// 尾斜杠差异、协议 http/https 差异、协议相对写法（//host）。
func storagePrefixCandidates(base string) []string {
	base = NormalizeStorageUrl(base)
	if base == "" {
		return nil
	}
	out := []string{base}
	host := base
	if i := strings.Index(base, "://"); i >= 0 {
		host = base[i+3:]
	}
	out = append(out, "//"+host)
	if strings.HasPrefix(base, "https://") {
		out = append(out, "http://"+host)
	} else if strings.HasPrefix(base, "http://") {
		out = append(out, "https://"+host)
	}
	return out
}

// matchPrefix 在 s 上匹配候选前缀，要求边界是 "/" 或结尾（避免 host 前缀误伤同前缀的其它域名）。
func matchPrefix(s string, candidates []string) (string, bool) {
	for _, c := range candidates {
		if c == "" || !strings.HasPrefix(s, c) {
			continue
		}
		if len(s) == len(c) || s[len(c)] == '/' {
			return s[len(c):], true
		}
	}
	return "", false
}

// HasStoragePrefix 判断地址 s 是否指向本站资源。
// ⚠️ base 为空时**必须**返回 false：否则 strings.HasPrefix(s, "") 恒真，
// 会把外链图片当成自家的去加 thumb_ 前缀，生成 https://other.com/thumb_x.webp 这种死链。
func HasStoragePrefix(s, base string) bool {
	s = strings.TrimSpace(s)
	if s == "" || NormalizeStorageUrl(base) == "" {
		return false
	}
	_, ok := matchPrefix(s, storagePrefixCandidates(base))
	return ok
}

// IsLocalResource 判断地址是否是本站资源（也就是能不能安全地加 thumb_ 前缀）。
//   - 相对路径（"uploads/a.png"、"/uploads/a.png"）必然是本站的
//   - 绝对地址只有命中所配置的资源地址才算本站（外链、data:、blob: 都不算）
// ⚠️ base 为空时绝对地址一律判否：这正是「外链图片被加上 thumb_ 变成死链」的根因。
func IsLocalResource(s, base string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if !IsAbsoluteUrl(s) {
		return true
	}
	return HasStoragePrefix(s, base)
}

// StripStorageUrl 去掉资源前缀，得到应该存进库的相对路径。
// base 为空 / 不匹配时原样返回（不会把外链截断）。
func StripStorageUrl(s, base string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	rest, ok := matchPrefix(s, storagePrefixCandidates(base))
	if !ok {
		return s
	}
	if rest == "" {
		return ""
	}
	if strings.HasPrefix(rest, "/") {
		return rest
	}
	return s
}

package provider

import (
	"net/url"
	"strings"
)

// IsAllowedCORSOrigin 判断跨域请求的 Origin 是否属于「本安装自己」。
//
// 为什么需要它：多站点模式下每个站点都有自己的域名，独立部署的管理端、H5、
// 以及其他站点页面经常要跨域调用 /api/*。旧实现只放行 127.0.0.1/localhost，
// 正式域名一律拿不到 Access-Control-Allow-Origin，浏览器会直接拦掉请求，
// 表现为「开了多站点之后 API 跨站用不了」。
//
// 放行范围刻意保持收敛，不用通配符（否则任意第三方页面都能带凭据读接口）：
//   - 本机地址（localhost / 127.0.0.1 / [::1]，任意端口），便于本地调试；
//   - 本安装内任一站点配置的域名：BaseUrl / FrontUrl / MobileUrl / AdminUrl，
//     www 前缀差异视为同一域名（配置里写不写 www 都常见）。
//
// 同源请求（Origin 的 host 与请求 Host 一致）不需要 CORS 头，所以这里不做判断，
// 也避免依赖可被请求头伪造的 X-Host。
func IsAllowedCORSOrigin(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	originHost := normalizedHost(parsed.Hostname())
	if isLoopbackHost(originHost) {
		return true
	}

	for _, w := range websites.Values() {
		if w == nil || w.System == nil {
			continue
		}
		for _, raw := range []string{w.System.BaseUrl, w.System.FrontUrl, w.System.MobileUrl, w.System.AdminUrl} {
			if raw == "" {
				continue
			}
			p, err := url.Parse(raw)
			if err != nil || p.Hostname() == "" {
				continue
			}
			if normalizedHost(p.Hostname()) == originHost {
				return true
			}
		}
	}

	return false
}

// normalizedHost 统一小写并去掉 www. 前缀。
func normalizedHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	return strings.TrimPrefix(host, "www.")
}

func isLoopbackHost(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	}
	return strings.HasSuffix(host, ".localhost")
}

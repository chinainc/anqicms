package controller

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/provider"
)

// BusinessPage 输出商家工作台页面。
//
// 页面本体与静态资源都放在 public/business/ 下：
//   - /business/assets/... 这类带后缀的资源由 FileServe 直接从 public 目录读盘返回（生产环境由 nginx 直出）；
//   - 只有 /business、/business/ 这类"目录地址"需要这里兜底指向 index.html。
//
// 路由里用 {p:path} 兜住了子路径（见 route/base.go 的说明），所以这里还要区分两种情况：
// p 为空 → 目录地址，出 index.html；p 非空 → 静态资源，交给 FileServe。
func BusinessPage(ctx iris.Context) {
	if sub := ctx.Params().Get("p"); sub != "" && sub != "/" {
		if FileServe(ctx) {
			return
		}
		NotFound(ctx)
		return
	}

	// 目录地址统一带尾斜杠：页面里的静态资源用的是相对路径（assets/css/style.css），
	// 直接访问 /business 时浏览器会按根路径 /assets/... 去解析而 404，所以先 301 到 /business/。
	if reqPath := ctx.RequestPath(false); !strings.HasSuffix(reqPath, "/") {
		ctx.Redirect(reqPath+"/", 301)
		return
	}

	currentSite := provider.CurrentSite(ctx)
	if currentSite == nil {
		NotFound(ctx)
		return
	}

	pageFile := filepath.Join(currentSite.RootPath, "public", "business", "index.html")
	info, err := os.Stat(pageFile)
	if err != nil || info.IsDir() {
		NotFound(ctx)
		return
	}

	ctx.ServeFile(pageFile)
}

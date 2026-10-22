package controller

import (
	"net/url"
	"testing"

	"kandaoni.com/anqicms/provider"
)

func TestNewMemberPager(t *testing.T) {
	pager := newMemberPager("/member/favorites", url.Values{}, 1, 10, 25)

	if pager.TotalPage != 3 {
		t.Fatalf("TotalPage = %d, want 3", pager.TotalPage)
	}
	if pager.PrevLink != "" {
		t.Errorf("第一页不应有上一页链接，得到 %q", pager.PrevLink)
	}
	if pager.NextLink != "/member/favorites?page=2" {
		t.Errorf("NextLink = %q", pager.NextLink)
	}
	if len(pager.Links) != 3 {
		t.Fatalf("Links 数量 = %d, want 3", len(pager.Links))
	}
	if !pager.Links[0].Current {
		t.Error("第一页应为当前页")
	}
	if pager.Links[1].Link != "/member/favorites?page=2" {
		t.Errorf("第二页链接 = %q", pager.Links[1].Link)
	}
}

func TestNewMemberPagerKeepsQuery(t *testing.T) {
	query := url.Values{"status": {"waiting"}}
	pager := newMemberPager("/member/orders", query, 2, 10, 30)

	// 第一页不带 page 参数，只保留筛选条件
	if pager.PrevLink != "/member/orders?status=waiting" {
		t.Errorf("PrevLink = %q", pager.PrevLink)
	}
	if pager.NextLink != "/member/orders?page=3&status=waiting" {
		t.Errorf("NextLink = %q", pager.NextLink)
	}
	// 原 query 不能被修改
	if query.Get("page") != "" {
		t.Error("base query 被写入了 page 参数")
	}
}

func TestNewMemberPagerEmptyTotal(t *testing.T) {
	pager := newMemberPager("/member/orders", url.Values{}, 1, 12, 0)
	if pager.TotalPage != 1 {
		t.Fatalf("空列表 TotalPage = %d, want 1", pager.TotalPage)
	}
	if pager.PrevLink != "" || pager.NextLink != "" {
		t.Error("空列表不应有翻页链接")
	}
	if len(pager.Links) != 1 || !pager.Links[0].Current {
		t.Error("空列表应只有第 1 页且为当前页")
	}
}

func TestMemberOrderStatusValid(t *testing.T) {
	for _, status := range []string{"", "waiting", "paid", "delivery", "finished", "refunding", "closed"} {
		if !memberOrderStatusValid(status) {
			t.Errorf("%q 应为合法状态", status)
		}
	}
	for _, status := range []string{"abc", "1", "paid2", " "} {
		if memberOrderStatusValid(status) {
			t.Errorf("%q 应为非法状态", status)
		}
	}
}

func TestMemberNavLinks(t *testing.T) {
	site := &provider.Website{}
	nav := memberNav(site)
	if len(nav) != len(memberNavEntries) {
		t.Fatalf("菜单数量 = %d, want %d", len(nav), len(memberNavEntries))
	}
	if nav[0].Route != "index" || nav[0].Link != "/member" {
		t.Errorf("首页菜单异常: %+v", nav[0])
	}
	for _, item := range nav[1:] {
		if item.Link != "/member/"+item.Route {
			t.Errorf("%s 链接异常: %s", item.Route, item.Link)
		}
	}
}

func TestMemberPagesWhitelist(t *testing.T) {
	for route, tpl := range memberPages {
		if tpl == "" {
			t.Errorf("%s 的模板为空", route)
		}
	}
	if _, ok := memberPages["index"]; !ok {
		t.Error("缺少 index 页面")
	}
	if _, ok := memberPages["../etc/passwd"]; ok {
		t.Error("白名单不应包含路径穿越")
	}
}

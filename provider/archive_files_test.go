package provider

import (
	"reflect"
	"testing"
)

func TestNormalizeUploadRef(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"/uploads/202309/15/a.webp", "uploads/202309/15/a.webp"},
		{"uploads/202309/15/a.webp", "uploads/202309/15/a.webp"},
		{"http://127.0.0.1:8001/uploads/202309/15/a.webp", "uploads/202309/15/a.webp"},
		{"https://cdn.example.com/uploads/202309/15/a.webp?v=1", "uploads/202309/15/a.webp"},
		{"//cdn.example.com/uploads/a.png#top", "uploads/a.png"},
		{"/uploads/%E4%B8%AD%E6%96%87.png", "uploads/中文.png"},
		{"\"uploads/a.png\"", "uploads/a.png"},
		{"uploads/a.png.", "uploads/a.png"},
		{"https://other.com/static/a.png", ""},
		{"data:image/png;base64,AAAA", ""},
		{"uploads/../etc/passwd", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := NormalizeUploadRef(c.in); got != c.want {
			t.Errorf("NormalizeUploadRef(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExtractUploadRefs(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "html 正文",
			in:   `<p>hi</p><img src="/uploads/202309/15/a.webp" alt="x"><a href='uploads/files/doc.pdf'>下载</a>`,
			want: []string{"uploads/202309/15/a.webp", "uploads/files/doc.pdf"},
		},
		{
			name: "图集 JSON",
			in:   `{"/uploads/202309/15/a.webp","/uploads/202309/15/b.webp"}`,
			want: []string{"uploads/202309/15/a.webp", "uploads/202309/15/b.webp"},
		},
		{
			name: "绝对地址与去重",
			in:   `<img src="http://127.0.0.1:8001/uploads/a.png"><img src="/uploads/a.png">`,
			want: []string{"uploads/a.png"},
		},
		{
			name: "外链与 data 不算本站资源",
			in:   `<img src="https://other.com/a.png"><img src="data:image/png;base64,AAAA">`,
			want: nil,
		},
		{
			name: "style 背景图",
			in:   `<div style="background:url(/uploads/bg.png)"></div>`,
			want: []string{"uploads/bg.png"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ExtractUploadRefs(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("ExtractUploadRefs(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

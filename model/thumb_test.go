package model

import "testing"

// 资源地址拼接曾经出过三类错误地址，这里把它们钉死：
//  1. 资源地址带尾斜杠 → http://host//uploads/x.webp
//  2. 漏拼接符 → http://hostuploads/x.webp
//  3. 资源地址为空时把外链也加上 thumb_ → https://other.com/thumb_x.webp（死链）
func TestArchiveGetThumb(t *testing.T) {
	cases := []struct {
		name         string
		storageUrl   string
		images       []string
		defaultThumb string
		wantLogo     string
		wantThumb    string
	}{
		{
			name:       "资源地址带尾斜杠，不能出现双斜杠",
			storageUrl: "https://cdn.a.com/",
			images:     []string{"/uploads/2024/a.webp"},
			wantLogo:   "https://cdn.a.com/uploads/2024/a.webp",
			wantThumb:  "https://cdn.a.com/uploads/2024/thumb_a.webp",
		},
		{
			name:       "相对路径没有前导斜杠也能拼",
			storageUrl: "https://cdn.a.com",
			images:     []string{"uploads/2024/a.webp"},
			wantLogo:   "https://cdn.a.com/uploads/2024/a.webp",
			wantThumb:  "https://cdn.a.com/uploads/2024/thumb_a.webp",
		},
		{
			name:       "资源地址为空时，本地相对路径仍然要出缩略图",
			storageUrl: "",
			images:     []string{"/uploads/2024/a.webp"},
			wantLogo:   "/uploads/2024/a.webp",
			wantThumb:  "/uploads/2024/thumb_a.webp",
		},
		{
			name:       "资源地址为空时，外链不能被拼前缀也不能加 thumb_",
			storageUrl: "",
			images:     []string{"https://other.com/p/a.webp"},
			wantLogo:   "https://other.com/p/a.webp",
			wantThumb:  "https://other.com/p/a.webp",
		},
		{
			name:       "配置了资源地址时，外链同样不能被加 thumb_",
			storageUrl: "https://cdn.a.com",
			images:     []string{"https://other.com/p/a.webp"},
			wantLogo:   "https://other.com/p/a.webp",
			wantThumb:  "https://other.com/p/a.webp",
		},
		{
			name:       "协议相对地址保持原样",
			storageUrl: "https://cdn.a.com",
			images:     []string{"//other.com/p/a.webp"},
			wantLogo:   "//other.com/p/a.webp",
			wantThumb:  "//other.com/p/a.webp",
		},
		{
			name:       "本站资源用协议相对写法也要加 thumb_",
			storageUrl: "https://cdn.a.com",
			images:     []string{"//cdn.a.com/uploads/a.webp"},
			wantLogo:   "//cdn.a.com/uploads/a.webp",
			wantThumb:  "//cdn.a.com/uploads/thumb_a.webp",
		},
		{
			name:       "svg 不加 thumb_",
			storageUrl: "https://cdn.a.com",
			images:     []string{"/uploads/a.svg"},
			wantLogo:   "https://cdn.a.com/uploads/a.svg",
			wantThumb:  "https://cdn.a.com/uploads/a.svg",
		},
		{
			name:         "没有图片时用默认缩略图",
			storageUrl:   "https://cdn.a.com/",
			defaultThumb: "/static/default/thumb.png",
			wantLogo:     "https://cdn.a.com/static/default/thumb.png",
			wantThumb:    "https://cdn.a.com/static/default/thumb_thumb.png",
		},
		{
			name:         "默认缩略图是外链时不能加 thumb_",
			storageUrl:   "https://cdn.a.com",
			defaultThumb: "https://other.com/def.png",
			wantLogo:     "https://other.com/def.png",
			wantThumb:    "https://other.com/def.png",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := &Archive{Images: append([]string(nil), c.images...)}
			a.GetThumb(c.storageUrl, c.defaultThumb)
			if a.Logo != c.wantLogo {
				t.Fatalf("Logo = %q, want %q", a.Logo, c.wantLogo)
			}
			if a.Thumb != c.wantThumb {
				t.Fatalf("Thumb = %q, want %q", a.Thumb, c.wantThumb)
			}
		})
	}
}

func TestAttachmentGetThumb(t *testing.T) {
	// 资源地址为空 + 外链 Logo：不能拼出 https://other.com 之外的怪地址，也不能加 thumb_
	attach := &Attachment{FileLocation: "uploads/2024/v.mp4", Logo: "https://other.com/p/poster.webp", IsImage: 0}
	attach.GetThumb("")
	if attach.FilePath != "/uploads/2024/v.mp4" {
		t.Fatalf("FilePath = %q, want /uploads/2024/v.mp4", attach.FilePath)
	}
	if attach.Thumb != "https://other.com/p/poster.webp" {
		t.Fatalf("Thumb = %q, want 原样外链", attach.Thumb)
	}

	// 本地图片：缩略图是本目录下的 thumb_ 文件
	img := &Attachment{FileLocation: "uploads/2024/a.webp", Logo: "uploads/2024/a.webp", IsImage: 1}
	img.GetThumb("https://cdn.a.com/")
	if img.FilePath != "https://cdn.a.com/uploads/2024/a.webp" {
		t.Fatalf("FilePath = %q", img.FilePath)
	}
	if img.Thumb != "https://cdn.a.com/uploads/2024/thumb_a.webp" {
		t.Fatalf("Thumb = %q", img.Thumb)
	}
}

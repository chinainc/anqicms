// Command targz 把目录打成 tar.gz，并显式写入权限位。
//
// 为什么不用系统的 tar.exe：Windows 上创建的文件没有「可执行位」，bsdtar 打出来
// 全是 0644，用户解包后 `./install.sh` / `./anqicms` 会 Permission denied；
// 另外 Compress-Archive 在 PowerShell 5.1 里会用反斜杠做目录分隔符。
// 这里用 Go 自己打，权限与路径完全可控。
//
// 用法：
//
//	go run ./tools/targz -src ...\anqicms-v3.6.8-linux-amd64 -out ...\anqicms-v3.6.8-linux-amd64.tar.gz
package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	src := flag.String("src", "", "源目录")
	out := flag.String("out", "", "输出 tar.gz 路径")
	flag.Parse()
	if *src == "" || *out == "" {
		log.Fatal("usage: targz -src <dir> -out <file.tar.gz>")
	}
	srcAbs, err := filepath.Abs(*src)
	if err != nil {
		log.Fatal(err)
	}
	base := filepath.Base(srcAbs)
	if fi, err := os.Stat(srcAbs); err != nil || !fi.IsDir() {
		log.Fatalf("源目录不存在：%s", srcAbs)
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	gz.Name = base + ".tar"
	gz.ModTime = time.Now()
	tw := tar.NewWriter(gz)

	// 顶层目录条目
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir, Name: base + "/", Mode: 0o755,
		ModTime: time.Now(), Format: tar.FormatPAX,
	}); err != nil {
		log.Fatal(err)
	}

	count := 0
	err = filepath.Walk(srcAbs, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcAbs, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		name := base + "/" + filepath.ToSlash(rel)

		switch {
		case fi.IsDir():
			return tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir, Name: name + "/", Mode: 0o755,
				ModTime: fi.ModTime(), Format: tar.FormatPAX,
			})
		case fi.Mode().IsRegular():
			mode := int64(0o644)
			if isExecutable(rel) {
				mode = 0o755
			}
			h := &tar.Header{
				Typeflag: tar.TypeReg, Name: name, Mode: mode, Size: fi.Size(),
				ModTime: fi.ModTime(), Format: tar.FormatPAX,
				Uname: "root", Gname: "root",
			}
			if err := tw.WriteHeader(h); err != nil {
				return err
			}
			fp, err := os.Open(p)
			if err != nil {
				return err
			}
			_, err = io.Copy(tw, fp)
			fp.Close()
			if err != nil {
				return err
			}
			count++
			return nil
		default:
			return nil // 跳过符号链接等
		}
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		log.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("tar.gz -> %s (%d files)\n", *out, count)
}

// isExecutable 判断是否应带可执行位：shell 脚本与主程序二进制。
func isExecutable(rel string) bool {
	base := filepath.Base(rel)
	if strings.HasSuffix(base, ".sh") {
		return true
	}
	switch base {
	case "anqicms":
		return true
	}
	return false
}

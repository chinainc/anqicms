// Command zipdir 把目录打成 zip（路径统一用正斜杠、文件名按 UTF-8 写）。
//
// 为什么不用 PowerShell 的 Compress-Archive / ZipFile.CreateFromDirectory：
// .NET Framework（PS 5.1 用的那套）打包时目录分隔符不归一，结果 zip 里是
// "sql\xxx.sql" 这种反斜杠路径 —— Linux 上解包会得到名字里带反斜杠的文件。
//
// 用法：
//
//	go run ./tools/zipdir -src E:\CodeBuddy\anqicms发布\anqicms-v3.6.8-windows -out ...\anqicms-v3.6.8-windows.zip
package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func main() {
	src := flag.String("src", "", "源目录")
	out := flag.String("out", "", "输出 zip 路径")
	flag.Parse()
	if *src == "" || *out == "" {
		log.Fatal("usage: zipdir -src <dir> -out <file.zip>")
	}
	srcAbs, err := filepath.Abs(*src)
	if err != nil {
		log.Fatal(err)
	}
	if fi, err := os.Stat(srcAbs); err != nil || !fi.IsDir() {
		log.Fatalf("源目录不存在：%s", srcAbs)
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)

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
		// 关键：反斜杠 → 正斜杠（zip 规范只认 '/'）
		name := filepath.ToSlash(rel)

		if fi.IsDir() {
			h := &zip.FileHeader{Name: name + "/", Method: zip.Store}
			h.SetMode(fi.Mode() | 0o111)
			h.Modified = fi.ModTime()
			_, err := zw.CreateHeader(h)
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}

		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(0o644)
		h.Modified = fi.ModTime()
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		fp, err := os.Open(p)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, fp)
		fp.Close()
		if err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("zip -> %s (%d files)\n", *out, count)
}

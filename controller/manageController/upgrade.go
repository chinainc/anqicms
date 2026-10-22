package manageController

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/library"
	"kandaoni.com/anqicms/provider"
)

// 上传升级包（.zip / .tar.gz）并就地升级
//
// 与官方在线升级（下载 anqicms-<os>-<arch>-v<ver>.zip）的区别：
//   - 升级包由管理员自己上传，服务器不需要访问外网；
//   - 支持 zip 与 tar.gz 两种格式（按文件魔数识别，不看扩展名）；
//   - 包内根目录若存在 version.json，自动读取版本号与更新说明，无需手填；
//   - 升级完成后由前端调用 /system/api/anqi/restart 重启进程（服务端不自杀）。
const (
	// upgradeUploadMaxSize 单个升级包上限（512MB）
	upgradeUploadMaxSize = 512 << 20
	// upgradeUnpackMaxSize 解压后总大小上限，防 zip bomb
	upgradeUnpackMaxSize = 2 << 30
	// upgradeMaxFileCount 解压条目数上限
	upgradeMaxFileCount = 20000
)

// upgradeManifest 升级包内的 version.json（字段与发版脚本保持一致，全部可缺省）
type upgradeManifest struct {
	Version string   `json:"version"`
	Date    string   `json:"date"`
	Note    string   `json:"note"`
	SQL     []string `json:"sql"`
}

type upgradeStats struct {
	Version string   `json:"version"`
	Files   int      `json:"files"`
	Skipped int      `json:"skipped"`
	Bytes   int64    `json:"bytes"`
	SQL     []string `json:"sql"`
}

// GetUpgradeEnv 上传升级页需要的环境信息（当前版本、平台、主程序名、上传上限）
func GetUpgradeEnv(ctx iris.Context) {
	exec := ""
	if p, err := os.Executable(); err == nil {
		exec = filepath.Base(p)
	}
	ctx.JSON(iris.Map{
		"code": config.StatusOK,
		"msg":  "",
		"data": iris.Map{
			"version":    config.Version,
			"trial":      config.Trial,
			"goos":       runtime.GOOS,
			"goarch":     runtime.GOARCH,
			"exec":       exec,
			"exec_path":  config.ExecPath,
			"max_size":   upgradeUploadMaxSize,
			"skip_rules": "template/、public/、data/、uploads/、logs/、config.json、*.sh、*.bat 不会被覆盖",
		},
	})
}

// VersionUploadUpgrade 上传升级包并执行升级
//
// 参数说明：
//   - 表单文件 "file": 升级包（.zip 或 .tar.gz）。
//   - 表单 "version": 版本号，包内有 version.json 时可留空。
//   - 表单 "force": 传 "1" 表示同版本也强制升级。
func VersionUploadUpgrade(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)

	ctx.SetMaxRequestBodySize(upgradeUploadMaxSize + 1<<20)

	file, header, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": "请选择要上传的升级包：" + err.Error()})
		return
	}
	defer file.Close()

	if header.Size <= 0 {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": "升级包是空文件"})
		return
	}
	if header.Size > upgradeUploadMaxSize {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": fmt.Sprintf("升级包最大 %d MB，当前 %.1f MB", upgradeUploadMaxSize>>20, float64(header.Size)/(1<<20))})
		return
	}

	// 上传的文件落在系统临时目录，升级结束后删除
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("anqicms-upgrade-%d", time.Now().UnixNano()))
	if err = os.MkdirAll(tmpDir, 0o755); err != nil {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": "无法创建临时目录：" + err.Error()})
		return
	}
	defer os.RemoveAll(tmpDir)

	pkgPath := filepath.Join(tmpDir, filepath.Base(header.Filename))
	dst, err := os.Create(pkgPath)
	if err != nil {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": "写入临时文件失败：" + err.Error()})
		return
	}
	written, err := io.Copy(dst, file)
	_ = dst.Close()
	if err != nil {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": "保存升级包失败：" + err.Error()})
		return
	}

	kind := detectUpgradeArchive(pkgPath)
	if kind == "" {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": "无法识别的升级包格式，仅支持 zip 与 tar.gz"})
		return
	}

	// 读取包内 version.json（可缺省）
	manifest, _ := readUpgradeManifest(pkgPath, kind)
	version := strings.TrimSpace(ctx.PostValue("version"))
	if version == "" && manifest != nil {
		version = strings.TrimSpace(manifest.Version)
	}
	force := ctx.PostValue("force") == "1"
	if version == "" && !force {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": "升级包内没有 version.json，请填写版本号或勾选强制升级"})
		return
	}
	if !force && version != "" {
		if library.VersionCompare(version, config.Version) <= 0 {
			ctx.JSON(iris.Map{
				"code": config.StatusFailed,
				"msg":  fmt.Sprintf("包版本 %s 不高于当前版本 %s；如确认要覆盖安装，请勾选「强制升级」", version, config.Version),
			})
			return
		}
	}

	stats, err := applyUpgradeArchive(pkgPath, kind)
	if err != nil {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": "升级失败：" + err.Error()})
		return
	}
	stats.Version = version
	if manifest != nil && len(manifest.SQL) > 0 {
		stats.SQL = manifest.SQL
	}

	currentSite.AddAdminLog(ctx, fmt.Sprintf("上传升级包升级：%s -> %s（%s，%d 个文件，%.1f MB）",
		config.Version, version, filepath.Base(header.Filename), stats.Files, float64(written)/(1<<20)))

	msg := fmt.Sprintf("升级完成：写入 %d 个文件，跳过 %d 个；请重启服务后生效", stats.Files, stats.Skipped)
	if len(stats.SQL) > 0 {
		msg += fmt.Sprintf("。包内附带 %d 个 SQL，需要另行执行", len(stats.SQL))
	}
	ctx.JSON(iris.Map{
		"code": config.StatusOK,
		"msg":  msg,
		"data": stats,
	})
}

// detectUpgradeArchive 按魔数识别压缩格式，返回 "zip" / "tar.gz" / ""
func detectUpgradeArchive(file string) string {
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()

	head := make([]byte, 4)
	n, _ := io.ReadFull(f, head)
	if n < 2 {
		return ""
	}
	if head[0] == 'P' && head[1] == 'K' {
		return "zip"
	}
	if head[0] == 0x1f && head[1] == 0x8b {
		return "tar.gz"
	}

	return ""
}

// readUpgradeManifest 读取包内根目录（或唯一顶层目录下）的 version.json
func readUpgradeManifest(file, kind string) (*upgradeManifest, error) {
	if kind == "zip" {
		reader, err := zip.OpenReader(file)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		_, data, err := readZipEntry(reader.File, "version.json")
		if err != nil {
			return nil, err
		}
		var manifest upgradeManifest
		if err = json.Unmarshal(data, &manifest); err != nil {
			return nil, err
		}
		return &manifest, nil
	}

	tr, closer, err := openTarGz(file)
	if err != nil {
		return nil, err
	}
	defer closer()
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, os.ErrNotExist
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if path.Base(strings.TrimPrefix(hdr.Name, "./")) != "version.json" {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(tr, 1<<20))
		if err != nil {
			return nil, err
		}
		var manifest upgradeManifest
		if err = json.Unmarshal(data, &manifest); err != nil {
			return nil, err
		}
		return &manifest, nil
	}
}

func readZipEntry(files []*zip.File, baseName string) (string, []byte, error) {
	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}
		name := strings.TrimPrefix(strings.ReplaceAll(f.Name, "\\", "/"), "./")
		if path.Base(name) != baseName {
			continue
		}
		// 只接受根目录或唯一一层前缀下的文件
		if strings.Count(strings.Trim(name, "/"), "/") > 1 {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return name, nil, err
		}
		data, err := io.ReadAll(io.LimitReader(rc, 1<<20))
		_ = rc.Close()
		return name, data, err
	}
	return "", nil, os.ErrNotExist
}

func openTarGz(file string) (*tar.Reader, func(), error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, nil, err
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	closer := func() {
		_ = gz.Close()
		_ = f.Close()
	}
	return tar.NewReader(gz), closer, nil
}

// applyUpgradeArchive 解压升级包并覆盖到程序目录
func applyUpgradeArchive(pkgPath, kind string) (*upgradeStats, error) {
	// 统一用 zip 读取器与 tar 读取器的公共回调处理
	handler := func(name string, mode os.FileMode, isDir bool, r io.Reader, size int64) error {
		return writeUpgradeEntry(name, mode, isDir, r, size)
	}

	var stats *upgradeStats
	var err error
	if kind == "zip" {
		stats, err = upgradeFromZip(pkgPath, handler)
	} else {
		stats, err = upgradeFromTarGz(pkgPath, handler)
	}
	if err != nil {
		return nil, err
	}
	// 包内带主程序时替换（.new -> 当前程序）
	if err = replaceUpgradeExecutable(); err != nil {
		return nil, err
	}

	return stats, nil
}

type upgradeEntryFunc func(name string, mode os.FileMode, isDir bool, r io.Reader, size int64) error

// upgradeCollector 统一处理条目：剥离公共前缀、过滤不覆盖的文件、写盘
type upgradeCollector struct {
	stats    *upgradeStats
	prefix   string
	rootDone bool
}

func upgradeFromZip(file string, fn upgradeEntryFunc) (*upgradeStats, error) {
	reader, err := zip.OpenReader(file)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	collector := &upgradeCollector{
		stats:  &upgradeStats{},
		prefix: commonPrefixFromZip(reader.File),
	}

	for _, f := range reader.File {
		if collector.stats.Files+collector.stats.Skipped > upgradeMaxFileCount {
			return nil, fmt.Errorf("升级包条目超过 %d 个，已中止", upgradeMaxFileCount)
		}
		if collector.stats.Bytes > upgradeUnpackMaxSize {
			return nil, fmt.Errorf("解压后体积超过上限，已中止")
		}
		isDir := f.FileInfo().IsDir()
		name := collector.relativeName(f.Name)
		if name == "" {
			continue
		}
		if !isDir && shouldSkipUpgradeFile(name) {
			collector.stats.Skipped++
			continue
		}
		if isDir {
			if err = fn(name, f.Mode(), true, nil, 0); err != nil {
				return nil, err
			}
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		err = fn(name, f.Mode(), false, rc, int64(f.UncompressedSize64))
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		collector.stats.Files++
		collector.stats.Bytes += int64(f.UncompressedSize64)
	}

	return collector.stats, nil
}

func upgradeFromTarGz(file string, fn upgradeEntryFunc) (*upgradeStats, error) {
	// 先扫一遍取公共前缀
	tr, closer, err := openTarGz(file)
	if err != nil {
		return nil, err
	}
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			closer()
			return nil, err
		}
		names = append(names, hdr.Name)
	}
	closer()

	collector := &upgradeCollector{
		stats:  &upgradeStats{},
		prefix: commonPrefixFromNames(names),
	}

	tr, closer, err = openTarGz(file)
	if err != nil {
		return nil, err
	}
	defer closer()

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if collector.stats.Files+collector.stats.Skipped > upgradeMaxFileCount {
			return nil, fmt.Errorf("升级包条目超过 %d 个，已中止", upgradeMaxFileCount)
		}
		if collector.stats.Bytes > upgradeUnpackMaxSize {
			return nil, fmt.Errorf("解压后体积超过上限，已中止")
		}
		name := collector.relativeName(hdr.Name)
		if name == "" {
			continue
		}
		isDir := hdr.Typeflag == tar.TypeDir
		if !isDir && hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			// 跳过链接等非常规条目，避免越权写入
			collector.stats.Skipped++
			continue
		}
		if !isDir && shouldSkipUpgradeFile(name) {
			collector.stats.Skipped++
			continue
		}
		if isDir {
			if err = fn(name, os.FileMode(hdr.Mode), true, nil, 0); err != nil {
				return nil, err
			}
			continue
		}
		if err = fn(name, os.FileMode(hdr.Mode), false, tr, hdr.Size); err != nil {
			return nil, err
		}
		collector.stats.Files++
		collector.stats.Bytes += hdr.Size
	}

	return collector.stats, nil
}

// relativeName 去掉包内公共前缀，返回相对程序目录的路径
func (c *upgradeCollector) relativeName(name string) string {
	name = strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "./")
	name = strings.TrimPrefix(name, "/")
	if c.prefix != "" {
		name = strings.TrimPrefix(name, c.prefix+"/")
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "." {
		return ""
	}

	return name
}

func commonPrefixFromZip(files []*zip.File) string {
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, f.Name)
	}
	return commonPrefixFromNames(names)
}

// commonPrefixFromNames 计算所有条目共同的唯一顶层目录（如 anqicms-v3.7.0/），
// 只有「所有条目都在同一目录下，且该目录本身不是文件」时才返回
func commonPrefixFromNames(names []string) string {
	prefix := ""
	for _, name := range names {
		name = strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "./")
		name = strings.TrimPrefix(name, "/")
		if name == "" {
			continue
		}
		first := name
		if idx := strings.Index(name, "/"); idx >= 0 {
			first = name[:idx]
		} else {
			// 有根级文件，不剥离前缀
			return ""
		}
		if prefix == "" {
			prefix = first
			continue
		}
		if prefix != first {
			return ""
		}
	}
	// 顶层目录本身可能也是一个条目
	for _, name := range names {
		n := strings.TrimSuffix(strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "./"), "/")
		if n == prefix {
			continue
		}
	}
	return prefix
}

// shouldSkipUpgradeFile 不覆盖的文件：模板、静态资源、用户数据与脚本
func shouldSkipUpgradeFile(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range []string{
		"template/", "public/", "data/", "uploads/", "logs/", "static/",
	} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	for _, suffix := range []string{".sh", ".bat", ".cmd", ".ps1"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	switch lower {
	case "config.json", "favicon.ico", "version.json":
		return true
	}

	return false
}

// writeUpgradeEntry 写入单个条目；主程序先写 .new，全部成功后再替换
func writeUpgradeEntry(name string, mode os.FileMode, isDir bool, r io.Reader, _ int64) error {
	target := filepath.Join(config.ExecPath, filepath.FromSlash(name))
	// 防止 ../ 越界
	cleanTarget := filepath.Clean(target)
	base := filepath.Clean(config.ExecPath)
	if cleanTarget != base && !strings.HasPrefix(cleanTarget, base+string(os.PathSeparator)) {
		return fmt.Errorf("非法的包内路径：%s", name)
	}

	if isDir {
		return os.MkdirAll(cleanTarget, 0o755)
	}

	if isUpgradeExecutable(name) {
		// 主程序写到 .new，最后统一替换，避免运行中文件被占用
		cleanTarget += ".new"
	}
	if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if isUpgradeExecutable(name) {
		_ = os.Chmod(cleanTarget, 0o755)
	}

	return nil
}

func isUpgradeExecutable(name string) bool {
	base := strings.ToLower(path.Base(name))

	return base == "anqicms" || base == "anqicms.exe" || base == "anqicms-server" || base == "anqicms-server.exe"
}

// replaceUpgradeExecutable 用解压出的 .new 替换当前主程序
func replaceUpgradeExecutable() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self = filepath.Clean(self)
	newPath := self + ".new"
	if _, err = os.Stat(newPath); err != nil {
		// 包里没有主程序，说明只更新了其他文件
		return nil
	}
	oldPath := self + ".old"
	_ = os.Remove(oldPath)
	if err = os.Rename(self, oldPath); err != nil {
		return fmt.Errorf("备份旧主程序失败（可能被占用）：%w", err)
	}
	if err = os.Rename(newPath, self); err != nil {
		_ = os.Rename(oldPath, self)
		return fmt.Errorf("替换主程序失败：%w", err)
	}
	_ = os.Chmod(self, 0o755)
	_ = os.Remove(oldPath)

	return nil
}

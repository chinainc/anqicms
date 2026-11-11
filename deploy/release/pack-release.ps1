# AnqiCMS release packer
#
# 打包逻辑移植自 E:\CodeBuddy\tmago\deploy\release\pack-release.ps1（2026-10-22）：
# 参数化版本、正斜杠 zip、tar.gz 可执行位、Linux 双架构、在线一键安装脚本、
# version.json + SHA256SUMS。
#
# 产出（默认输出目录 E:\CodeBuddy\anqicms发布）：
#   1) 升级包（给已装好的站点用，后台「系统 » 上传升级包」）：
#        anqicms-v<ver>-windows-upgrade.zip   Windows 升级包（anqicms.exe + system + locales + deploy + version.json）
#        anqicms-v<ver>-linux-upgrade.zip     Linux   升级包（anqicms     + 同上）
#        anqicms-v<ver>.zip                   兼容旧后台的单包（内容同 windows-upgrade）
#        ⚠️ 升级包里**不放** template/ public/ data/ logs/ config.json：
#           后台升级逻辑本身也会跳过这些（用户改过的模板与上传文件不能被覆盖）。
#   2) 完整安装包
#        anqicms-v<ver>-windows.zip          Windows 解压即用（内置 /install 安装向导）
#        anqicms-v<ver>-linux-amd64.tar.gz   Linux x86_64（install.sh + systemd）
#        anqicms-v<ver>-linux-arm64.tar.gz   Linux arm64（-NoArm64 可跳过）
#   3) 在线一键安装：<发布目录>\install.sh、<发布目录>\uninstall.sh
#        curl -fsSL <镜像>/install.sh | sh   → 读 version.json → 选架构 → 校验 sha256 → 解包 → 装
#        curl -fsSL <镜像>/uninstall.sh | sh → 停并删 systemd 服务 + 询问是否删安装目录
#   4) version.json（升级读 packages.*.url/checksum；安装读 install.*）
#   5) SHA256SUMS.txt
#
# 用法：
#   powershell -NoProfile -ExecutionPolicy Bypass -File deploy\release\pack-release.ps1
#     -Version 3.6.9       指定版本（不传则用下面的 $ver）
#     -Mirror https://...  写进在线安装脚本的镜像地址
#     -NoArm64             跳过 linux/arm64
#     -OutDir D:\publish   换输出目录（默认 E:\CodeBuddy\anqicms发布）
param(
  [string]$Version = '',
  [string]$Mirror  = '',
  [switch]$NoArm64,
  [string]$OutDir  = ''
)

$ErrorActionPreference = 'Stop'

$root   = 'e:\CodeBuddy\anqicms'
$mirror = if ($Mirror)  { $Mirror }  else { 'https://yingcom.oss-us-west-1.aliyuncs.com/anqicms' }
$out    = if ($OutDir) { $OutDir } else { 'E:\CodeBuddy\anqicms发布' }

# 版本号默认取 config/constant.go 的 `const Version = "x.y.z"`（后台「关于」也读它），
# 这样改代码里的版本号就等于改发版号，不会出现「程序说一个版本、包名说另一个」。
$ver = $Version
if (-not $ver) {
  $constFile = Join-Path $root 'config\constant.go'
  if (Test-Path $constFile) {
    $m = Select-String -Path $constFile -Pattern 'const\s+Version\s*=\s*"([^"]+)"' | Select-Object -First 1
    if ($m) { $ver = $m.Matches[0].Groups[1].Value }
  }
}
if (-not $ver) { $ver = '3.6.10' }
if (-not (Test-Path $out)) { New-Item -ItemType Directory -Path $out -Force | Out-Null }

$stage  = Join-Path $env:TEMP ('anqicms-pack-' + $ver)
$win    = Join-Path $out ('anqicms-v' + $ver + '-windows')
$lin    = Join-Path $out ('anqicms-v' + $ver + '-linux-amd64')
$linArm = Join-Path $out ('anqicms-v' + $ver + '-linux-arm64')
$go     = 'C:\Go\bin\go.exe'
$relDir = Join-Path $root 'deploy\release'

function Assert-Ok {
  param([string]$What)
  if ($LASTEXITCODE -ne 0) { throw ($What + ' failed (exit=' + $LASTEXITCODE + ')') }
}

Remove-Item -Recurse -Force $stage -ErrorAction SilentlyContinue
$stageAmd = Join-Path $stage 'linux-amd64'
$stageArm = Join-Path $stage 'linux-arm64'
New-Item -ItemType Directory -Path $stage, $stageAmd, $stageArm -Force | Out-Null

# ---------------------------------------------------------------- binaries
Write-Host '== build binaries =='
Push-Location $root
try {
  # Windows 版带程序图标：资源文件必须在仓库根目录参与链接。
  # 打完立刻删掉，否则下面交叉编译 linux 会报「无法识别的目标文件」。
  Copy-Item (Join-Path $root 'source\anqicms_syso') (Join-Path $root 'anqicms.syso') -Force
  & $go build -trimpath -ldflags '-w -s' -o (Join-Path $stage 'anqicms.exe') ./main
  $rc = $LASTEXITCODE
  Remove-Item (Join-Path $root 'anqicms.syso') -Force -ErrorAction SilentlyContinue
  if ($rc -ne 0) { throw ('windows build failed (exit=' + $rc + ')') }

  $env:GOOS = 'linux'; $env:CGO_ENABLED = '0'
  $env:GOARCH = 'amd64'
  & $go build -trimpath -ldflags '-w -s' -o (Join-Path $stageAmd 'anqicms') ./main
  Assert-Ok 'linux/amd64 build'
  if (-not $NoArm64) {
    $env:GOARCH = 'arm64'
    & $go build -trimpath -ldflags '-w -s' -o (Join-Path $stageArm 'anqicms') ./main
    Assert-Ok 'linux/arm64 build'
  }
  Remove-Item Env:GOOS -ErrorAction SilentlyContinue
  Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
  Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
} finally {
  Remove-Item (Join-Path $root 'anqicms.syso') -Force -ErrorAction SilentlyContinue
  Pop-Location
}
Get-ChildItem $stage -File -Recurse -ErrorAction SilentlyContinue | ForEach-Object {
  Write-Host ('  bin  ' + $_.Name + '  ' + [math]::Round($_.Length / 1MB, 1) + ' MB')
}

# ---------------------------------------------------------------- tree filler
# 安装包/升级包共用的程序文件（不含二进制，不含用户数据）
function Copy-AppFiles {
  param([string]$Dir, [string]$OS, [string]$Arch)
  Copy-Item -Recurse (Join-Path $root 'template')    $Dir
  Copy-Item -Recurse (Join-Path $root 'system')      $Dir
  Copy-Item -Recurse (Join-Path $root 'locales')     $Dir
  Copy-Item -Recurse (Join-Path $root 'clientFiles') $Dir
  # public 只带静态资源与 xsl；线上生成的 sitemap-*.xml 与用户上传的 uploads 不要
  New-Item -ItemType Directory -Path (Join-Path $Dir 'public\static') -Force | Out-Null
  Copy-Item -Recurse (Join-Path $root 'public\static\default') (Join-Path $Dir 'public\static')
  # 商家工作台（/business）：程序自带的静态页面，controller/business.go 从 public/business 读盘，
  # 不带就会 404。注意后台升级逻辑会跳过整个 public/，所以它只随完整安装包分发。
  if (Test-Path (Join-Path $root 'public\business')) {
    Copy-Item -Recurse (Join-Path $root 'public\business') (Join-Path $Dir 'public')
  }
  Copy-Item (Join-Path $root 'public\*.xsl') (Join-Path $Dir 'public') -ErrorAction SilentlyContinue
  New-Item -ItemType Directory -Path (Join-Path $Dir 'public\uploads') -Force | Out-Null
  foreach ($f in @('dictionary.txt', 'License', 'README.md', 'CHANGELOG.md', 'config.sample.json')) {
    Copy-Item (Join-Path $root $f) $Dir -Force
  }
  Move-Item (Join-Path $Dir 'config.sample.json') (Join-Path $Dir 'config.example.json') -Force
  if (Test-Path (Join-Path $root 'doc')) { Copy-Item -Recurse (Join-Path $root 'doc') $Dir }
  New-Item -ItemType Directory -Path (Join-Path $Dir 'source') -Force | Out-Null
  if ($OS -eq 'windows') {
    Copy-Item (Join-Path $root ('source\cwebp_windows_' + $Arch + '.exe')) (Join-Path $Dir 'source')
  } else {
    Copy-Item (Join-Path $root ('source\cwebp_' + $OS + '_' + $Arch)) (Join-Path $Dir 'source')
  }
  foreach ($d in @('data', 'logs', 'cache')) {
    New-Item -ItemType Directory -Path (Join-Path $Dir $d) -Force | Out-Null
  }
}

# ---------------------------------------------------------------- windows tree
Write-Host '== windows installer tree =='
Remove-Item -Recurse -Force $win -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $win | Out-Null
Copy-Item (Join-Path $stage 'anqicms.exe') $win
Copy-AppFiles -Dir $win -OS 'windows' -Arch 'amd64'
Get-ChildItem (Join-Path $root 'deploy\windows') -File | ForEach-Object {
  Copy-Item $_.FullName $win -Force
}
[IO.File]::WriteAllText((Join-Path $win 'VERSION.txt'), ('anqicms ' + $ver + "`r`n"), (New-Object Text.UTF8Encoding($false)))

# ---------------------------------------------------------------- linux trees
Write-Host '== linux installer tree =='
function New-LinuxTree {
  param([string]$Dir, [string]$Arch, [string]$BinDir)
  Remove-Item -Recurse -Force $Dir -ErrorAction SilentlyContinue
  New-Item -ItemType Directory -Path $Dir | Out-Null
  Copy-Item (Join-Path $BinDir 'anqicms') $Dir
  Copy-AppFiles -Dir $Dir -OS 'linux' -Arch $Arch
  # deploy 只带脚本与服务单元（release 子目录是打包器本身，windows 是 bat，都不进 linux 包）
  robocopy (Join-Path $root 'deploy') (Join-Path $Dir 'deploy') /E /XD release windows /NFL /NDL /NJH /NJS /NP | Out-Null
  Copy-Item (Join-Path $root 'deploy\install.sh')   $Dir -Force
  Copy-Item (Join-Path $root 'deploy\uninstall.sh') $Dir -Force
  [IO.File]::WriteAllText((Join-Path $Dir 'VERSION.txt'), ('anqicms ' + $ver + "`n"), (New-Object Text.UTF8Encoding($false)))
  [IO.File]::WriteAllText((Join-Path $Dir 'PACKAGE_ARCH'), ($Arch + "`n"), (New-Object Text.UTF8Encoding($false)))
  Write-Host ('  ' + (Split-Path -Leaf $Dir))
}
New-LinuxTree -Dir $lin -Arch 'amd64' -BinDir $stageAmd
if (-not $NoArm64) { New-LinuxTree -Dir $linArm -Arch 'arm64' -BinDir $stageArm }

# ---------------------------------------------------------------- upgrade packages
Write-Host '== upgrade packages =='
$note = ('3.6.10：AI 工具路径门闸 Windows 适配（系统目录保护、盘符归一）；修复多处资源句柄未关闭与统计日志读取锁语义（改读锁）；' +
         '修复模板包重名目录、锚文本替换空指针、站点初始化空指针容错；MCP 意图层域映射补全。3.6.9：新增商家工作台（/business，程序自带静态页面，页面与资源随完整安装包分发）；' +
         '清理未使用的历史函数与死代码约 2400 行（翻译/推送/标签 SQL/旧 AI 技能等已停用实现），统一结构体 tag 与代码缩进。')

$sqlList = @()
$vjBase = [ordered]@{
  version = $ver
  date    = (Get-Date -Format 'yyyy-MM-dd')
  note    = $note
  sql     = $sqlList
}
$vjBaseTxt = ($vjBase | ConvertTo-Json -Depth 5)

$upRoot = Join-Path $env:TEMP ('anqicms-up-' + $ver)
Remove-Item -Recurse -Force $upRoot -ErrorAction SilentlyContinue

function New-UpgradeTree {
  param([string]$Dir, [string]$ExeSrc, [string]$ExeName, [string]$Arch)
  Remove-Item -Recurse -Force $Dir -ErrorAction SilentlyContinue
  New-Item -ItemType Directory -Path $Dir | Out-Null
  Copy-Item $ExeSrc (Join-Path $Dir $ExeName) -Force
  # 升级包只带程序文件：template/ public/ 由后台升级逻辑跳过（用户改过的模板不能被覆盖）
  Copy-Item -Recurse (Join-Path $root 'system')   $Dir
  Copy-Item -Recurse (Join-Path $root 'locales')  $Dir
  Copy-Item -Recurse (Join-Path $root 'clientFiles') $Dir
  Copy-Item (Join-Path $root 'dictionary.txt') $Dir -Force
  Copy-Item (Join-Path $root 'CHANGELOG.md')   $Dir -Force
  New-Item -ItemType Directory -Path (Join-Path $Dir 'source') -Force | Out-Null
  $src = if ($ExeName -like '*.exe') { 'source\cwebp_windows_' + $Arch + '.exe' } else { 'source\cwebp_linux_' + $Arch }
  Copy-Item (Join-Path $root $src) (Join-Path $Dir 'source')
  robocopy (Join-Path $root 'deploy') (Join-Path $Dir 'deploy') /E /XD release windows /NFL /NDL /NJH /NJS /NP | Out-Null
  [IO.File]::WriteAllText((Join-Path $Dir 'version.json'), $vjBaseTxt, (New-Object Text.UTF8Encoding($false)))
}

$upWinStage = Join-Path $upRoot 'windows'
$upLinStage = Join-Path $upRoot 'linux'
New-UpgradeTree -Dir $upWinStage -ExeSrc (Join-Path $stage 'anqicms.exe') -ExeName 'anqicms.exe' -Arch 'amd64'
New-UpgradeTree -Dir $upLinStage -ExeSrc (Join-Path $stageAmd 'anqicms') -ExeName 'anqicms' -Arch 'amd64'

$upWinTmp = Join-Path $env:TEMP ('anqicms-v' + $ver + '-windows-upgrade.zip')
$upLinTmp = Join-Path $env:TEMP ('anqicms-v' + $ver + '-linux-upgrade.zip')
Remove-Item -Force $upWinTmp, $upLinTmp -ErrorAction SilentlyContinue
Push-Location $root
try {
  & $go run ./tools/zipdir -src $upWinStage -out $upWinTmp
  Assert-Ok 'zipdir (windows upgrade)'
  & $go run ./tools/zipdir -src $upLinStage -out $upLinTmp
  Assert-Ok 'zipdir (linux upgrade)'
} finally { Pop-Location }
if (-not (Test-Path $upWinTmp) -or -not (Test-Path $upLinTmp)) { throw '打包升级包失败' }
$upWinSum = (Get-FileHash $upWinTmp -Algorithm SHA256).Hash.ToLower()
$upLinSum = (Get-FileHash $upLinTmp -Algorithm SHA256).Hash.ToLower()
Write-Host ('  windows upgrade sha256=' + $upWinSum)
Write-Host ('  linux   upgrade sha256=' + $upLinSum)

New-Item -ItemType Directory -Path $relDir -Force | Out-Null
Copy-Item $upWinTmp (Join-Path $relDir ('anqicms-v' + $ver + '.zip')) -Force
Copy-Item $upWinTmp (Join-Path $relDir ('anqicms-v' + $ver + '-windows-upgrade.zip')) -Force
Copy-Item $upLinTmp (Join-Path $relDir ('anqicms-v' + $ver + '-linux-upgrade.zip')) -Force
Remove-Item -Recurse -Force $upRoot -ErrorAction SilentlyContinue

# ---------------------------------------------------------------- version.json
# 必须**先写** version.json 再打安装包：安装目录根部要带一份（后台读取当前版本）。
# packages 按平台给包；顶层 url/checksum 指向 Windows 升级包（旧后台只认它）。
Write-Host '== version.json =='
$vj = [ordered]@{
  name     = 'anqicms'
  version  = $ver
  date     = (Get-Date -Format 'yyyy-MM-dd')
  url      = ('anqicms-v' + $ver + '.zip')
  checksum = ('sha256:' + $upWinSum)
  note     = $note
  sql      = $sqlList
  packages = [ordered]@{
    windows = [ordered]@{
      url      = ('anqicms-v' + $ver + '-windows-upgrade.zip')
      checksum = ('sha256:' + $upWinSum)
    }
    linux   = [ordered]@{
      url      = ('anqicms-v' + $ver + '-linux-upgrade.zip')
      checksum = ('sha256:' + $upLinSum)
    }
  }
}
$jtxt = ($vj | ConvertTo-Json -Depth 5)
[IO.File]::WriteAllText((Join-Path $relDir 'version.json'), $jtxt, (New-Object Text.UTF8Encoding($false)))
Copy-Item (Join-Path $relDir 'version.json') (Join-Path $out 'version.json') -Force
Copy-Item (Join-Path $relDir 'version.json') (Join-Path $win 'version.json') -Force
Copy-Item (Join-Path $relDir 'version.json') (Join-Path $lin 'version.json') -Force
if (-not $NoArm64) { Copy-Item (Join-Path $relDir 'version.json') (Join-Path $linArm 'version.json') -Force }
Write-Host '  version.json written (release dir + installer trees)'

# ---------------------------------------------------------------- archives
# 归档统一走 tools/zipdir 与 tools/targz：
#   - zip 内路径必须是正斜杠（Compress-Archive 在 PS5.1 会写反斜杠）
#   - tar.gz 必须带 0755（Windows 上文件没有可执行位，系统 tar 会打成 0644）
Write-Host '== archives =='
$winZip    = Join-Path $out ('anqicms-v' + $ver + '-windows.zip')
$linAmdTgz = Join-Path $out ('anqicms-v' + $ver + '-linux-amd64.tar.gz')
$linArmTgz = Join-Path $out ('anqicms-v' + $ver + '-linux-arm64.tar.gz')
Remove-Item -Force $winZip, $linAmdTgz, $linArmTgz -ErrorAction SilentlyContinue
Push-Location $root
try {
  & $go run ./tools/zipdir -src $win -out $winZip
  Assert-Ok 'zipdir (windows installer)'
  & $go run ./tools/targz -src $lin -out $linAmdTgz
  Assert-Ok 'targz (linux amd64)'
  if (-not $NoArm64) {
    & $go run ./tools/targz -src $linArm -out $linArmTgz
    Assert-Ok 'targz (linux arm64)'
  }
} finally { Pop-Location }
if (-not (Test-Path $winZip) -or -not (Test-Path $linAmdTgz)) { throw '打包安装包失败' }

$upZip  = Join-Path $out ('anqicms-v' + $ver + '.zip')
$upWZip = Join-Path $out ('anqicms-v' + $ver + '-windows-upgrade.zip')
$upLZip = Join-Path $out ('anqicms-v' + $ver + '-linux-upgrade.zip')
Copy-Item $upWinTmp $upZip  -Force
Copy-Item $upWinTmp $upWZip -Force
Copy-Item $upLinTmp $upLZip -Force

# ---------------------------------------------------------------- install manifest + one-liner
# 安装包也在 version.json 里登记（install.*）：在线安装脚本照着下载 + 校验 sha256。
# ⚠️ install 用**扁平键**（linux_amd64_file 这种）：install.sh 里没有 jq，
# 只能靠 sed 在压缩成一行的 JSON 上取字段；嵌套对象里夹着花括号，正则很难写对。
$winSum    = (Get-FileHash $winZip    -Algorithm SHA256).Hash.ToLower()
$linAmdSum = (Get-FileHash $linAmdTgz -Algorithm SHA256).Hash.ToLower()
$vj['install'] = [ordered]@{
  'windows_file'       = (Split-Path -Leaf $winZip)
  'windows_sha256'     = $winSum
  'linux_amd64_file'   = (Split-Path -Leaf $linAmdTgz)
  'linux_amd64_sha256' = $linAmdSum
}
if (-not $NoArm64) {
  $linArmSum = (Get-FileHash $linArmTgz -Algorithm SHA256).Hash.ToLower()
  $vj['install']['linux_arm64_file']   = (Split-Path -Leaf $linArmTgz)
  $vj['install']['linux_arm64_sha256'] = $linArmSum
}
$jtxt = ($vj | ConvertTo-Json -Depth 6)
[IO.File]::WriteAllText((Join-Path $relDir 'version.json'), $jtxt, (New-Object Text.UTF8Encoding($false)))
Copy-Item (Join-Path $relDir 'version.json') (Join-Path $out 'version.json') -Force
Write-Host ('  version.json +install.* (' + (Get-Item (Join-Path $out 'version.json')).Length + ' bytes)')

# 在线一键安装：把 deploy\online-install.sh 里的 __DEFAULT_MIRROR__ 换成发布镜像
$online = [IO.File]::ReadAllText((Join-Path $root 'deploy\online-install.sh'))
$online = $online.Replace('__DEFAULT_MIRROR__', $mirror.TrimEnd('/'))
[IO.File]::WriteAllText((Join-Path $out 'install.sh'), $online, (New-Object Text.UTF8Encoding($false)))
Write-Host ('  install.sh written (mirror=' + $mirror + ')')

# 在线卸载：deploy\uninstall.sh 原样放到发布目录
$uninst = [IO.File]::ReadAllText((Join-Path $root 'deploy\uninstall.sh'))
[IO.File]::WriteAllText((Join-Path $out 'uninstall.sh'), $uninst, (New-Object Text.UTF8Encoding($false)))
Write-Host '  uninstall.sh written'

Write-Host '== checksums =='
$pkgList = @($upZip, $upWZip, $upLZip, $winZip, $linAmdTgz)
if (-not $NoArm64) { $pkgList += $linArmTgz }
$lines = @()
foreach ($f in $pkgList) {
  $h  = (Get-FileHash $f -Algorithm SHA256).Hash.ToLower()
  $sz = (Get-Item $f).Length
  $lines += ($h + '  ' + (Split-Path -Leaf $f) + '  ' + $sz)
  Write-Host ('  ' + (Split-Path -Leaf $f) + '  ' + [math]::Round($sz / 1MB, 2) + ' MB')
}
# ⚠️ 数组元素里带 `+` 拼接的**必须各自加括号**，否则 PS5.1 会按 $OFS 把后面元素串进来
$hdr = @(
  ('# anqicms ' + $ver + ' release manifest'),
  ('# generated: ' + (Get-Date -Format 'yyyy-MM-dd HH:mm:ss')),
  '# format: sha256  file  bytes',
  '#',
  ('# anqicms-v' + $ver + '-windows-upgrade.zip  upgrade package for Windows (admin > upload upgrade package)'),
  ('# anqicms-v' + $ver + '-linux-upgrade.zip    upgrade package for Linux   (same, packages.linux in version.json)'),
  ('# anqicms-v' + $ver + '.zip                  legacy single upgrade package (= windows-upgrade)'),
  ('# anqicms-v' + $ver + '-windows.zip          full installer for Windows (unzip and run, built-in /install wizard)'),
  ('# anqicms-v' + $ver + '-linux-amd64.tar.gz   full installer for Linux x86_64 (install.sh + systemd)'),
  ('# anqicms-v' + $ver + '-linux-arm64.tar.gz   full installer for Linux arm64'),
  ''
)
[IO.File]::WriteAllLines((Join-Path $out 'SHA256SUMS.txt'), ($hdr + $lines), (New-Object Text.UTF8Encoding($false)))
Remove-Item -Recurse -Force $stage -ErrorAction SilentlyContinue
Write-Host ''
Write-Host ('done -> ' + $out)

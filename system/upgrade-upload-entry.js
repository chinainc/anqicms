/**
 * 后台「系统升级」页内「上传升级包」卡片注入脚本
 *
 * 背景：后台前端是 umi/antd 打包出来的 SPA，菜单与页面都是编译产物，
 * 直接改源码不现实。所以这里做成一枚运行时注入卡片：
 *   1. 只在 /system/tool/upgrade（系统升级）这一页注入，插在「当前版本」卡片之后；
 *   2. 卡片样式复用 antd 的类名 + 少量自绘，视觉与原生页面一致；
 *   3. React 重新渲染可能抹掉注入节点，用 MutationObserver 兜底补回；
 *   4. 登录态沿用后台下发的 admin_token cookie（middleware 已支持 cookie 兜底）。
 *
 * 上传逻辑对应后端 /system/api/version/upload-upgrade（controller/manageController/upgrade.go），
 * 纯本地上传升级包，服务器不需要访问外网。
 * 独立页 /system/upgrade-upload.html 保留为同功能的直接入口（不在菜单里）。
 */
(function () {
    'use strict';

    var CARD_ID = 'anqicms-upgrade-upload-card';
    var STYLE_ID = 'anqicms-upgrade-upload-style';
    // 后台 SPA 的历史路由：/system/tool/upgrade
    var ROUTE_SUFFIX = '/tool/upgrade';

    var state = {
        envLoaded: false,
        busy: false
    };

    function isUpgradePage() {
        var p = window.location.pathname.replace(/\/+$/, '');
        return p.indexOf('/system') === 0 && p.slice(-ROUTE_SUFFIX.length) === ROUTE_SUFFIX;
    }

    function injectStyle() {
        if (document.getElementById(STYLE_ID)) {
            return;
        }
        var style = document.createElement('style');
        style.id = STYLE_ID;
        // antd v5 的样式是 CSS-in-JS（类名带 hash），光写 ant-btn/ant-card 不会有样式，
        // 所以卡片与控件全部自带样式，避免依赖前端构建产物。
        style.textContent = [
            '.anqicms-up-card{margin-top:16px;padding:20px 24px;background:#fff;border:1px solid #f0f0f0;border-radius:8px;font-size:14px;color:rgba(0,0,0,.88)}',
            '.anqicms-up-card .up-btn{height:32px;padding:0 15px;border:1px solid #d9d9d9;border-radius:6px;background:#fff;color:rgba(0,0,0,.88);font-size:14px;font-family:inherit;line-height:1.6;cursor:pointer;transition:all .2s}',
            '.anqicms-up-card .up-btn:hover{border-color:#4096ff;color:#4096ff}',
            '.anqicms-up-card .up-btn-primary{background:#1677ff;border-color:#1677ff;color:#fff}',
            '.anqicms-up-card .up-btn-primary:hover{background:#4096ff;border-color:#4096ff;color:#fff}',
            '.anqicms-up-card .up-btn[disabled]{opacity:.5;cursor:not-allowed;border-color:#d9d9d9;color:rgba(0,0,0,.25);background:#fff}',
            '.anqicms-up-card .up-btn-primary[disabled]{background:#1677ff;color:#fff}',
            '.anqicms-up-card .up-input{height:32px;width:180px;padding:4px 11px;border:1px solid #d9d9d9;border-radius:6px;font-size:14px;color:rgba(0,0,0,.88);outline:none;transition:all .2s}',
            '.anqicms-up-card .up-input:focus{border-color:#1677ff;box-shadow:0 0 0 2px rgba(5,145,255,.1)}',
            '.anqicms-up-card .up-head{display:flex;align-items:center;gap:8px;margin:0 0 8px}',
            '.anqicms-up-card .up-head h4{margin:0;font-size:15px;font-weight:600;color:#1f2329}',
            '.anqicms-up-card .up-head::before{content:"";width:3px;height:14px;border-radius:2px;background:#1677ff}',
            '.anqicms-up-card .up-meta{margin-left:auto;font-size:12px;color:#86909c}',
            '.anqicms-up-card .up-hint{margin:0 0 14px;font-size:13px;color:#5c6b7a}',
            '.anqicms-up-card .up-row{display:flex;align-items:center;gap:10px;flex-wrap:wrap;margin-bottom:12px}',
            '.anqicms-up-card .up-file{font-size:13px;color:#86909c}',
            '.anqicms-up-card .up-check{display:inline-flex;align-items:center;gap:6px;font-size:13px;color:#5c6b7a;cursor:pointer}',
            '.anqicms-up-card .up-label{font-size:13px;color:#5c6b7a}',
            '.anqicms-up-card .up-inline-hint{font-size:12px;color:#86909c}',
            '.anqicms-up-card .up-progress{display:none;height:8px;border-radius:4px;background:#eef1f5;overflow:hidden;margin:0 0 10px}',
            '.anqicms-up-card .up-progress.is-show{display:block}',
            '.anqicms-up-card .up-progress span{display:block;height:100%;width:0;background:#1677ff;transition:width .2s}',
            '.anqicms-up-card .up-result{display:none;margin:0 0 10px;padding:10px 12px;border-radius:6px;font-size:13px;white-space:pre-wrap;word-break:break-all}',
            '.anqicms-up-card .up-result.is-show{display:block}',
            '.anqicms-up-card .up-result.is-ok{background:#e8f7ee;color:#12703b;border:1px solid #b7e4c7}',
            '.anqicms-up-card .up-result.is-err{background:#fdecec;color:#a8071a;border:1px solid #ffc9c9}',
            '.anqicms-up-card .up-tips{margin:6px 0 0;padding-left:18px;font-size:12px;line-height:1.9;color:#86909c}'
        ].join('');
        document.head.appendChild(style);
    }

    function buildCard() {
        var card = document.createElement('div');
        card.id = CARD_ID;
        card.className = 'anqicms-up-card';
        card.innerHTML = [
            '<div class="up-head"><h4>方式二：上传升级包</h4><span class="up-meta" id="up-meta"></span></div>',
            '<p class="up-hint">把打好的升级包（<b>.zip</b> 或 <b>.tar.gz</b>）直接上传到服务器升级，服务器不需要访问外网，也不用先传到别的地方。</p>',
            '<div class="up-row">',
            '  <button type="button" class="up-btn" id="up-pick">选择文件</button>',
            '  <span class="up-file" id="up-file">未选择文件</span>',
            '  <button type="button" class="up-btn up-btn-primary" id="up-do" disabled>上传并升级</button>',
            '  <label class="up-check"><input type="checkbox" id="up-force"> 强制升级（即使版本号相同）</label>',
            '  <input type="file" id="up-input" accept=".zip,.gz,.tar.gz" hidden>',
            '</div>',
            '<div class="up-row">',
            '  <span class="up-label">版本号</span>',
            '  <input type="text" class="up-input" id="up-version" placeholder="版本号（可选）">',
            '  <span class="up-inline-hint">包内有 version.json 时自动取版本号与 SQL 清单，无需填写</span>',
            '</div>',
            '<div class="up-progress" id="up-progress"><span id="up-bar"></span></div>',
            '<div class="up-result" id="up-result"></div>',
            '<ul class="up-tips">',
            '  <li>升级包上限 512 MB；解压后总量上限 2 GB。</li>',
            '  <li>不会覆盖：template/、public/、data/、uploads/、logs/、config.json、*.sh、*.bat。</li>',
            '  <li>升级完成后需要重启服务才生效，重启期间站点会有几秒钟不可用。</li>',
            '</ul>'
        ].join('');

        bindCard(card);

        return card;
    }

    function bindCard(card) {
        var input = card.querySelector('#up-input');
        var fileName = card.querySelector('#up-file');
        var pickBtn = card.querySelector('#up-pick');
        var doBtn = card.querySelector('#up-do');
        var forceBox = card.querySelector('#up-force');
        var versionInput = card.querySelector('#up-version');
        var progress = card.querySelector('#up-progress');
        var bar = card.querySelector('#up-bar');
        var resultBox = card.querySelector('#up-result');

        function showResult(ok, message, withRestart) {
            resultBox.className = 'up-result is-show ' + (ok ? 'is-ok' : 'is-err');
            resultBox.textContent = message;
            if (!withRestart) {
                return;
            }
            var btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'up-btn up-btn-primary';
            btn.style.marginTop = '8px';
            btn.textContent = '立即重启服务';
            btn.addEventListener('click', function () {
                btn.disabled = true;
                btn.textContent = '正在重启...';
                fetch('/system/api/anqi/restart', {
                    method: 'POST',
                    credentials: 'same-origin',
                    headers: { 'Content-Type': 'application/json' },
                    body: '{}'
                }).then(function () {
                    btn.textContent = '已发送重启指令，5 秒后刷新';
                    setTimeout(function () { window.location.reload(); }, 5000);
                }).catch(function () {
                    btn.disabled = false;
                    btn.textContent = '重启失败，请手动重启';
                });
            });
            resultBox.appendChild(document.createElement('br'));
            resultBox.appendChild(btn);
        }

        function formatSize(size) {
            if (size >= 1024 * 1024) {
                return (size / 1024 / 1024).toFixed(1) + ' MB';
            }
            if (size >= 1024) {
                return Math.round(size / 1024) + ' KB';
            }
            return size + ' B';
        }

        pickBtn.addEventListener('click', function () {
            input.click();
        });

        input.addEventListener('change', function () {
            if (input.files && input.files[0]) {
                fileName.textContent = input.files[0].name + '（' + formatSize(input.files[0].size) + '）';
                doBtn.disabled = false;
            } else {
                fileName.textContent = '未选择文件';
                doBtn.disabled = true;
            }
        });

        doBtn.addEventListener('click', function () {
            if (state.busy || !input.files || !input.files[0]) {
                return;
            }
            var file = input.files[0];
            if (file.size > 512 * 1024 * 1024) {
                showResult(false, '升级包超过 512 MB，请拆分后重试。');
                return;
            }
            if (!window.confirm('确定使用「' + file.name + '」升级？升级会覆盖程序文件。')) {
                return;
            }

            var form = new FormData();
            form.append('file', file);
            if (versionInput.value.trim() !== '') {
                form.append('version', versionInput.value.trim());
            }
            if (forceBox.checked) {
                form.append('force', '1');
            }

            state.busy = true;
            doBtn.disabled = true;
            pickBtn.disabled = true;
            progress.className = 'up-progress is-show';
            bar.style.width = '0%';
            resultBox.className = 'up-result';

            var xhr = new XMLHttpRequest();
            xhr.open('POST', '/system/api/version/upload-upgrade', true);
            xhr.withCredentials = true;
            xhr.upload.onprogress = function (e) {
                if (e.lengthComputable) {
                    var percent = Math.floor(e.loaded / e.total * 100);
                    bar.style.width = percent + '%';
                    doBtn.textContent = '上传中 ' + percent + '%';
                }
            };
            xhr.onload = function () {
                state.busy = false;
                progress.className = 'up-progress';
                doBtn.textContent = '上传并升级';
                doBtn.disabled = false;
                pickBtn.disabled = false;

                var res = {};
                try {
                    res = JSON.parse(xhr.responseText);
                } catch (e) {
                    res = { code: -1, msg: '服务端返回异常：' + String(xhr.responseText).slice(0, 200) };
                }
                if (res.code === 0) {
                    showResult(true, res.msg || '升级完成', true);
                    input.value = '';
                    fileName.textContent = '未选择文件';
                    versionInput.value = '';
                    doBtn.disabled = true;
                } else {
                    showResult(false, res.msg || '升级失败');
                }
            };
            xhr.onerror = function () {
                state.busy = false;
                progress.className = 'up-progress';
                doBtn.textContent = '上传并升级';
                doBtn.disabled = false;
                pickBtn.disabled = false;
                showResult(false, '上传失败，请检查网络或升级包大小是否超限');
            };
            xhr.send(form);
        });

        // 环境信息（主程序名/平台），失败不影响上传
        if (!state.envLoaded) {
            state.envLoaded = true;
            fetch('/system/api/version/env', { credentials: 'same-origin' })
                .then(function (res) { return res.json(); })
                .then(function (res) {
                    var data = (res && res.data) || {};
                    var meta = card.querySelector('#up-meta');
                    if (!meta) {
                        return;
                    }
                    var parts = [];
                    if (data.exec) {
                        parts.push('主程序 ' + data.exec);
                    }
                    if (data.goos) {
                        parts.push(data.goos + '/' + data.goarch);
                    }
                    meta.textContent = parts.join(' · ');
                })
                .catch(function () {});
        }
    }

    // 找到「当前版本」所在内容容器，把卡片插到它后面
    function tryInject() {
        if (!isUpgradePage() || document.getElementById(CARD_ID)) {
            return;
        }
        var container = document.querySelector('.ant-pro-page-container-children-container');
        if (!container) {
            return;
        }
        var exists = container.querySelector('.ant-card');
        if (!exists && container.textContent.indexOf('当前版本') < 0) {
            // 页面还没渲染完，等下一次观察回调
            return;
        }
        injectStyle();
        var card = buildCard();
        if (exists && exists.parentNode) {
            exists.parentNode.insertBefore(card, exists.nextSibling);
        } else {
            container.appendChild(card);
        }
    }

    function watch() {
        tryInject();
        var observer = new MutationObserver(function () {
            tryInject();
        });
        observer.observe(document.body, { childList: true, subtree: true });
        // 路由切换是 pushState，MutationObserver 一般能覆盖，这里再兜一次
        window.addEventListener('popstate', function () {
            setTimeout(tryInject, 300);
        });
        setInterval(tryInject, 1500);
    }

    function boot() {
        if (window.location.pathname.indexOf('/system') !== 0) {
            return;
        }
        // 登录页不注入
        if (/\/system\/(login|install)/.test(window.location.pathname)) {
            return;
        }
        watch();
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', boot);
    } else {
        boot();
    }
})();

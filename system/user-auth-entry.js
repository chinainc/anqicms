/**
 * 后台「认证管理」入口注入脚本
 *
 * 后台前端是 umi/antd 打包出来的 SPA，菜单和路由都在编译产物里，
 * 无法直接加页面。所以这里在右下角放一个入口按钮（带待审核数量角标），
 * 点击后用全屏 iframe 打开独立审核页 /system/user-auth.html。
 *
 * 登录态用后台下发的 admin_token cookie（middleware 已支持 cookie 兜底）。
 */
(function () {
    'use strict';

    var BTN_ID = 'anqicms-user-auth-btn';
    var MASK_ID = 'anqicms-user-auth-mask';
    var PAGE_URL = '/system/user-auth.html';
    var POLL_MS = 60000;

    var badgeEl = null;

    function isAdminPage() {
        return window.location.pathname.indexOf('/system') === 0
            && !/\/system\/(login|install)/.test(window.location.pathname)
            && !/\/system\/user-auth/.test(window.location.pathname);
    }

    function createButton() {
        var btn = document.createElement('button');
        btn.id = BTN_ID;
        btn.type = 'button';
        btn.title = '认证管理：审核会员实名 / 企业认证';
        btn.innerHTML = '<span style="font-size:15px;line-height:1">&#128737;</span><span>认证审核</span><b class="anqicms-auth-badge" style="display:none;min-width:18px;height:18px;padding:0 5px;border-radius:9px;background:#e5484d;color:#fff;font-size:11px;font-weight:600;line-height:18px;text-align:center"></b>';
        btn.style.cssText = [
            'position:fixed',
            'right:24px',
            'bottom:76px',
            'z-index:9998',
            'display:flex',
            'align-items:center',
            'gap:6px',
            'height:40px',
            'padding:0 16px',
            'border:0',
            'border-radius:20px',
            'background:#1a9e59',
            'color:#fff',
            'font-size:13px',
            'font-family:inherit',
            'cursor:pointer',
            'box-shadow:0 4px 14px rgba(26,158,89,.35)',
            'transition:transform .15s, box-shadow .15s'
        ].join(';');
        btn.addEventListener('mouseenter', function () {
            btn.style.transform = 'translateY(-2px)';
            btn.style.boxShadow = '0 8px 20px rgba(26,158,89,.42)';
        });
        btn.addEventListener('mouseleave', function () {
            btn.style.transform = 'none';
            btn.style.boxShadow = '0 4px 14px rgba(26,158,89,.35)';
        });
        btn.addEventListener('click', openPanel);
        badgeEl = btn.querySelector('.anqicms-auth-badge');

        return btn;
    }

    function openPanel() {
        var existed = document.getElementById(MASK_ID);
        if (existed) {
            existed.style.display = 'flex';
            var frame = existed.querySelector('iframe');
            if (frame) {
                frame.src = PAGE_URL;
            }
            return;
        }
        var mask = document.createElement('div');
        mask.id = MASK_ID;
        mask.style.cssText = [
            'position:fixed',
            'inset:0',
            'z-index:9999',
            'display:flex',
            'align-items:center',
            'justify-content:center',
            'background:rgba(15,23,32,.45)',
            'padding:32px 24px'
        ].join(';');

        var panel = document.createElement('div');
        panel.style.cssText = [
            'position:relative',
            'width:min(1100px,100%)',
            'height:min(820px,100%)',
            'background:#f5f7fa',
            'border-radius:10px',
            'overflow:hidden',
            'box-shadow:0 18px 48px rgba(0,0,0,.24)'
        ].join(';');

        var close = document.createElement('button');
        close.type = 'button';
        close.innerHTML = '&times;';
        close.title = '关闭';
        close.style.cssText = [
            'position:absolute',
            'right:10px',
            'top:8px',
            'z-index:2',
            'width:28px',
            'height:28px',
            'border:0',
            'border-radius:50%',
            'background:rgba(0,0,0,.06)',
            'color:#4a5764',
            'font-size:20px',
            'line-height:1',
            'cursor:pointer'
        ].join(';');
        close.addEventListener('click', function () {
            mask.style.display = 'none';
        });

        var frame = document.createElement('iframe');
        frame.src = PAGE_URL;
        frame.style.cssText = 'width:100%;height:100%;border:0;background:#f5f7fa';

        panel.appendChild(close);
        panel.appendChild(frame);
        mask.appendChild(panel);
        mask.addEventListener('click', function (e) {
            if (e.target === mask) {
                mask.style.display = 'none';
            }
        });
        document.body.appendChild(mask);
    }

    // 拉一次待审核数量，显示角标
    function refreshBadge() {
        fetch('/system/api/plugin/userauth/list?status=0&page=1&pageSize=1', {
            credentials: 'same-origin'
        }).then(function (res) {
            return res.json();
        }).then(function (res) {
            if (!res || res.code !== 0 || !badgeEl) {
                return;
            }
            var pending = (res.data && res.data.pending) || 0;
            if (pending > 0) {
                badgeEl.textContent = pending > 99 ? '99+' : String(pending);
                badgeEl.style.display = 'inline-block';
            } else {
                badgeEl.style.display = 'none';
            }
        }).catch(function () {});
    }

    function boot() {
        if (!isAdminPage()) {
            return;
        }
        if (document.getElementById(BTN_ID)) {
            refreshBadge();
            return;
        }
        document.body.appendChild(createButton());
        refreshBadge();
        setInterval(refreshBadge, POLL_MS);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', boot);
    } else {
        boot();
    }
})();

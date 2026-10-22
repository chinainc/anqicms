/**
 * 后台「可视化装修」入口注入脚本。
 *
 * 后台前端是 umi 打包产物，加不了路由，所以在右下角放一个入口按钮，
 * 点击用全屏面板打开独立装修页 /system/diy.html（该页由 AdminFileServ 直接提供）。
 */
(function () {
    'use strict';

    var BTN_ID = 'anqicms-diy-btn';
    var MASK_ID = 'anqicms-diy-mask';
    var PAGE_URL = '/system/diy.html';

    function isAdminPage() {
        return window.location.pathname.indexOf('/system') === 0
            && !/\/system\/(login|install)/.test(window.location.pathname)
            && !/\/system\/diy/.test(window.location.pathname);
    }

    function openPanel() {
        var existed = document.getElementById(MASK_ID);
        if (existed) {
            existed.style.display = 'flex';
            return;
        }
        var mask = document.createElement('div');
        mask.id = MASK_ID;
        mask.style.cssText = [
            'position:fixed', 'inset:0', 'z-index:9999',
            'display:flex', 'align-items:center', 'justify-content:center',
            'background:rgba(15,23,32,.45)', 'padding:28px 20px'
        ].join(';');

        var panel = document.createElement('div');
        panel.style.cssText = [
            'position:relative', 'width:min(1280px,100%)', 'height:min(860px,100%)',
            'background:#f5f7fa', 'border-radius:10px', 'overflow:hidden',
            'box-shadow:0 18px 48px rgba(0,0,0,.24)'
        ].join(';');

        var close = document.createElement('button');
        close.type = 'button';
        close.innerHTML = '&times;';
        close.title = '关闭';
        close.style.cssText = [
            'position:absolute', 'right:10px', 'top:8px', 'z-index:2',
            'width:28px', 'height:28px', 'border:0', 'border-radius:50%',
            'background:rgba(0,0,0,.06)', 'color:#4a5764', 'font-size:20px',
            'line-height:1', 'cursor:pointer'
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

    function createButton() {
        var btn = document.createElement('button');
        btn.id = BTN_ID;
        btn.type = 'button';
        btn.title = '可视化装修：拖拽区块、实时预览首页';
        btn.innerHTML = '<span style="font-size:15px;line-height:1">&#129513;</span><span>可视化装修</span>';
        btn.style.cssText = [
            'position:fixed', 'right:24px', 'bottom:126px', 'z-index:9998',
            'display:flex', 'align-items:center', 'gap:6px',
            'height:40px', 'padding:0 16px',
            'border:0', 'border-radius:20px',
            'background:#1677ff', 'color:#fff',
            'font-size:13px', 'font-family:inherit', 'cursor:pointer',
            'box-shadow:0 4px 14px rgba(22,119,255,.35)',
            'transition:transform .15s, box-shadow .15s'
        ].join(';');
        btn.addEventListener('mouseenter', function () {
            btn.style.transform = 'translateY(-2px)';
            btn.style.boxShadow = '0 8px 20px rgba(22,119,255,.42)';
        });
        btn.addEventListener('mouseleave', function () {
            btn.style.transform = 'none';
            btn.style.boxShadow = '0 4px 14px rgba(22,119,255,.35)';
        });
        btn.addEventListener('click', openPanel);
        return btn;
    }

    function boot() {
        if (!isAdminPage() || document.getElementById(BTN_ID)) {
            return;
        }
        document.body.appendChild(createButton());
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', boot);
    } else {
        boot();
    }
})();

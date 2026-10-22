/**
 * 前台商城交互：加入购物车、收藏
 * 按钮约定：
 *   <button class="js-add-cart" data-id="内容ID">
 *   <button class="js-favorite" data-id="内容ID">
 */
(function (window, document) {
    var toastEl = null;
    var toastTimer = null;

    function toast(msg) {
        if (!msg) {
            return;
        }
        if (!toastEl) {
            toastEl = document.createElement('div');
            toastEl.style.cssText = 'position:fixed;left:50%;top:18%;transform:translateX(-50%);z-index:9999;' +
                'padding:10px 20px;border-radius:6px;background:rgba(31,45,61,.92);color:#fff;font-size:14px;' +
                'line-height:1.5;max-width:80%;transition:opacity .25s;opacity:0;';
            document.body.appendChild(toastEl);
        }
        toastEl.textContent = msg;
        toastEl.style.opacity = '1';
        if (toastTimer) {
            clearTimeout(toastTimer);
        }
        toastTimer = setTimeout(function () {
            toastEl.style.opacity = '0';
        }, 2200);
    }

    function post(url, body) {
        return fetch(url, {
            method: 'POST',
            credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body || {})
        }).then(function (res) {
            return res.text();
        }).then(function (text) {
            var data = {};
            try {
                data = text ? JSON.parse(text) : {};
            } catch (e) {
                data = { code: -1, msg: '' };
            }
            if (data.code === 0) {
                return data;
            }
            throw new Error(data.msg || '操作失败');
        });
    }

    function bind(selector, handler) {
        var list = document.querySelectorAll(selector);
        Array.prototype.forEach.call(list, function (btn) {
            btn.addEventListener('click', function () {
                var id = parseInt(btn.getAttribute('data-id'), 10) || 0;
                if (!id) {
                    return;
                }
                btn.disabled = true;
                handler(btn, id);
            });
        });
    }

    bind('.js-add-cart', function (btn, id) {
        post('/api/cart/add', { archive_id: id, quantity: 1 }).then(function (res) {
            btn.disabled = false;
            toast('已加入购物车');
            var badge = document.querySelector('.top-cart');
            if (badge && res.data && typeof res.data.count !== 'undefined') {
                badge.setAttribute('data-count', res.data.count);
            }
        }).catch(function (err) {
            btn.disabled = false;
            toast(err.message);
        });
    });

    bind('.js-favorite', function (btn, id) {
        post('/api/favorite/add', { archive_id: id }).then(function () {
            btn.disabled = false;
            btn.classList.add('is-active');
            toast('已收藏');
        }).catch(function (err) {
            btn.disabled = false;
            toast(err.message);
        });
    });

    window.Shop = { toast: toast, post: post };
})(window, document);

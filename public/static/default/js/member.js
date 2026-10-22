/**
 * 会员中心 / 登录注册页公用脚本
 * - Member.request(url, {method, body})  统一请求 /api/*，按 {code,msg,data} 协议处理
 * - Member.toast(msg)                    轻提示
 * 页面可在引入本文件之前用 window.MEMBER_I18N 覆盖文案
 */
(function (window, document) {
    var defaults = {
        saveSuccess: '保存成功',
        uploadSuccess: '上传成功',
        passwordsNotMatch: '两次输入的密码不一致',
        passwordTooShort: '密码长度不能少于 6 位',
        enterAccountAndPassword: '请输入账号和密码',
        enterUserName: '请输入用户名',
        enterEmail: '请输入邮箱',
        enterEmailOrPhone: '请填写邮箱或手机号',
        registerSuccess: '注册成功',
        resetEmailSent: '重置邮件已发送，请查收邮件',
        resetPasswordSuccess: '密码重置成功，请重新登录',
        confirmRemove: '确定要移除吗？',
        requestFailed: '请求失败，请稍后再试',
        notLoggedIn: '登录状态已失效，请重新登录'
    };

    var dict = {};
    var key;
    for (key in defaults) {
        if (Object.prototype.hasOwnProperty.call(defaults, key)) {
            dict[key] = defaults[key];
        }
    }
    if (window.MEMBER_I18N) {
        for (key in window.MEMBER_I18N) {
            if (Object.prototype.hasOwnProperty.call(window.MEMBER_I18N, key) && window.MEMBER_I18N[key]) {
                dict[key] = window.MEMBER_I18N[key];
            }
        }
    }
    window.MEMBER_I18N = dict;

    var toastTimer = null;

    function toast(message, duration) {
        if (!message) {
            return;
        }
        var el = document.getElementById('member-toast');
        if (!el) {
            el = document.createElement('div');
            el.id = 'member-toast';
            el.className = 'member-toast';
            document.body.appendChild(el);
        }
        el.textContent = message;
        el.classList.add('is-show');
        if (toastTimer) {
            clearTimeout(toastTimer);
        }
        toastTimer = setTimeout(function () {
            el.classList.remove('is-show');
        }, duration || 2200);
    }

    function errorToast(err) {
        toast((err && err.message) || dict.requestFailed);
    }

    function parseJSON(response) {
        return response.text().then(function (text) {
            if (!text) {
                return {};
            }
            try {
                return JSON.parse(text);
            } catch (e) {
                return { code: -1, msg: dict.requestFailed };
            }
        });
    }

    function send(url, options) {
        options = options || {};
        var init = {
            method: options.method || (options.body ? 'POST' : 'GET'),
            credentials: 'same-origin',
            headers: {}
        };
        if (options.body instanceof FormData) {
            init.body = options.body;
        } else if (options.body) {
            init.headers['Content-Type'] = 'application/json';
            init.body = JSON.stringify(options.body);
        }
        return fetch(url, init).then(function (response) {
            return parseJSON(response).then(function (result) {
                if (result && result.code === 0) {
                    return result;
                }
                throw new Error((result && result.msg) || dict.requestFailed);
            });
        });
    }

    window.Member = {
        get: function (url) {
            return send(url, { method: 'GET' });
        },
        post: function (url, body) {
            return send(url, { method: 'POST', body: body });
        },
        request: send,
        toast: toast,
        errorToast: errorToast
    };
})(window, document);

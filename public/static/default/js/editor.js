/**
 * 轻量富文本编辑器（无第三方依赖）
 *
 * 用法：
 *   <div class="mz-editor" id="content-editor" data-upload="/api/user/upload"></div>
 *   var editor = new MemberEditor(document.getElementById('content-editor'));
 *   editor.getHtml();  // 取内容
 *   editor.setHtml(html);
 *
 * 依赖浏览器 document.execCommand（已废弃但各浏览器仍完整支持），
 * 目的是让会员端发布页具备与后台一致的排版能力，而不引入额外资源包。
 */
(function (window, document) {
    'use strict';

    // 工具栏按钮一律用文字标签，不依赖图标字体（会员中心可能没加载字体文件）
    var BUTTONS = [
        { cmd: 'undo', title: '撤销', text: '↶' },
        { cmd: 'redo', title: '重做', text: '↷' },
        { cmd: 'bold', title: '加粗', text: 'B', style: 'font-weight:700' },
        { cmd: 'italic', title: '斜体', text: 'I', style: 'font-style:italic' },
        { cmd: 'underline', title: '下划线', text: 'U', style: 'text-decoration:underline' },
        { cmd: 'strikeThrough', title: '删除线', text: 'S', style: 'text-decoration:line-through' },
        { cmd: 'formatBlock:h2', title: '二级标题', text: 'H2' },
        { cmd: 'removeFormat', title: '清除格式', text: '清除' },
        { cmd: 'insertUnorderedList', title: '无序列表', text: '• 列表' },
        { cmd: 'insertOrderedList', title: '有序列表', text: '1. 列表' },
        { cmd: 'justifyLeft', title: '左对齐', text: '左' },
        { cmd: 'justifyCenter', title: '居中', text: '中' },
        { cmd: 'justifyRight', title: '右对齐', text: '右' },
        { cmd: 'formatBlock:blockquote', title: '引用', text: '❝' },
        { cmd: 'createLink', title: '插入链接', text: '链接' },
        { cmd: 'unlink', title: '移除链接', text: '去链接' },
        { cmd: 'insertImage', title: '插入图片', text: '图片' },
        { cmd: 'source', title: 'HTML 源码', text: '</>' }
    ];

    function el(tag, className, html) {
        var node = document.createElement(tag);
        if (className) {
            node.className = className;
        }
        if (html !== undefined) {
            node.innerHTML = html;
        }
        return node;
    }

    function MemberEditor(container, options) {
        options = options || {};
        this.container = container;
        this.uploadUrl = options.uploadUrl || container.getAttribute('data-upload') || '/api/user/upload';
        this.placeholder = options.placeholder || '请输入内容...';
        this.sourceMode = false;
        this.build();
    }

    MemberEditor.prototype.build = function () {
        var self = this;
        var root = el('div', 'mzed');
        var toolbar = el('div', 'mzed-toolbar');

        BUTTONS.forEach(function (item) {
            var btn = el('button');
            btn.type = 'button';
            btn.className = 'mzed-btn';
            btn.title = item.title;
            btn.textContent = item.text;
            btn.setAttribute('data-cmd', item.cmd);
            if (item.style) {
                btn.setAttribute('style', item.style);
            }
            if (item.cmd === 'source') {
                btn.addEventListener('click', function () {
                    self.toggleSource();
                });
            } else {
                btn.addEventListener('mousedown', function (e) {
                    // 防止点击工具栏时选区丢失
                    e.preventDefault();
                    self.exec(item.cmd);
                });
            }
            toolbar.appendChild(btn);
        });

        root.appendChild(toolbar);

        var body = el('div', 'mzed-body');
        var area = el('div', 'mzed-area');
        area.setAttribute('contenteditable', 'true');
        area.setAttribute('data-placeholder', this.placeholder);
        area.spellcheck = false;
        area.addEventListener('paste', function (e) {
            // 只保留纯 HTML，去掉脚本等危险标签
            var html = (e.clipboardData || window.clipboardData).getData('text/html');
            if (html) {
                e.preventDefault();
                self.insertHtml(self.sanitize(html));
            }
        });
        body.appendChild(area);
        root.appendChild(body);

        var source = el('textarea', 'mzed-source');
        source.hidden = true;
        root.appendChild(source);

        var fileInput = el('input');
        fileInput.type = 'file';
        fileInput.accept = 'image/*';
        fileInput.hidden = true;
        fileInput.addEventListener('change', function () {
            if (fileInput.files && fileInput.files[0]) {
                self.uploadImage(fileInput.files[0]);
            }
            fileInput.value = '';
        });
        root.appendChild(fileInput);

        this.root = root;
        this.area = area;
        this.source = source;
        this.fileInput = fileInput;
        this.container.appendChild(root);
    };

    MemberEditor.prototype.sanitize = function (html) {
        var tmp = document.createElement('div');
        tmp.innerHTML = html;
        ['script', 'style', 'iframe', 'object', 'embed', 'link', 'meta'].forEach(function (tag) {
            var nodes = tmp.getElementsByTagName(tag);
            while (nodes.length) {
                nodes[0].parentNode.removeChild(nodes[0]);
            }
        });
        var all = tmp.getElementsByTagName('*');
        for (var i = 0; i < all.length; i++) {
            var attrs = all[i].attributes;
            for (var j = attrs.length - 1; j >= 0; j--) {
                var name = attrs[j].name.toLowerCase();
                if (name.indexOf('on') === 0 || name === 'srcdoc') {
                    all[i].removeAttribute(attrs[j].name);
                }
            }
        }
        return tmp.innerHTML;
    };

    MemberEditor.prototype.focus = function () {
        this.area.focus();
    };

    MemberEditor.prototype.exec = function (cmd) {
        var parts = cmd.split(':');
        var name = parts[0];
        var value = parts[1] || null;
        if (name === 'insertImage') {
            this.fileInput.click();
            return;
        }
        if (name === 'createLink') {
            var url = window.prompt('请输入链接地址', 'https://');
            if (!url) {
                return;
            }
            value = url;
        }
        this.focus();
        try {
            document.execCommand(name, false, value);
        } catch (e) {
            // 忽略不支持的命令
        }
    };

    MemberEditor.prototype.insertHtml = function (html) {
        this.focus();
        try {
            document.execCommand('insertHTML', false, html);
        } catch (e) {
            this.area.innerHTML += html;
        }
    };

    MemberEditor.prototype.uploadImage = function (file) {
        var self = this;
        if (!/^image\//.test(file.type)) {
            window.alert('请选择图片文件');
            return;
        }
        var form = new FormData();
        form.append('file', file);
        var xhr = new XMLHttpRequest();
        xhr.open('POST', this.uploadUrl, true);
        xhr.withCredentials = true;
        xhr.onload = function () {
            var result = {};
            try {
                result = JSON.parse(xhr.responseText);
            } catch (e) {
                result = {};
            }
            if (result.code === 0 && result.data) {
                var src = result.data.url || result.data.path || '';
                if (src) {
                    self.insertHtml('<p><img src="' + src + '" alt=""></p>');
                }
            } else {
                window.alert((result && result.msg) || '图片上传失败');
            }
        };
        xhr.onerror = function () {
            window.alert('图片上传失败，请检查网络');
        };
        xhr.send(form);
    };

    MemberEditor.prototype.toggleSource = function () {
        if (!this.sourceMode) {
            this.source.value = this.area.innerHTML.trim();
            this.source.hidden = false;
            this.area.hidden = true;
            this.sourceMode = true;
        } else {
            this.area.innerHTML = this.sanitize(this.source.value);
            this.source.hidden = true;
            this.area.hidden = false;
            this.sourceMode = false;
        }
    };

    MemberEditor.prototype.getHtml = function () {
        var html = this.sourceMode ? this.source.value : this.area.innerHTML;
        // 去掉纯空内容（浏览器会塞 <br>）
        if (html.replace(/<br\s*\/?>|&nbsp;|\s/gi, '') === '') {
            return '';
        }
        return html.trim();
    };

    MemberEditor.prototype.setHtml = function (html) {
        this.area.innerHTML = this.sanitize(html || '');
    };

    window.MemberEditor = MemberEditor;
})(window, document);

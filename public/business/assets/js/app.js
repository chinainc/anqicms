/* =========================================================
   商家工作台 · 交互
   数据卡切换 / 询盘列表 / 待办 / 侧栏高亮 / 轻提示
   ========================================================= */
(function () {
  'use strict';

  /* ---------- 演示数据 ---------- */
  var METRICS = {
    realtime: [
      { label: '搜索曝光次数', value: '6,066', delta: '162%', up: true },
      { label: '搜索点击次数', value: '95', delta: '39.7%', up: true },
      { label: '询盘个数', value: '131', delta: '81.9%', up: true },
      { label: '访客人数', value: '0', delta: '100%', up: false },
      { label: '访问人数', value: '0', delta: '100%', up: false },
      { label: 'IM咨询人数', value: '7', delta: '0%', up: true }
    ],
    shop: [
      { label: '商品访客数', value: '1,382', delta: '8.4%', up: true },
      { label: '商品浏览量', value: '3,915', delta: '12.1%', up: true },
      { label: '收藏人数', value: '76', delta: '3.2%', up: true },
      { label: '加购人数', value: '41', delta: '2.6%', up: false },
      { label: '询盘转化率', value: '1.8%', delta: '0.4%', up: true },
      { label: '客单价', value: 'US$356', delta: '5.1%', up: false }
    ],
    market: [
      { label: '营销曝光量', value: '12,640', delta: '23.4%', up: true },
      { label: '点击量', value: '486', delta: '15.2%', up: true },
      { label: '营销花费', value: 'US$128.40', delta: '6.8%', up: false },
      { label: '营销询盘', value: '9', delta: '28.6%', up: true },
      { label: '点击率', value: '3.85%', delta: '1.2%', up: false },
      { label: 'ROI', value: '6.4', delta: '0.8%', up: true }
    ]
  };

  var RFQ = [
    {
      flag: 'us',
      title: '2026 Hot Selling Bandage Gown For Women Prom Party Dress',
      sub: 'Hello,I am currently developing my own fashion brand,specializes in evening dresses.',
      price: 'US$20.00/Piece',
      time: '5 小时前',
      seats: '剩余 8 个报价席位'
    },
    {
      flag: 'de',
      title: 'Maxi Dress Ladies',
      sub: "I'm looking for maxi, Product specifications:Material:Spandex/Elastane…",
      price: 'US$2,000.00/Piece',
      time: '10 小时前',
      seats: '剩余 2 个报价席位'
    },
    {
      flag: 'gb',
      title: 'Yellow Abstract Made Little Occasional Maxi Dress',
      sub: 'Made in China, custom sizes and labels available for bulk order.',
      price: 'US$2.00/Piece',
      time: '2 小时前',
      seats: '剩余 2 个报价席位'
    },
    {
      flag: 'fr',
      title: 'Strapless Solid Color Hourglass Waist Dress Is Slim Fit',
      sub: 'Made in China, need 500pcs per color, please quote with shipping.',
      price: 'US$18.50/Piece',
      time: '13 小时前',
      seats: '剩余 5 个报价席位'
    },
    {
      flag: 'au',
      title: 'Wholesale Custom Trench Coat Women Double Breasted',
      sub: 'Looking for 300pcs per style, OEM accepted, samples needed first.',
      price: 'US$26.00/Piece',
      time: '1 小时前',
      seats: '剩余 6 个报价席位'
    }
  ];

  var TODO = [
    { label: '未读人数', value: 0 },
    { label: '询盘回复', value: 3 },
    { label: '可营销客户', value: 20 },
    { label: '日超期', value: 0 },
    { label: '逾期待办', value: 0 },
    { label: '客诉不通过', value: 0 }
  ];

  /* ---------- 菜单结构 ---------- */
  var MENU = [
    { key: 'home', name: '首页', icon: 'i-home' },
    { key: 'common', name: '常用', icon: 'i-star', children: [
      { key: 'favorite', name: '我的收藏' },
      { key: 'recent', name: '最近访问' },
      { key: 'shortcut', name: '自定义快捷入口' }
    ] },
    { key: 'shop', name: '店铺管理', icon: 'i-shop', children: [
      { key: 'decorate', name: '装修店铺' },
      { key: 'manage', name: '管理店铺' },
      { key: 'compliance', name: '合规中心' },
      { key: 'shop-setting', name: '店铺设置' },
      { key: 'company', name: '管理公司信息' },
      { key: 'tariff', name: '设置交易品关税' },
      { key: 'trade-record', name: '贸易记录' }
    ] },
    { key: 'goods', name: '商品管理', icon: 'i-goods', children: [
      { key: 'goods-publish', name: '商品发布', ext: true },
      { key: 'goods-list', name: '商品管理' },
      { key: 'goods-operate', name: '商品运营' },
      { key: 'goods-console', name: '商品运营工作台' },
      { key: 'goods-industry', name: '行业定向征品' },
      { key: 'goods-ai-select', name: '智能选品', ext: true },
      { key: 'goods-semi', name: '半托管商品管理' },
      { key: 'goods-certified', name: '管理认证商品', ext: true },
      { key: 'goods-tools', name: '工具中心' },
      { key: 'goods-idea', name: '创意工坊', ext: true },
      { key: 'goods-image', name: '图片银行' },
      { key: 'goods-video', name: '视频银行' },
      { key: 'goods-catalog', name: '产品目录' },
      { key: 'goods-group', name: '商品分组与排序' },
      { key: 'goods-more', name: '更多工具' }
    ] },
    { key: 'media', name: '媒体中心', icon: 'i-play', dot: true, children: [
      { key: 'media-video', name: '视频' },
      { key: 'media-publish-video', name: '视频发布' },
      { key: 'media-publish-manage', name: '发布管理' },
      { key: 'media-contribute', name: '视频投稿', dot: true },
      { key: 'media-material', name: '视频素材', ext: true },
      { key: 'media-sns-share', name: 'SNS智能分享' },
      { key: 'media-sns-account', name: 'SNS账号管理' },
      { key: 'media-live', name: '直播' },
      { key: 'media-my-live', name: '我的直播' },
      { key: 'media-live-manage', name: '直播管理' },
      { key: 'media-3d', name: '3D' },
      { key: 'media-vr', name: 'VR实景展厅' },
      { key: 'media-3d-gen', name: '3D模型生成', dot: true }
    ] },
    { key: 'data', name: '数据参谋', icon: 'i-chart', children: [
      { key: 'data-shop', name: '店铺' },
      { key: 'data-overview', name: '数据概览' },
      { key: 'data-map', name: '数据地图' },
      { key: 'data-shop-analysis', name: '店铺分析' },
      { key: 'data-trade-analysis', name: '交易分析' },
      { key: 'data-showcase-analysis', name: '橱窗分析' },
      { key: 'data-staff-analysis', name: '员工分析' },
      { key: 'data-visitor-detail', name: '访客详情' },
      { key: 'data-opportunity', name: '商机' },
      { key: 'data-market-source', name: '市场来源' },
      { key: 'data-product-diagnosis', name: '产品诊断' },
      { key: 'data-selection', name: '选品洞察' },
      { key: 'data-traffic-source', name: '流量来源' },
      { key: 'data-buyer-insight', name: '买家洞察' },
      { key: 'data-ai', name: '智能' },
      { key: 'data-ai-marketing', name: '智能营销' },
      { key: 'data-p4p-effect', name: '直通车效果' },
      { key: 'data-academy', name: '学堂' },
      { key: 'data-school', name: '数据学园' },
      { key: 'data-dict', name: '数据字典' }
    ] },
    { key: 'marketing', name: '营销中心', icon: 'i-mega', children: [
      { key: 'mkt-home', name: '首页' },
      { key: 'mkt-p4p', name: '直通车' },
      { key: 'mkt-site-promo', name: '全站推广' },
      { key: 'mkt-standard', name: '标准推广' },
      { key: 'mkt-brand', name: '品牌推广' },
      { key: 'mkt-brand-pool', name: '品牌聚量' },
      { key: 'mkt-brand-keyword', name: '品牌关键词' },
      { key: 'mkt-brand-effect', name: '品效千单' },
      { key: 'mkt-annual', name: '营销年框' },
      { key: 'mkt-brand-zone', name: '品牌专区' },
      { key: 'mkt-analysis', name: '数据分析' },
      { key: 'mkt-p4p-report', name: '直通车报告' },
      { key: 'mkt-site-diagnosis', name: '全站营销诊断' },
      { key: 'mkt-brand-report', name: '品牌报告' },
      { key: 'mkt-brand-diagnosis', name: '品牌诊断' },
      { key: 'mkt-tools', name: '工具' },
      { key: 'mkt-creative', name: '创意中心' },
      { key: 'mkt-activity', name: '活动中心' },
      { key: 'mkt-marketing-tools', name: '营销工具' },
      { key: 'mkt-log', name: '操作日志' },
      { key: 'mkt-account', name: '推广账户' },
      { key: 'mkt-level', name: '营销等级' },
      { key: 'mkt-finance', name: '财务信息' }
    ] },
    { key: 'activity', name: '活动中心', icon: 'i-gift', children: [
      { key: 'platform', name: '平台活动' },
      { key: 'big-sale', name: '大促活动' },
      { key: 'shop-campaign', name: '店铺自营销' },
      { key: 'coupon', name: '优惠券' },
      { key: 'signup', name: '活动报名' },
      { key: 'activity-data', name: '活动数据' }
    ] },
    { key: 'im', name: '商机沟通', icon: 'i-chat', children: [
      { key: 'inquiry', name: '询盘管理' },
      { key: 'message', name: '旺铺留言' },
      { key: 'tm', name: 'TradeManager' },
      { key: 'inquiry-assist', name: '询盘助手' },
      { key: 'edm', name: '邮件营销' }
    ] },
    { key: 'crm', name: '客户管理', icon: 'i-users', children: [
      { key: 'crm-home', name: '客户概览' },
      { key: 'crm-groups', name: '客群' },
      { key: 'crm-fans', name: '粉丝列表' },
      { key: 'customer-list', name: '客户列表' },
      { key: 'crm-pool', name: '公海客户' },
      { key: 'crm-group-manage', name: '客群管理' },
      { key: 'crm-recycle', name: '回收站' },
      { key: 'crm-marketing', name: '营销' },
      { key: 'crm-camp', name: '客户营销' },
      { key: 'crm-aireach', name: 'AiReach' },
      { key: 'crm-activity', name: '营销活动' },
      { key: 'crm-template', name: '模板管理' },
      { key: 'crm-setting', name: '设置' },
      { key: 'crm-pool-setting', name: '公海设置' },
      { key: 'crm-edm', name: 'EDM设置' },
      { key: 'crm-category', name: '自定义品类' },
      { key: 'crm-shortcut', name: '快捷入口' },
      { key: 'crm-card-req', name: '索要名片的需求' },
      { key: 'crm-card-got', name: '收到的名片' }
    ] },
    { key: 'trade', name: '交易管理', icon: 'i-cart', children: [
      { key: 'trade-draft', name: '起草信用保障订单' },
      { key: 'trade-overview', name: '交易总览' },
      { key: 'order', name: '订单管理' },
      { key: 'trade-all', name: '所有订单' },
      { key: 'trade-review', name: '评价管理' },
      { key: 'trade-aftersale', name: '售后&amp;退款管理' },
      { key: 'trade-collect', name: '收款服务' },
      { key: 'trade-acp', name: '创建ACP+账单' },
      { key: 'trade-bill', name: '账单管理' },
      { key: 'trade-rights', name: '服务权益' },
      { key: 'trade-protect', name: '交易保障服务' },
      { key: 'trade-data', name: '交易核心数据' },
      { key: 'trade-growth', name: '交易资金成长中心' }
    ] },
    { key: 'export', name: '出口服务', icon: 'i-truck', children: [
      { key: 'export-order', name: '立即下单' },
      { key: 'export-orders', name: '出口服务订单' },
      { key: 'onetouch', name: '一达通' },
      { key: 'export-workbench', name: '出口工作台' },
      { key: 'export-prepare', name: '下单准备' },
      { key: 'export-brand', name: '品牌管理' },
      { key: 'export-product', name: '产品管理' },
      { key: 'export-invoicer', name: '开票人管理' },
      { key: 'export-customs', name: '报关单管理' },
      { key: 'export-taxrule', name: '智能税则' },
      { key: 'export-cert', name: '办证服务' },
      { key: 'export-inspect', name: '查验服务' },
      { key: 'export-tax', name: '退税管理' },
      { key: 'export-tax-follow', name: '退税跟进(3+N)' },
      { key: 'export-docs', name: '备案单证' },
      { key: 'export-agent', name: '代理证列表' },
      { key: 'export-entrust', name: '委托证(2+N)' },
      { key: 'export-tariff-svc', name: '带税服务' },
      { key: 'export-compliance', name: '合规服务' },
      { key: 'export-finance', name: '财税服务' },
      { key: 'export-ai-tax', name: 'AI退税全托管' },
      { key: 'export-purchase', name: '服务采购订单' },
      { key: 'export-more', name: '更多操作' },
      { key: 'export-base', name: '基础信息设置' }
    ] },
    { key: 'logistics', name: '物流服务', icon: 'i-box', children: [
      { key: 'logi-quote', name: '查询报价并下单' },
      { key: 'logi-workbench', name: '物流工作台' },
      { key: 'logi-order', name: '订单管理' },
      { key: 'logi-orders', name: '物流订单' },
      { key: 'logi-track', name: '物流跟踪' },
      { key: 'logi-freight', name: '运费模板' },
      { key: 'logi-address', name: '接收地址管理' },
      { key: 'logi-settle', name: '结算与权益' },
      { key: 'logi-claim', name: '物流索赔与开票' },
      { key: 'logi-rights', name: '物流客户权益' },
      { key: 'logi-claim-manage', name: '理赔管理' },
      { key: 'logi-domestic', name: '国内仓发货' },
      { key: 'logi-signup', name: '商品报名仓发' },
      { key: 'logi-warehouse', name: '仓库工作台' },
      { key: 'logi-overseas', name: '海外仓管理' },
      { key: 'logi-quality', name: '认证仓质量管理' },
      { key: 'logi-goods-bind', name: '商品与货品绑定' }
    ] },
    { key: 'fund', name: '资金管理', icon: 'i-wallet', children: [
      { key: 'fund-home', name: 'Alibaba.com Pay' },
      { key: 'fund-service', name: '账户服务' },
      { key: 'fund-overview', name: '账户总览' },
      { key: 'fund-detail', name: '资金明细' },
      { key: 'fund-collect', name: '收款服务' },
      { key: 'fund-collect-manage', name: '收款管理' },
      { key: 'fund-tuoshou', name: '托收' },
      { key: 'fund-lc', name: '信用证' },
      { key: 'fund-withdraw-apply', name: '提款申请' },
      { key: 'fund-withdraw', name: '提款' },
      { key: 'fund-pay', name: '付款' },
      { key: 'fund-fx', name: '汇兑结汇' },
      { key: 'fund-fx-service', name: '汇兑服务' },
      { key: 'fund-finance', name: '融资服务' },
      { key: 'fund-insurance', name: '保险服务' },
      { key: 'fund-bill-service', name: '账票服务' },
      { key: 'fund-fee', name: '费用缴纳' },
      { key: 'fund-invoice', name: '开发票' },
      { key: 'fund-account', name: '个人中心' },
      { key: 'fund-security', name: '账户与安全' }
    ] },
    { key: 'verify', name: '认证中心', icon: 'i-badge', children: [
      { key: 'verify-cert', name: '证书管理' },
      { key: 'verify-manage', name: '认证管理' },
      { key: 'verify-home', name: '认证首页' },
      { key: 'verify-site', name: '实地认证' },
      { key: 'verify-gold', name: '金品认证' },
      { key: 'verify-report', name: '认证报告' },
      { key: 'verify-show', name: '全景实力表达' },
      { key: 'verify-industry', name: '行业竞争力' },
      { key: 'verify-label', name: '实力标签' },
      { key: 'verify-card', name: '实力卡片配置' },
      { key: 'verify-media', name: '实力多媒体管理' }
    ] },
    { key: 'grow', name: '商家成长', icon: 'i-growth', children: [
      { key: 'grow-level', name: '商家等级' },
      { key: 'grow-guide', name: '成长指引' },
      { key: 'grow-home', name: '商家成长' },
      { key: 'grow-member', name: '会员中心' },
      { key: 'grow-rights', name: '专属权益' },
      { key: 'grow-class', name: '外贸云课堂' }
    ] },
    { key: 'assistant', name: '生意助手', icon: 'i-robot', dot: true, children: [
      { key: 'ai-manage', name: '管理' },
      { key: 'ai-home', name: '生意助手概览' },
      { key: 'ai-staff', name: '数字员工' },
      { key: 'ai-wiki', name: '知识库' },
      { key: 'ai-buy', name: '购买记录' },
      { key: 'ai-guide', name: '使用说明' },
      { key: 'ai-camp', name: '活动' },
      { key: 'ai-task', name: '评测任务' }
    ] },
    { key: 'service', name: '服务中心', icon: 'i-headset', children: [
      { key: 'svc-rule', name: '规则与合规' },
      { key: 'svc-rule-channel', name: '规则频道' },
      { key: 'svc-complaint', name: '投诉违规中心' },
      { key: 'svc-ip', name: '知识产权保护' },
      { key: 'svc-appeal', name: '申诉中心' },
      { key: 'svc-compliance', name: '合规管理' },
      { key: 'svc-market', name: '服务市场' },
      { key: 'svc-selection', name: '选品中心' },
      { key: 'svc-trade-market', name: '外贸服务市场' },
      { key: 'svc-open', name: '开放平台' },
      { key: 'svc-open-info', name: '开放信息' },
      { key: 'svc-app-config', name: '应用配置及权限' },
      { key: 'svc-account', name: '账户中心' },
      { key: 'svc-promo', name: '促销活动' },
      { key: 'svc-customer', name: '客户服务' },
      { key: 'svc-customer-center', name: '客户服务中心' }
    ] }
  ];

  var current = { group: 'home', page: 'home' };

  function groupOf(key) {
    for (var i = 0; i < MENU.length; i++) {
      if (MENU[i].key === key) return MENU[i];
    }
    return null;
  }

  /* ---------- 导航渲染 ---------- */
  function renderNav() {
    var nav = document.getElementById('nav');
    if (!nav) return;
    nav.innerHTML = MENU.map(function (m) {
      return '<li class="nav__cell">' +
          '<a class="nav__item" href="#" data-group="' + m.key + '" title="' + m.name + '">' +
            '<svg class="ico"><use href="#' + m.icon + '"/></svg>' +
            '<span>' + m.name + '</span>' +
            (m.dot ? '<i class="nav__dot"></i>' : '') +
          '</a>' +
        '</li>';
    }).join('') +
      '<li class="nav__more" id="navMore" hidden>' +
        '<button class="nav__item nav__morebtn" type="button">更多' +
          '<svg class="caret"><use href="#i-arrow"/></svg></button>' +
        '<div class="nav__drop" id="navDrop"></div>' +
      '</li>';
  }

  /* 一级菜单过多时收进「更多」下拉 */
  function layoutNav() {
    var nav = document.getElementById('nav');
    var more = document.getElementById('navMore');
    var drop = document.getElementById('navDrop');
    if (!nav || !more || !drop) return;
    var cells = Array.prototype.filter.call(nav.children, function (li) {
      return li.classList && li.classList.contains('nav__cell');
    });
    if (!cells.length) return;

    cells.forEach(function (li) { li.hidden = false; });
    more.hidden = true;
    drop.innerHTML = '';
    more.classList.remove('is-open');

    var avail = nav.clientWidth;
    var widths = cells.map(function (li) { return li.getBoundingClientRect().width; });
    var total = widths.reduce(function (s, w) { return s + w; }, 0);
    if (total <= avail) return;

    var moreW = 82;
    var used = 0;
    var rest = [];
    cells.forEach(function (li, i) {
      if (used + widths[i] <= avail - moreW) used += widths[i];
      else rest.push(li);
    });
    if (rest.length) {
      rest.forEach(function (li) { li.hidden = true; });
      drop.innerHTML = rest.map(function (li) {
        var a = li.querySelector('.nav__item');
        return '<a class="nav__dropitem" href="#" data-group="' + a.dataset.group + '">' +
          a.innerHTML + '</a>';
      }).join('');
      more.hidden = false;
      paintNav();
    }
  }

  function paintNav() {
    var items = document.querySelectorAll('.nav__item, .nav__dropitem');
    Array.prototype.forEach.call(items, function (it) {
      var m = groupOf(it.dataset.group);
      var on = !!m && it.dataset.group === current.group;
      it.classList.toggle('is-active', on);
      it.classList.toggle('is-open', on && !!(m && m.children));
    });
  }

  function renderSubnav(m) {
    var box = document.getElementById('subnav');
    var title = document.getElementById('subnavTitle');
    var list = document.getElementById('subnavList');
    if (!box) return;
    if (!m || !m.children) {
      box.hidden = true;
      return;
    }
    title.textContent = m.name;
    list.innerHTML = m.children.map(function (c) {
      return '<li><a class="subnav__item" href="#" data-page="' + c.key + '">' +
        '<span class="subnav__text">' + c.name + '</span>' +
        (c.dot ? '<i class="subnav__dot"></i>' : '') +
        (c.ext ? '<span class="subnav__ext">↗</span>' : '') +
      '</a></li>';
    }).join('');
    box.hidden = false;
  }

  function paintSubnav() {
    var items = document.querySelectorAll('.subnav__item');
    Array.prototype.forEach.call(items, function (it) {
      it.classList.toggle('is-active', it.dataset.page === current.page);
    });
  }

  function showDashboard() {
    document.getElementById('dashboard').hidden = false;
    document.getElementById('pageview').hidden = true;
  }

  function showPage(m, child) {
    document.getElementById('pageCrumb').textContent = m.name + ' / ' + child.name;
    document.getElementById('pageTitle').textContent = child.name;
    document.getElementById('dashboard').hidden = true;
    document.getElementById('pageview').hidden = false;
    renderPage(m, child);
  }

  function openGroup(key) {
    var m = groupOf(key);
    if (!m) return;
    current.group = key;
    paintNav();
    if (!m.children) {
      current.page = key;
      renderSubnav(null);
      showDashboard();
      return;
    }
    renderSubnav(m);
    var first = defaultPageFor(m);
    current.page = first;
    var start = m.children[0];
    m.children.forEach(function (c) { if (c.key === first) start = c; });
    paintSubnav();
    showPage(m, start);
  }

  function openPage(key) {
    var m = groupOf(current.group);
    if (!m || !m.children) return;
    var child = null;
    m.children.forEach(function (c) { if (c.key === key) child = c; });
    if (!child) return;
    current.page = child.key;
    paintSubnav();
    showPage(m, child);
  }

  /* ---------- 二级页面数据 ---------- */
  var LEVEL_CLS = { 'VIP 客户': 'purple', '重点客户': 'orange', '普通客户': 'blue', '潜在客户': 'gray' };

  var ORDER_ST = {
    unpaid: ['待买家付款', 'orange'],
    paid: ['待发货', 'blue'],
    shipped: ['已发货', 'purple'],
    done: ['已完成', 'green'],
    closed: ['已关闭', 'gray']
  };

  var CUSTOMERS = [
    { name: 'Emily Carter', company: 'Luxe Apparel LLC', ini: 'EC', country: '美国', source: '询盘', inq: 12, ord: 3, follow: '2026-11-08', level: '重点客户', owner: '张敏', tab: 'mine vip' },
    { name: 'Aisha Rahman', company: 'Dubai Trends Trading FZE', ini: 'AR', country: '阿联酋', source: '旺铺', inq: 24, ord: 7, follow: '2026-11-08', level: 'VIP 客户', owner: '王强', tab: 'mine vip' },
    { name: 'Lucas Meyer', company: 'Meyer Fashion GmbH', ini: 'LM', country: '德国', source: 'RFQ', inq: 8, ord: 1, follow: '2026-11-07', level: '普通客户', owner: '李静', tab: 'mine' },
    { name: 'James Wilson', company: 'Wilson Retail UK Ltd.', ini: 'JW', country: '英国', source: '询盘', inq: 16, ord: 4, follow: '2026-11-06', level: '重点客户', owner: '李静', tab: 'mine vip' },
    { name: 'Sophie Martin', company: 'Paris Mode SARL', ini: 'SM', country: '法国', source: '展会', inq: 5, ord: 0, follow: '2026-11-06', level: '普通客户', owner: '张敏', tab: 'mine' },
    { name: 'Yuki Tanaka', company: 'Tokyo Style Co., Ltd.', ini: 'YT', country: '日本', source: '询盘', inq: 21, ord: 6, follow: '2026-11-05', level: 'VIP 客户', owner: '李静', tab: 'mine vip' },
    { name: 'Olivia Brown', company: 'Sydney Boutique Pty Ltd.', ini: 'OB', country: '澳大利亚', source: '站内信', inq: 3, ord: 0, follow: '2026-11-04', level: '潜在客户', owner: '王强', tab: 'pool' },
    { name: 'Diego Alvarez', company: 'Moda Madrid S.L.', ini: 'DA', country: '西班牙', source: 'RFQ', inq: 9, ord: 2, follow: '2026-11-03', level: '普通客户', owner: '张敏', tab: 'mine' },
    { name: 'Ahmed Hassan', company: 'Cairo Fashion Import', ini: 'AH', country: '埃及', source: '旺铺', inq: 2, ord: 0, follow: '2026-10-30', level: '潜在客户', owner: '-', tab: 'pool' },
    { name: 'Ana Silva', company: 'Lisboa Confeccoes Lda', ini: 'AS', country: '葡萄牙', source: '展会', inq: 6, ord: 1, follow: '2026-10-28', level: '普通客户', owner: '-', tab: 'pool' }
  ];

  var ORDERS = [
    { no: '2026110800123', t: '2026-11-08 14:22', goods: '女士针织收腰长袖连衣裙', qty: 60, buyer: 'Emily Carter', amount: 'US$1,860.00', ship: 'US$120.00', pay: 'T/T', st: 'paid' },
    { no: '2026110800119', t: '2026-11-08 11:05', goods: '法式方领碎花茶歇裙', qty: 120, buyer: 'Aisha Rahman', amount: 'US$3,240.00', ship: 'US$0.00', pay: '信用卡', st: 'unpaid' },
    { no: '2026110700098', t: '2026-11-07 17:48', goods: '双面呢羊毛大衣', qty: 30, buyer: 'James Wilson', amount: 'US$2,175.00', ship: 'US$160.00', pay: 'T/T', st: 'shipped' },
    { no: '2026110700091', t: '2026-11-07 09:31', goods: '高腰阔腿西装裤', qty: 200, buyer: 'Yuki Tanaka', amount: 'US$2,980.00', ship: 'US$240.00', pay: '支付宝', st: 'done' },
    { no: '2026110600087', t: '2026-11-06 16:02', goods: '缎面吊带晚礼服', qty: 80, buyer: 'Sophie Martin', amount: 'US$4,120.00', ship: 'US$180.00', pay: '信用证', st: 'done' },
    { no: '2026110600081', t: '2026-11-06 10:26', goods: '修身显瘦职业套裙', qty: 45, buyer: 'Diego Alvarez', amount: 'US$1,395.00', ship: 'US$95.00', pay: 'T/T', st: 'paid' },
    { no: '2026110500076', t: '2026-11-05 15:11', goods: '复古格纹半身裙', qty: 25, buyer: 'Lucas Meyer', amount: 'US$675.00', ship: 'US$60.00', pay: '信用卡', st: 'closed' },
    { no: '2026110500072', t: '2026-11-05 08:44', goods: '宽松休闲针织开衫', qty: 150, buyer: 'Olivia Brown', amount: 'US$2,250.00', ship: 'US$130.00', pay: 'T/T', st: 'done' }
  ];

  var ONETOUCH_STATS = [
    { label: '累计出口额', value: 'US$1,286,400', foot: '较上月 <b>+18.2%</b>' },
    { label: '待处理服务单', value: '6', foot: '其中 2 单待补资料' },
    { label: '退税中金额', value: 'US$86,420', foot: '预计 12 月到账' },
    { label: '授信额度', value: 'US$500,000', foot: '已用 US$182,000' }
  ];

  var ONETOUCH_FLOW = [
    { t: '出口订单确认', d: '已授权', done: true },
    { t: '报关资料提交', d: '待补 1 项', done: true },
    { t: '海关通关', d: '正常', done: true },
    { t: '外汇结算', d: '自动结汇', done: false },
    { t: '退税申报', d: '进行中', done: false }
  ];

  var ONETOUCH = [
    { no: 'YDT20261108001', order: '2026110800123', customs: '531620260100123456', amount: 'US$1,860.00', pass: '已通关', passCls: 'green', tax: '退税中', taxCls: 'orange' },
    { no: 'YDT20261107004', order: '2026110700098', customs: '531620260100119873', amount: 'US$2,175.00', pass: '已通关', passCls: 'green', tax: '已退税', taxCls: 'green' },
    { no: 'YDT20261106003', order: '2026110600087', customs: '531620260100114420', amount: 'US$4,120.00', pass: '已通关', passCls: 'green', tax: '申报中', taxCls: 'blue' },
    { no: 'YDT20261105002', order: '2026110500072', customs: '-', amount: 'US$2,250.00', pass: '待报关', passCls: 'orange', tax: '未申报', taxCls: 'gray' },
    { no: 'YDT20261104001', order: '2026110400068', customs: '531620260100108810', amount: 'US$3,480.00', pass: '查验中', passCls: 'purple', tax: '未申报', taxCls: 'gray' }
  ];

  /* ---------- 二级页面 · 组件片段 ---------- */
  function tabsHTML(tabs) {
    return '<div class="pv-tabs">' + tabs.map(function (t, i) {
      return '<button class="pv-tab' + (i === 0 ? ' is-active' : '') + '" type="button" data-tabk="' + t.k + '">' +
        t.n + '<b>' + t.c + '</b></button>';
    }).join('') + '</div>';
  }

  function filtersHTML(list) {
    var cells = list.map(function (f) {
      if (f.t === 'input') {
        return '<div class="pv-field"><label>' + f.label + '</label>' +
          '<input class="pv-input" type="text" placeholder="' + f.ph + '"></div>';
      }
      return '<div class="pv-field"><label>' + f.label + '</label><select class="pv-select">' +
        f.opts.map(function (o) { return '<option>' + o + '</option>'; }).join('') +
        '</select></div>';
    }).join('');
    return '<div class="pv-filters">' + cells +
      '<div class="pv-filters__acts">' +
        '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="查询">查询</button>' +
        '<button class="pv-btn" type="button" data-pv-reset>重置</button>' +
      '</div></div>';
  }

  function barHTML(hint, acts) {
    return '<div class="pv-bar"><p class="pv-bar__hint">' + hint + '</p><div class="pv-bar__acts">' +
      acts.map(function (a) {
        return '<button class="pv-btn' + (a.pri ? ' pv-btn--pri' : '') + '" type="button" data-pv-act="' +
          a.n + '">' + a.n + '</button>';
      }).join('') + '</div></div>';
  }

  function pagerHTML(total, page, last) {
    var nums = '';
    for (var i = 1; i <= Math.min(last, 3); i++) {
      nums += '<button class="pv-page' + (i === page ? ' is-active' : '') + '" type="button" data-pv-page="' + i + '">' + i + '</button>';
    }
    if (last > 4) {
      nums += '<span class="pv-pager__gap">…</span>' +
        '<button class="pv-page" type="button" data-pv-page="' + last + '">' + last + '</button>';
    }
    return '<div class="pv-pager">' +
      '<span class="pv-pager__total">共 ' + total + ' 条，当前第 ' + page + ' / ' + last + ' 页</span>' +
      '<button class="pv-page" type="button" data-pv-nav="prev">上一页</button>' +
      nums +
      '<button class="pv-page" type="button" data-pv-nav="next">下一页</button>' +
      '<select class="pv-select pv-select--sm"><option>10 条/页</option><option>20 条/页</option><option>50 条/页</option></select>' +
      '</div>';
  }

  function tagHTML(text, cls) {
    return '<span class="pv-tag pv-tag--' + cls + '">' + text + '</span>';
  }

  /* ---------- 二级页面 · 客户列表 ---------- */
  function custRow(c) {
    return '<tr data-tab="' + c.tab + '">' +
      '<td><input class="pv-check pv-rowchk" type="checkbox"></td>' +
      '<td><div class="pv-cust"><span class="pv-ava">' + c.ini + '</span><div class="pv-cust__box">' +
        '<a class="pv-cust__name" href="#">' + c.name + '</a>' +
        '<p class="pv-cust__sub">' + c.company + '</p></div></div></td>' +
      '<td>' + c.country + '</td>' +
      '<td>' + c.source + '</td>' +
      '<td class="ta-c num">' + c.inq + '</td>' +
      '<td class="ta-c num">' + c.ord + '</td>' +
      '<td class="pv-muted">' + c.follow + '</td>' +
      '<td>' + tagHTML(c.level, LEVEL_CLS[c.level]) + '</td>' +
      '<td>' + c.owner + '</td>' +
      '<td><div class="pv-ops"><a class="pv-link" href="#">跟进</a>' +
        '<a class="pv-link pv-link--muted" href="#">详情</a></div></td>' +
      '</tr>';
  }

  function pageCustomer() {
    var tabs = [
      { k: 'all', n: '全部客户', c: 128 },
      { k: 'mine', n: '我负责的', c: 86 },
      { k: 'vip', n: '重点客户', c: 24 },
      { k: 'pool', n: '公海客户', c: 42 }
    ];
    var html = '<div class="pv-card">' +
      tabsHTML(tabs) +
      filtersHTML([
        { t: 'input', label: '客户信息', ph: '客户名称 / 公司名' },
        { t: 'select', label: '国家/地区', opts: ['全部', '美国', '德国', '英国', '法国', '阿联酋', '日本'] },
        { t: 'select', label: '客户等级', opts: ['全部', 'VIP 客户', '重点客户', '普通客户', '潜在客户'] },
        { t: 'select', label: '客户来源', opts: ['全部', '询盘', 'RFQ', '旺铺', '展会', '站内信'] }
      ]) +
      barHTML('共 <b>128</b> 位客户，其中 <b>12</b> 位近 7 天有新询盘', [
        { n: '新建客户', pri: true }, { n: '导入客户' }, { n: '批量打标签' }, { n: '导出' }
      ]) +
      '<div class="pv-tablewrap"><table class="pv-table"><thead><tr>' +
        '<th style="width:34px"><input class="pv-check" type="checkbox" id="pvAll"></th>' +
        '<th>客户信息</th><th>国家/地区</th><th>客户来源</th>' +
        '<th class="ta-c">询盘数</th><th class="ta-c">订单数</th>' +
        '<th>最近跟进</th><th>客户等级</th><th>负责人</th><th style="width:118px">操作</th>' +
      '</tr></thead><tbody>' + CUSTOMERS.map(custRow).join('') + '</tbody></table></div>' +
      pagerHTML(128, 1, 13) +
      '</div>';
    return { html: html };
  }

  /* ---------- 二级页面 · 订单管理 ---------- */
  function orderRow(o) {
    var st = ORDER_ST[o.st];
    return '<tr data-tab="' + o.st + '">' +
      '<td><input class="pv-check pv-rowchk" type="checkbox"></td>' +
      '<td><div class="pv-goods"><span class="pv-thumb">' + o.goods.charAt(0) + '</span>' +
        '<div class="pv-order"><a class="pv-order__no" href="#">' + o.no + '</a>' +
        '<p class="pv-order__sub">' + o.goods + ' × ' + o.qty + '</p>' +
        '<p class="pv-order__sub">下单时间：' + o.t + '</p></div></div></td>' +
      '<td>' + o.buyer + '</td>' +
      '<td class="num">' + o.amount + '</td>' +
      '<td class="num num--plain pv-muted">' + o.ship + '</td>' +
      '<td>' + tagHTML(st[0], st[1]) + '</td>' +
      '<td class="pv-muted">' + o.pay + '</td>' +
      '<td><div class="pv-ops"><a class="pv-link" href="#">订单详情</a>' +
        '<a class="pv-link pv-link--muted" href="#">联系买家</a></div></td>' +
      '</tr>';
  }

  function pageOrder() {
    var tabs = [
      { k: 'all', n: '全部订单', c: 128 },
      { k: 'unpaid', n: '待买家付款', c: 3 },
      { k: 'paid', n: '待发货', c: 12 },
      { k: 'shipped', n: '已发货', c: 24 },
      { k: 'done', n: '已完成', c: 86 },
      { k: 'closed', n: '已关闭', c: 3 }
    ];
    var html = '<div class="pv-card">' +
      tabsHTML(tabs) +
      filtersHTML([
        { t: 'input', label: '订单号', ph: '订单号 / 买家账号' },
        { t: 'input', label: '商品', ph: '商品名称 / 型号' },
        { t: 'select', label: '创建时间', opts: ['近 7 天', '近 30 天', '近 90 天', '全部'] },
        { t: 'select', label: '支付方式', opts: ['全部', 'T/T', '信用卡', '支付宝', '信用证'] }
      ]) +
      barHTML('近 30 天成交 <b>US$48,620.00</b>，待发货 <b>12</b> 单，请在 3 个自然日内完成发货', [
        { n: '起草订单', pri: true }, { n: '批量发货' }, { n: '导出订单' }
      ]) +
      '<div class="pv-tablewrap"><table class="pv-table"><thead><tr>' +
        '<th style="width:34px"><input class="pv-check" type="checkbox" id="pvAll"></th>' +
        '<th>订单信息</th><th>买家</th><th>订单金额</th><th>运费</th>' +
        '<th>订单状态</th><th>支付方式</th><th style="width:150px">操作</th>' +
      '</tr></thead><tbody>' + ORDERS.map(orderRow).join('') + '</tbody></table></div>' +
      pagerHTML(128, 1, 13) +
      '</div>';
    return { html: html };
  }

  /* ---------- 二级页面 · 一达通（出口服务） ---------- */
  function onetouchRow(r) {
    return '<tr>' +
      '<td><a class="pv-order__no" href="#">' + r.no + '</a></td>' +
      '<td class="pv-muted">' + r.order + '</td>' +
      '<td class="pv-muted">' + r.customs + '</td>' +
      '<td class="num">' + r.amount + '</td>' +
      '<td>' + tagHTML(r.pass, r.passCls) + '</td>' +
      '<td>' + tagHTML(r.tax, r.taxCls) + '</td>' +
      '<td><div class="pv-ops"><a class="pv-link" href="#">服务详情</a>' +
        '<a class="pv-link pv-link--muted" href="#">报关资料</a></div></td>' +
      '</tr>';
  }

  function pageOnetouch() {
    var stats = '<div class="pv-stats">' + ONETOUCH_STATS.map(function (s) {
      return '<div class="pv-stat"><p class="pv-stat__label">' + s.label + '</p>' +
        '<p class="pv-stat__value">' + s.value + '</p>' +
        '<p class="pv-stat__foot">' + s.foot + '</p></div>';
    }).join('') + '</div>';

    var flow = '<div class="pv-flow">' + ONETOUCH_FLOW.map(function (s, i) {
      return (i ? '<span class="pv-step__line"></span>' : '') +
        '<div class="pv-step' + (s.done ? ' is-done' : '') + '">' +
          '<span class="pv-step__no">' + (s.done ? '✓' : i + 1) + '</span>' +
          '<span class="pv-step__box">' +
            '<span class="pv-step__t">' + s.t + '</span>' +
            '<span class="pv-step__d">' + s.d + '</span>' +
          '</span>' +
        '</div>';
    }).join('') + '</div>';

    var html = stats +
      '<div class="pv-card">' +
        '<div class="pv-bar"><p class="pv-bar__hint">一达通出口服务 · 通关、结汇、退税一站式办理</p>' +
          '<div class="pv-bar__acts">' +
            '<button class="pv-btn" type="button" data-pv-act="下载服务协议">下载服务协议</button>' +
            '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="申请一达通服务">申请一达通服务</button>' +
          '</div></div>' +
        flow +
        '<div class="pv-tablewrap"><table class="pv-table"><thead><tr>' +
          '<th>服务单号</th><th>关联订单</th><th>报关单号</th><th>出口金额</th>' +
          '<th>通关状态</th><th>退税状态</th><th style="width:150px">操作</th>' +
        '</tr></thead><tbody>' + ONETOUCH.map(onetouchRow).join('') + '</tbody></table></div>' +
        pagerHTML(24, 1, 3) +
      '</div>';
    return {
      html: html,
      acts: '<button class="pv-btn" type="button" data-pv-act="服务规则">服务规则</button>' +
        '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="在线咨询">在线咨询</button>'
    };
  }

  /* ---------- 通用小组件 ---------- */
  function kpiRowHTML(list) {
    return '<div class="xp-kpis">' + list.map(function (k) {
      return '<div class="xp-kpi"><p class="xp-kpi__label">' + k.label + '</p>' +
        '<p class="xp-kpi__value">' + k.v + '</p>' +
        '<p class="xp-kpi__foot' + (k.up ? ' is-up' : '') + '">' + k.foot + '</p></div>';
    }).join('') + '</div>';
  }

  function heroHTML(t, d, btn) {
    return '<div class="xp-hero"><div class="xp-hero__main">' +
      '<p class="xp-hero__t">' + t + '</p>' +
      '<p class="xp-hero__d">' + d + '</p></div>' +
      '<div class="xp-hero__side">' + (btn || '') + '</div></div>';
  }

  function sectionHead(t, link) {
    return '<div class="xp-shead"><h3 class="xp-shead__t">' + t + '</h3>' +
      (link ? '<a class="pv-link" href="#">' + link + '</a>' : '') + '</div>';
  }

  /* ---------- 二级页面 · 推广计划（营销中心 / 全站推广） ---------- */
  var PLAN_KPI = [
    { label: '全站商机量', v: '0', foot: '较昨日 0%' },
    { label: '全站商机成本', v: '¥ 0.00', foot: '较昨日 0%' },
    { label: 'L1+ 全站商机量', v: '0', foot: '较昨日 0%' },
    { label: '全站商机转化率', v: '0.00%', foot: '较昨日 0%' },
    { label: '点击量', v: '50', foot: '较昨日 ↑100%', up: true },
    { label: '点击成本', v: '¥ 4.24', foot: '较昨日 ↑100%', up: true }
  ];

  var PLANS = [
    { name: '全站推广_爆品_优质助推_20261101', type: '全站推广-优质助推', diag: '可优化1项', budget: '¥200.00/天', bid: '¥91.38', goods: 19, cost: '¥211.82', opp: 0 },
    { name: '全站推广_爆品_强力拉新_20261028', type: '全站推广-强力拉新', diag: '出价过低', low: true, budget: '¥150.00/天', bid: '¥37.00', goods: 0, cost: '¥0.00', opp: 0 },
    { name: '全站推广_爆品_优质助推_20261020', type: '全站推广-优质助推', diag: '可优化2项', budget: '¥150.00/天', bid: '¥43.00', goods: 22, cost: '¥0.00', opp: 0 },
    { name: '优质助推 05-12', type: '全站推广-优质助推', diag: '出价过低', low: true, budget: '¥150.00/天', bid: '¥45.00', goods: 11, cost: '¥0.00', opp: 0 },
    { name: '优质计划 11-29', type: '全站推广-优质助推', diag: '出价过低', low: true, budget: '¥150.00/天', bid: '¥42.00', goods: 52, cost: '¥0.00', opp: 0 },
    { name: '全站推广_爆品_易直达_20261012', type: '全站推广-易直达', diag: '出价过低', low: true, budget: '¥150.00/天', bid: '¥45.00', goods: 82, cost: '¥0.00', opp: 0 },
    { name: '全站推广_爆品_强力拉新_20261008', type: '全站推广-强力拉新', diag: '出价过低', low: true, budget: '¥150.00/天', bid: '¥31.00', goods: 32, cost: '¥0.00', opp: 0 }
  ];

  function planRow(p) {
    return '<tr>' +
      '<td><label class="xp-plan"><input class="pv-check" type="checkbox">' +
        '<span class="xp-plan__box"><a class="xp-plan__name" href="#">' + p.name + '</a>' +
        '<span class="xp-plan__type">类型：' + p.type + '</span></span></label></td>' +
      '<td>全站商机</td>' +
      '<td>' + (p.diag ? '<span class="xp-diag"><i></i>' + p.diag + '</span>' : '-') + '</td>' +
      '<td class="num">' + p.budget + '</td>' +
      '<td><span class="xp-bid">' + p.bid + '</span>' +
        (p.low ? '<span class="xp-low">出价过低</span>' : '') +
        '<span class="xp-sub">目标商机成本</span></td>' +
      '<td class="ta-c num">' + p.goods + '</td>' +
      '<td class="num">' + p.cost + '</td>' +
      '<td class="ta-c num">' + p.opp + '</td>' +
      '<td class="ta-c"><span class="xp-rec">L3+ 推荐</span></td>' +
      '</tr>';
  }

  function pagePlan() {
    var tabs = [
      { k: 'plan', n: '推广计划' },
      { k: 'smart', n: '智能推广' },
      { k: 'site', n: '全站推广' }
    ];
    var html = '<div class="pv-card">' +
      '<div class="xp-planhead">' +
        '<div class="pv-tabs xp-tabs--plain">' + tabs.map(function (t, i) {
          return '<button class="pv-tab' + (i === 0 ? ' is-active' : '') + '" type="button">' + t.n + '</button>';
        }).join('') + '</div>' +
        '<p class="xp-planhead__count">推广中计划数 <b>1</b> <i class="xp-i">i</i></p>' +
      '</div>' +
      '<div class="xp-toolrow">' +
        '<div class="xp-toolrow__l">' +
          '<select class="pv-select pv-select--sm"><option>全部计划(16)</option><option>推广中(1)</option><option>已暂停(15)</option></select>' +
          '<span class="xp-daterange"><i class="xp-cal"></i>2026-11-08 ~ 2026-11-08</span>' +
        '</div>' +
        '<div class="xp-toolrow__r">' +
          '<a class="pv-link" href="#">自定义列</a>' +
          '<a class="pv-link" href="#">查看数据报告</a>' +
        '</div>' +
      '</div>' +
      kpiRowHTML(PLAN_KPI) +
      '<div class="xp-listhead">' +
        '<div class="xp-search"><svg class="ico"><use href="#i-search"/></svg>' +
          '<input class="xp-search__in" type="text" placeholder="请输入计划..."></div>' +
        '<button class="pv-btn" type="button" data-pv-act="全部筛选指标">全部筛选指标</button>' +
        '<a class="pv-link xp-listhead__col" href="#">自定义列</a>' +
      '</div>' +
      '<div class="pv-tablewrap"><table class="pv-table xp-plan-table"><thead><tr>' +
        '<th>计划</th><th>投放目标</th><th>诊断状态</th><th>预算</th><th>出价</th>' +
        '<th class="ta-c">推广商品数</th><th>花费</th><th class="ta-c">全站商机量</th><th class="ta-c">优质商机量</th>' +
      '</tr></thead><tbody>' + PLANS.map(planRow).join('') + '</tbody></table></div>' +
      pagerHTML(16, 1, 2) +
      '</div>';
    return {
      html: html,
      acts: '<button class="pv-btn" type="button" data-pv-act="推广设置">推广设置</button>' +
        '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="新建推广计划">新建推广计划</button>'
    };
  }

  /* ---------- 二级页面 · 申诉中心 ---------- */
  var APPEALS = [
    { id: 168, obj: '用户', type: '违规类型', way: '首次违背价格虚假或扰乱MOQ违规/0分/警告', time: '2026-04-21 00:18:56', st: '申诉超时', cls: 'gray' }
  ];

  function pageAppeal() {
    var html = '<div class="xp-tipbar">后续快捷提交申诉，如在规定时间内提交，申诉入口将关闭。' +
      '<button class="xp-tipbar__x" type="button" data-pv-act="关闭提示">×</button></div>' +
      '<div class="pv-card">' +
        filtersHTML([
          { t: 'input', label: '违规编号', ph: '请输入' },
          { t: 'input', label: '申诉编号', ph: '请输入' },
          { t: 'select', label: '处理方式', opts: ['请选择处理方式', '警告', '扣分', '下架', '限制发布'] },
          { t: 'select', label: '申诉状态', opts: ['请选择', '待申诉', '申诉中', '申诉超时', '申诉成功', '申诉失败'] }
        ]) +
        '<div class="pv-tablewrap"><table class="pv-table"><thead><tr>' +
          '<th style="width:60px">ID</th><th>处理对象</th><th>违规类型</th><th>处理方式</th>' +
          '<th>处理时间</th><th>申诉状态</th><th style="width:120px">操作</th>' +
        '</tr></thead><tbody>' + APPEALS.map(function (a) {
          return '<tr><td class="pv-muted">' + a.id + '</td>' +
            '<td>' + a.obj + '</td><td>' + a.type + '</td>' +
            '<td class="pv-muted">' + a.way + '</td>' +
            '<td class="pv-muted">' + a.time + '</td>' +
            '<td>' + tagHTML(a.st, a.cls) + '</td>' +
            '<td><span class="pv-link--muted">申诉已关闭</span></td></tr>';
        }).join('') + '</tbody></table></div>' +
        pagerHTML(1, 1, 1) +
      '</div>';
    return { html: html };
  }

  /* ---------- 二级页面 · 生意助手概览 ---------- */
  var AI_TABS = ['素材创意', '数字员工', '智能发品', '智能接待', '智能分析', '私有化&Agent'];

  var AI_KPI = [
    { ico: 'i-goods', label: 'AI发品数量', v: '-', peer: '头部同行数据：60' },
    { ico: 'i-chart', label: 'AI商品曝光量', v: '5556', peer: '头部同行数据：109893' },
    { ico: 'i-chat', label: 'AI服务商机数', v: '0', peer: '头部同行数据：174' },
    { ico: 'i-growth', label: 'AI节约运营时间', v: '34<small>小时</small>', peer: '头部同行数据：229小时' }
  ];

  var AI_FEATS = [
    { t: '视频翻译', tag: '新', d: '一站式视频翻译，视频字幕翻译与视频语音翻译一步到位。' },
    { t: '营销视频', tag: '新', d: '快速生成爆款数字人视频，支持29种国家语言，适配多种社媒平台。' },
    { t: '营销图生成', tag: '新', d: '只需选择商品链接中的图片，生意助手即可理解产品信息、添加核心卖点，智能生成营销海报。' },
    { t: '原生视频', d: '只需选择商品，一键根据商品主图生成英文讲解视频。' },
    { t: '店铺装修', tag: '新', d: '智能读取店铺信息，匹配适合模板，一键智能装修店铺。' },
    { t: '图片翻译', d: '只需上传商品图片，即可将图片中的文字翻译成多种语言。' }
  ];

  function pageAiHome() {
    var html = heroHTML('近30天生意助手已实现店铺经营提效，请继续使用并尝试新功能',
        '使用AI功能已明显提升了店铺经营效率，带来额外商机。您还有 7083 豆，快去试试吧！',
        '<a class="pv-link" href="#">查看全部使用数据 ›</a>') +
      '<div class="xp-aikpis">' + AI_KPI.map(function (k) {
        return '<div class="xp-aikpi"><span class="xp-aikpi__ico"><svg class="ico"><use href="#' + k.ico + '"/></svg></span>' +
          '<p class="xp-aikpi__label">' + k.label + '</p>' +
          '<p class="xp-aikpi__value">' + k.v + '</p>' +
          '<p class="xp-aikpi__peer">' + k.peer + '</p></div>';
      }).join('') + '</div>' +
      '<div class="pv-card">' +
        sectionHead('全部功能', '产品说明文档 ›') +
        '<div class="xp-aichips">' + AI_TABS.map(function (t, i) {
          return '<button class="xp-aichip' + (i === 0 ? ' is-active' : '') + '" type="button">' + t + '</button>';
        }).join('') + '</div>' +
        '<div class="xp-feats">' + AI_FEATS.map(function (f) {
          return '<div class="xp-feat"><p class="xp-feat__t">' + f.t +
            (f.tag ? '<span class="xp-feat__tag">' + f.tag + '</span>' : '') + '</p>' +
            '<p class="xp-feat__d">' + f.d + '</p>' +
            '<div class="xp-feat__acts">' +
              '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="立即使用' + f.t + '">立即使用</button>' +
              '<button class="pv-btn" type="button" data-pv-act="使用说明">使用说明</button>' +
            '</div></div>';
        }).join('') + '</div>' +
      '</div>';
    return { html: html };
  }

  /* ---------- 二级页面 · 商家等级 ---------- */
  var LEVEL_DIMS = [
    { n: '店铺满意度', v: '4.9', unit: '分', d: '近 90 天买家评价均分' },
    { n: '交易规模', v: '0', unit: 'USD', d: '近 90 天实收金额' },
    { n: '服务能力', v: '100', unit: '%', d: '按时发货率' },
    { n: '商机转化', v: '20', unit: '%', d: '支付转化率' }
  ];

  var LEVEL_PERKS = [
    { ico: 'i-search', t: '搜索排序优先' },
    { ico: 'i-users', t: '高管买家触达' },
    { ico: 'i-badge', t: '优质服务直达' },
    { ico: 'i-robot', t: '生意助手优惠' }
  ];

  function pageGrowLevel() {
    var html = '<div class="xp-notice">本月数据以自然月统计，次月 5 日更新；店铺等级与权益以实际生效等级为准。' +
      '<a class="pv-link" href="#">查看更多 FAQ</a></div>' +
      '<div class="xp-levelnotice">' +
        '<p class="xp-levelnotice__t">3 月发布的二级类目「女装」排名评分：本月店铺诊断等级为 <b>2 星</b>，预计下月店铺等级 <b>2 星</b></p>' +
        '<p class="xp-levelnotice__d">提升方向：扩大交易规模 · 及时跟进买家 · 线上深度沟通 · 提升询盘转化 · 定期回访老客户</p>' +
        '<div class="xp-levelnotice__chips">' +
          '<button class="pv-chip" type="button" data-pv-act="查看优化建议">优化后与原来的变化</button>' +
          '<button class="pv-chip" type="button" data-pv-act="诊断分析">诊断分析</button>' +
        '</div>' +
      '</div>' +
      '<div class="pv-card xp-levelcard">' +
        '<div class="xp-levelcard__top">' +
          '<div class="xp-levelcard__l"><span class="xp-levelcard__cat">类目：女装</span>' +
            '<p class="xp-levelcard__lv"><b>等级</b> 2 星 <span class="xp-stars">★★</span></p></div>' +
          '<button class="xp-levelcard__btn" type="button" data-pv-act="查看权益">点击可解锁 2 星权益 查看详情 ›</button>' +
        '</div>' +
        '<div class="xp-perks">' + LEVEL_PERKS.map(function (p) {
          return '<div class="xp-perk"><span class="xp-perk__ico"><svg class="ico"><use href="#' + p.ico + '"/></svg></span>' +
            '<span>' + p.t + '</span></div>';
        }).join('') + '</div>' +
      '</div>' +
      '<div class="pv-card">' +
        sectionHead('店铺分（原商家分层）', '等级评定标准') +
        '<p class="xp-hint">店铺分由交易、服务、商机三部分构成，是等级评定的核心依据。' +
          '<button class="pv-link" type="button" data-pv-act="查看明细">查看明细</button></p>' +
        '<div class="xp-dims">' + LEVEL_DIMS.map(function (d) {
          return '<div class="xp-dim"><p class="xp-dim__n">' + d.n + '</p>' +
            '<p class="xp-dim__v">' + d.v + '<small>' + d.unit + '</small></p>' +
            '<p class="xp-dim__d">' + d.d + '</p></div>';
        }).join('') + '</div>' +
      '</div>' +
      '<div class="pv-card">' +
        sectionHead('交易赏罚（0 星）', '查看规则') +
        '<p class="xp-hint">统计时间：2026-11-07（PST）。近 90 天无违规扣分，保持良好交易记录可继续累积店铺分。</p>' +
        '<div class="xp-penalty"><span>交易通道：0 个</span><span>违规扣分：0 分</span>' +
          '<span>服务保障：已加入</span></div>' +
      '</div>';
    return {
      html: html,
      acts: '<button class="pv-btn" type="button" data-pv-act="等级说明">等级说明</button>' +
        '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="提升等级">提升等级</button>'
    };
  }

  /* ---------- 二级页面 · 认证首页（金品诚企） ---------- */
  var CERT_TABS = [
    { n: '管理体系证书', c: 0 },
    { n: '产品证书', c: 0 },
    { n: '产品检测报告', c: 0 }
  ];

  function pageVerifyHome() {
    var html = '<div class="pv-card xp-profile">' +
        '<p class="xp-profile__brand">金品诚企会员中心</p>' +
        '<div class="xp-profile__body">' +
          '<div class="xp-profile__left">' +
            '<span class="xp-profile__logo">B</span>' +
            '<div class="xp-profile__info">' +
              '<p class="xp-profile__name">Dongguan Bafinasa Garments Co., Ltd. <i class="xp-i">i</i></p>' +
              '<p class="xp-profile__badges">' +
                '<span class="xp-vbadge">Verified 2YRS</span>' +
                '<span class="xp-vbadge xp-vbadge--gold">实力工厂</span>' +
              '</p>' +
              '<p class="xp-profile__date">金品服务到期时间：2027-03-01</p>' +
              '<p class="xp-profile__stars">★★★★★ <b>5.0</b></p>' +
            '</div>' +
          '</div>' +
          '<a class="pv-link" href="#">进入权益中心 ›</a>' +
          '<div class="xp-profile__banner"><span class="xp-profile__bannert">金品商家<br>欧洲运营三步走</span></div>' +
        '</div>' +
      '</div>' +
      '<div class="pv-card">' +
        sectionHead('实力认证及展示', '') +
        '<p class="xp-hint">由第三方深度认证，在前台旺铺为买家全面展示企业实力。' +
          '<span class="xp-inline-note">最近一次金品认证完成时间 2026-03-02</span></p>' +
        '<div class="xp-certgrid">' +
          '<div class="xp-cert">' +
            '<div class="xp-cert__head"><span>资质证书验真</span>' +
              '<span class="xp-cert__links"><a class="pv-link" href="#">前往场景展示</a>' +
              '<a class="pv-link" href="#">上传证书</a><a class="pv-link" href="#">查看更多</a></span></div>' +
            '<p class="xp-cert__d">买家查验真实资质，提升信任转化</p>' +
            '<div class="xp-certs__nums"><span>已认证证书 <b>0</b></span><span>已购商品 <b>0</b></span></div>' +
            '<div class="xp-certtabs">' + CERT_TABS.map(function (t) {
              return '<span class="xp-certtab">' + t.n + '<b>' + t.c + '</b></span>';
            }).join('') + '</div>' +
            '<p class="xp-cert__empty">您暂时没有证书需要补充</p>' +
            '<p class="xp-cert__warn">已失效证书 <b>0</b> <span>影响商品 0</span></p>' +
          '</div>' +
          '<div class="xp-cert xp-cert--side">' +
            '<div class="xp-cert__head"><span>行业竞争力</span><a class="pv-link" href="#">查看场景</a></div>' +
            '<p class="xp-cert__d">针对行业买家特性，差异化展示企业竞争优势，高效获客</p>' +
            '<p class="xp-cert__big">0<small>已认证数</small></p>' +
            '<button class="pv-btn" type="button" data-pv-act="去认证">去认证</button>' +
          '</div>' +
          '<div class="xp-cert xp-cert--side">' +
            '<div class="xp-cert__head"><span>金品实力标签</span><a class="pv-link" href="#">查看场景</a></div>' +
            '<p class="xp-cert__d">认证过的商家实力信息将聚合到买家关注的商家标签</p>' +
            '<p class="xp-cert__big">16<small>/ 38 已获得数</small></p>' +
            '<button class="pv-btn" type="button" data-pv-act="去管理">去管理</button>' +
          '</div>' +
          '<div class="xp-cert">' +
            '<div class="xp-cert__head"><span>实力多媒体管理</span><a class="pv-link" href="#">去管理</a></div>' +
            '<p class="xp-cert__d">通过图片、视频等多媒体全方位向买家展示企业实力</p>' +
            '<div class="xp-media"><span class="xp-media__box">验厂视频</span><span class="xp-media__box">360全景图</span>' +
              '<span class="xp-media__box">实力相册</span></div>' +
          '</div>' +
        '</div>' +
      '</div>';
    return {
      html: html,
      acts: '<button class="pv-btn" type="button" data-pv-act="认证报告">认证报告</button>' +
        '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="申请深度认证">申请深度认证</button>'
    };
  }

  /* ---------- 二级页面 · 资金总览 ---------- */
  var FUND_STEPS = [
    { n: '买家付款', d: '创建订单后，分享给买家并引导买家通过平台付款', btn: '去使用' },
    { n: '收款', d: '关联创建的订单，资金自动入账', btn: '去关联' },
    { n: '提现', d: '需要结汇提现账户，结汇 / 原币提现', btn: '去提现' }
  ];

  function pageFundOverview() {
    var steps = '<div class="xp-steps">' + FUND_STEPS.map(function (s, i) {
      return '<div class="xp-step"><span class="xp-step__no">' + (i + 1) + '</span>' +
        '<div class="xp-step__box"><p class="xp-step__t">' + s.n + '</p>' +
        '<p class="xp-step__d">' + s.d + '</p></div>' +
        '<button class="pv-btn" type="button" data-pv-act="' + s.btn + '">' + s.btn + '</button></div>';
    }).join('') + '</div>';

    var banner = '<div class="xp-paybanner">' +
        '<span class="xp-paybanner__logo">Alibaba.com <b>Pay</b></span>' +
        '<span class="xp-paybanner__t">阿联酋收款 T+0 到账</span>' +
        '<span class="xp-paybanner__d">买家 AED 本币付，商家 USD T+0 到账，无中间行扣费</span>' +
        '<button class="xp-paybanner__btn" type="button" data-pv-act="立即开通">立即开通</button>' +
        '<span class="xp-paybanner__glow"></span></div>';

    var asset = '<div class="pv-card xp-asset">' +
      '<div><p class="xp-asset__label">我的账户资产 <i class="xp-i">i</i></p>' +
        '<p class="xp-asset__value">USD <b>2,276.98</b></p></div>' +
      '<button class="pv-btn" type="button" data-pv-act="收款账户">收款账户</button>' +
      '</div>';

    var collect = '<div class="pv-card xp-collect">' +
      '<div class="xp-collect__head"><span>国际站订单收款</span>' +
        '<span class="xp-collect__acts"><a class="pv-link" href="#">关联</a>' +
        '<a class="pv-link" href="#">付款</a><a class="pv-link" href="#">提现</a></span></div>' +
      '<p class="xp-hint">全部国际站订单收款 ›</p>' +
      '<table class="pv-table"><thead><tr><th>币种</th><th>待关联金额</th><th>余额</th><th style="width:80px">操作</th></tr></thead>' +
      '<tbody><tr><td><span class="xp-flag">US</span> USD 美元</td>' +
        '<td class="num num--plain">0.00</td><td class="num">2,276.98</td>' +
        '<td><a class="pv-link" href="#">详情 ›</a></td></tr></tbody></table>' +
      '</div>';

    var global = '<div class="pv-card xp-global">' +
      '<p class="xp-global__brand">Alibaba.com Pay <b>全球收款</b></p>' +
      '<p class="xp-global__t">中东买家 AED 直付，当天美元到账</p>' +
      '<p class="xp-global__d">中东大买家 AED 本地收款，卖家本币付款，减少中间环节，资金高效回笼</p>' +
      '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="立即开通">立即开通 ›</button>' +
      '</div>';

    var html = steps + banner + asset +
      '<div class="xp-split">' + collect + global + '</div>';
    return {
      html: html,
      acts: '<button class="pv-btn" type="button" data-pv-act="切换币种">USD</button>' +
        '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="收款账户">管理账户</button>'
    };
  }

  /* ---------- 二级页面 · 查询报价并下单（物流） ---------- */
  var LOGI_LINES = [
    { n: '阿里无忧物流-标准', t: '12-20 个工作日', p: '¥ 86.00', tag: '推荐', cls: 'green' },
    { n: '阿里无忧物流-优先', t: '7-12 个工作日', p: '¥ 132.50', tag: '时效快', cls: 'blue' },
    { n: '海运整柜（拼箱）', t: '28-35 天', p: '¥ 1,860.00', tag: '大货', cls: 'purple' }
  ];

  function pageLogiQuote() {
    var form = '<div class="pv-card">' +
      '<div class="xp-quote">' +
        '<div class="xp-quote__field"><label>发货地</label>' +
          '<select class="pv-select"><option>中国大陆 · 广东 · 东莞</option><option>中国大陆 · 浙江 · 杭州</option></select></div>' +
        '<div class="xp-quote__field"><label>收货地</label>' +
          '<select class="pv-select"><option>美国 · CA · Los Angeles</option><option>德国 · BE · Berlin</option></select></div>' +
        '<div class="xp-quote__field xp-quote__field--wide"><label>包裹信息</label>' +
          '<span class="xp-dims-in">' +
            '<input class="xp-num" type="text" value="10"><i>×</i>' +
            '<input class="xp-num" type="text" value="10"><i>×</i>' +
            '<input class="xp-num" type="text" value="10"><i class="xp-unit">CM</i>' +
            '<input class="xp-num" type="text" value="1"><i class="xp-unit">KG</i>' +
          '</span></div>' +
        '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="查询报价">查询</button>' +
      '</div>' +
      '<p class="xp-quote__hint">支持全球 200+ 国家与地区的线路报价；多包裹请使用「批量下单」。</p>' +
      '</div>';

    var list = '<div class="pv-card">' +
      sectionHead('推荐物流方案', '报价规则') +
      '<div class="pv-tablewrap"><table class="pv-table"><thead><tr>' +
        '<th>物流方案</th><th>参考时效</th><th>预估运费</th><th>说明</th><th style="width:120px">操作</th>' +
      '</tr></thead><tbody>' + LOGI_LINES.map(function (l) {
        return '<tr><td><b>' + l.n + '</b> ' + tagHTML(l.tag, l.cls) + '</td>' +
          '<td>' + l.t + '</td><td class="num">' + l.p + '</td>' +
          '<td class="pv-muted">门到门，含基础保险</td>' +
          '<td><a class="pv-link" href="#">立即下单</a></td></tr>';
      }).join('') + '</tbody></table></div></div>';

    var news = '<div class="pv-card xp-news"><div class="xp-news__head">物流公告' +
      '<a class="pv-link" href="#">更多</a></div>' +
      '<ul class="xp-news__list">' +
        '<li><b>新功能</b> AI 智能报价全新升级，支持目的地查询、自动匹配最优线路。<span>11-06</span></li>' +
        '<li><b>优化</b> 物流公告已移至页面左侧，点击即可随时查看平台最新通知。<span>11-05</span></li>' +
        '<li><b>提醒</b> 旺季舱位紧张，建议提前 3 天预约下单。<span>11-01</span></li>' +
      '</ul></div>';

    return {
      html: form + '<div class="xp-split xp-split--73">' + list + news + '</div>',
      acts: '<button class="pv-btn" type="button" data-pv-act="物流公告">物流公告</button>' +
        '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="批量下单">批量下单</button>'
    };
  }

  /* ---------- 二级页面 · 出口服务订单 ---------- */
  function pageExportOrders() {
    var tabs = [
      { k: 'edit', n: '待修改', c: 0 },
      { k: 'confirm', n: '待确认', c: 0 },
      { k: 'audit', n: '审核中', c: 0 },
      { k: 'checked', n: '已勾选', c: 0 },
      { k: 'cancel', n: '已取消', c: 0 }
    ];
    var html = '<div class="pv-card">' +
      '<div class="xp-expbar">' +
        '<div class="pv-tabs xp-tabs--plain">' + tabs.map(function (t, i) {
          return '<button class="pv-tab' + (i === 0 ? ' is-active' : '') + '" type="button" data-tabk="' + t.k + '">' +
            t.n + '<b>' + t.c + '</b></button>';
        }).join('') + '</div>' +
        '<div class="xp-expbar__acts">' +
          '<button class="pv-btn" type="button" data-pv-act="批量编辑">批量编辑</button>' +
          '<button class="pv-btn" type="button" data-pv-act="批量下载">批量下载</button>' +
        '</div>' +
      '</div>' +
      '<div class="xp-taxbanner">' +
        '<span class="xp-taxbanner__t">全球计税</span>' +
        '<span class="xp-taxbanner__d">智能测算美国税费并计算关税，效率提升 95% 以上</span>' +
        '<button class="xp-taxbanner__btn" type="button" data-pv-act="立即升级">立即升级</button>' +
      '</div>' +
      filtersHTML([
        { t: 'input', label: '搜索关键字', ph: '搜索关键字' },
        { t: 'input', label: '下单时间', ph: '开始日期 - 结束日期' },
        { t: 'select', label: '出口方式', opts: ['请选择', '一达通出口', '自营出口', '代理出口'] },
        { t: 'select', label: '服务范围', opts: ['请选择', '通关', '结汇', '退税', '通关+退税'] },
        { t: 'select', label: '服务进度', opts: ['请选择', '资料准备', '报关中', '已通关', '已退税'] },
        { t: 'select', label: '物流状态', opts: ['请选择', '未发货', '运输中', '已签收'] },
        { t: 'select', label: '报关状态', opts: ['请选择', '未报关', '待放行', '已放行'] },
        { t: 'select', label: '退税状态', opts: ['请选择', '未申报', '申报中', '已退税'] }
      ]) +
      '<div class="xp-checkchips">' +
        '<label class="xp-checkchip"><input class="pv-check" type="checkbox">待修改</label>' +
        '<label class="xp-checkchip"><input class="pv-check" type="checkbox">待确认资料</label>' +
        '<label class="xp-checkchip"><input class="pv-check" type="checkbox">24 小时内到期</label>' +
      '</div>' +
      '<div class="pv-tablewrap"><table class="pv-table xp-exptable"><thead><tr>' +
        '<th>基础信息</th><th>物流服务</th><th>报关服务</th><th>退税服务</th><th>资金服务</th><th style="width:90px">操作</th>' +
      '</tr></thead><tbody><tr><td colspan="6" class="xp-expempty">' +
        '<div class="pv-empty"><span class="pv-empty__ico"><svg class="ico"><use href="#i-truck"/></svg></span>' +
        '<p class="pv-empty__t1">暂无出口服务订单</p>' +
        '<p class="pv-empty__t2">下单后系统将自动生成出口服务订单，您可在此查看通关、退税与资金进度。</p>' +
        '<div class="pv-empty__acts"><button class="pv-btn pv-btn--pri" type="button" data-pv-act="立即下单">立即下单</button>' +
        '<button class="pv-btn" type="button" data-pv-act="查看下单指引">查看下单指引</button></div></div>' +
      '</td></tr></tbody></table></div>' +
      '</div>' +
      '<div class="pv-card">' +
        sectionHead('订单进度 · 签约', '帮助中心') +
        '<div class="xp-faq">' +
          '<div class="xp-faq__box"><p class="xp-faq__t">首次出口常见问题</p>' +
            '<ol class="xp-faq__list">' +
              '<li><a href="#">代办出口服务的账户需要满足哪些条件？</a></li>' +
              '<li><a href="#">在线下单需要开通并满足哪些前提条件？</a></li>' +
              '<li><a href="#">出口订单中的资料由谁提供？</a></li>' +
              '<li><a href="#">出口退税与开票需要准备哪些资料？</a></li>' +
            '</ol></div>' +
          '<div class="xp-faq__side"><p class="xp-faq__t">需要协助？</p>' +
            '<p class="xp-hint">专属顾问 1 对 1 协助您完成首单出口。</p>' +
            '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="联系顾问">联系顾问</button>' +
            '<button class="pv-btn" type="button" data-pv-act="新客户专属客服">新客户专属客服</button></div>' +
        '</div>' +
      '</div>';
    return {
      html: html,
      acts: '<button class="pv-btn" type="button" data-pv-act="导出">导出</button>' +
        '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="立即下单">立即下单</button>'
    };
  }

  /* ---------- 二级页面 · 交易总览 ---------- */
  var TRADE_TODO = [
    { n: '待确认', v: 0 }, { n: '待支付', v: 0 }, { n: '待发货', v: 0 },
    { n: '7天内需发货', v: 0 }, { n: '售后/退款', v: 0 }
  ];
  var TRADE_PERF = [
    { n: '实收金额(USD)', v: '0.00' }, { n: '实收订单数', v: '0' },
    { n: '支付转化率', v: '1.80%' }, { n: '支付率', v: '20.00%' }, { n: '评价分', v: '5' }
  ];
  var TRADE_SVC = [
    { n: '按时发货率', v: '100.00%' }, { n: '品质退款率', v: '0%' },
    { n: '5天发货率', v: '100.00%' }, { n: '纠纷升级率', v: '0%' }
  ];

  function gaugeHTML(pct, txt) {
    return '<div class="xp-gauge"><span class="xp-gauge__arc"></span>' +
      '<span class="xp-gauge__txt">' + (txt || '') + '</span>' +
      '<span class="xp-gauge__pct" style="--pct:' + pct + '%"></span></div>';
  }

  function pageTradeOverview() {
    var notice = '<div class="xp-tipbar xp-tipbar--blue">' +
      '<b>ACP+，阿里国际站订单的新选择：</b>ACP+ 把管理订单和收、付、汇资金能力整合在一起：14 个收汇币种、结汇不限额度、' +
      '订单 + 资金一个账户，还有新智惠订单 USD 10,000 以内跨境资金服务费全免。' +
      '<a class="pv-link" href="#">立即体验！</a></div>';

    var todo = '<div class="pv-card">' +
      sectionHead('待处理订单', '所有订单 ›') +
      '<div class="xp-todogrid">' + TRADE_TODO.map(function (t) {
        return '<div class="xp-todo"><p class="xp-todo__v">' + t.v + '</p>' +
          '<p class="xp-todo__n">' + t.n + '</p></div>';
      }).join('') + '</div>' +
      '</div>';

    var warn = '<div class="pv-card">' + sectionHead('预警信息', '查看详情 ›') +
      '<p class="xp-hint">仅显示审核通过的预警指标；指标统计口径为近 30 天，更新时间为 2026-11-07。</p>' +
      '<div class="xp-emptyline">∅ 暂无待处理的异常订单</div></div>';

    var data = '<div class="pv-card">' + sectionHead('交易数据', '查看详情 ›') +
      '<p class="xp-hint">衡量店铺交易能力的重要指标，核心数据对外披露可提升买家信赖与成交率。更新时间 2026-11-07（PST）</p>' +
      '<p class="xp-subhead">交易表现</p>' +
      '<div class="xp-perf">' + TRADE_PERF.map(function (p) {
        return '<div><p class="xp-perf__n">' + p.n + '</p><p class="xp-perf__v">' + p.v + '</p></div>';
      }).join('') + '</div>' +
      '<p class="xp-subhead">交易服务能力</p>' +
      '<div class="xp-perf xp-perf--4">' + TRADE_SVC.map(function (p) {
        return '<div><p class="xp-perf__n">' + p.n + '</p><p class="xp-perf__v">' + p.v + '</p></div>';
      }).join('') + '</div>' +
      '</div>';

    var credit = '<div class="pv-card xp-credit">' +
      '<div class="xp-credit__head"><span class="xp-credit__logo">¥</span>' +
        '<span>信用保障服务升级版</span><em class="xp-credit__badge">有效</em></div>' +
      '<div class="xp-credit__row"><span>信用保障总额度</span><b>USD 134,000.00</b></div>' +
      gaugeHTML(98, '可用额度<br><b>USD 132,162.87</b>') +
      '<div class="xp-credit__row"><span>拒付综合服务额度</span><b>USD 12,000.00</b></div>' +
      gaugeHTML(100, '可用额度<br><b>USD 12,000.00</b>') +
      '<button class="pv-btn xp-credit__btn" type="button" data-pv-act="查看全部保障服务">查看全部保障服务</button>' +
      '</div>';

    var tools = '<div class="pv-card">' + sectionHead('常用工具', '') +
      '<div class="xp-tools"><button class="xp-tool" type="button" data-pv-act="运费模板">' +
        '<span class="xp-tool__ico"><svg class="ico"><use href="#i-truck"/></svg></span>运费模板</button>' +
        '<button class="xp-tool" type="button" data-pv-act="起草订单">' +
        '<span class="xp-tool__ico"><svg class="ico"><use href="#i-cart"/></svg></span>起草订单</button>' +
        '<button class="xp-tool" type="button" data-pv-act="退款管理">' +
        '<span class="xp-tool__ico"><svg class="ico"><use href="#i-wallet"/></svg></span>退款管理</button>' +
      '</div></div>';

    var html = notice + todo + warn +
      '<div class="xp-split xp-split--73 xp-split--top">' +
        '<div class="xp-col">' + data + '</div>' +
        '<div class="xp-col">' + credit + tools + '</div>' +
      '</div>';
    return {
      html: html,
      acts: '<button class="pv-btn" type="button" data-pv-act="起草e收汇订单">起草 e 收汇订单</button>' +
        '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="起草信用保障订单">起草信用保障订单</button>'
    };
  }

  /* ---------- 二级页面 · 客户概览（客户通） ---------- */
  var CRM_BARS = [
    { n: '未标记', v: 3206 }, { n: '设备客户', v: 0 }, { n: '线索客户', v: 0 },
    { n: '成交客户', v: 24 }, { n: '意向客户', v: 17 }
  ];

  var CRM_L1 = [
    { t: 'L1+ 优质公海客户', d: '公海客户列表里打上 L1+ 的客户，可优先加入跟进。', name: 'Bull Line', lv: 'L1', btn: '加为我的客户' },
    { t: 'L1+ 优质客户流失预警', d: '您的成交客户近 7 天与平台其他商家发生询盘，有流失风险。', btn: '查看全部 ›' },
    { t: 'L1+ 优质客户问询提醒', d: '长期未跟进的小询盘，但您可能错失优质商机，建议尽快跟进。', btn: '查看全部 ›' }
  ];

  function pageCrmHome() {
    var notice = '<div class="xp-tipbar">本页面全店商家专属的「精准RFQ推荐」模块数据已不再更新，预计 2025 年 9 月 24 日下线，请各位商家知悉。' +
      '<button class="xp-tipbar__x" type="button" data-pv-act="关闭提示">×</button></div>';

    var max = 3206;
    var bars = '<div class="xp-chart">' + CRM_BARS.map(function (b) {
      var h = max ? Math.max(4, Math.round(b.v / max * 96)) : 4;
      return '<div class="xp-chart__col"><span class="xp-chart__v">' + (b.v || '') + '</span>' +
        '<i class="xp-chart__bar" style="height:' + h + 'px"></i>' +
        '<span class="xp-chart__n">' + b.n + '</span></div>';
    }).join('') + '</div>';

    var overview = '<div class="pv-card">' + sectionHead('客户概览', '') +
      '<div class="xp-crmnums">' +
        '<div><p class="xp-crmnums__n">本月新增</p><p class="xp-crmnums__v">0</p><p class="xp-crmnums__d">上月新增：0</p></div>' +
        '<div><p class="xp-crmnums__n">本月跟进</p><p class="xp-crmnums__v">0</p><p class="xp-crmnums__d">上月跟进：0</p></div>' +
        '<div><p class="xp-crmnums__n">本月成交</p><p class="xp-crmnums__v">0</p><p class="xp-crmnums__d">上月成交：0</p></div>' +
      '</div>' +
      '<div class="xp-crmtotal"><span>客户总数</span><b>2</b>' +
        '<span class="xp-crmtotal__sub">客户列表：0　公海客户：0</span></div>' +
      bars +
      '</div>';

    var dynamic = '<div class="pv-card">' + sectionHead('客户动态', '') +
      '<div class="xp-emptydyn"><span class="pv-empty__ico"><svg class="ico"><use href="#i-users"/></svg></span>' +
      '<p class="pv-empty__t1">暂无客户动态</p>' +
      '<p class="pv-empty__t2">客户在店铺的询盘、下单等行为将在这里实时展示。</p></div></div>';

    var l1 = '<div class="pv-card">' + sectionHead('L1+ 优质客户运营', '') +
      '<div class="xp-l1note">当前仅支持系统推送，今日无需要您跟进的客户。客户下单、反馈或询盘时，将在此提醒您。</div>' +
      '<div class="xp-l1">' + CRM_L1.map(function (c) {
        return '<div class="xp-l1card"><p class="xp-l1card__t">' + c.t + '</p>' +
          '<p class="xp-l1card__d">' + c.d + '</p>' +
          (c.name ? '<p class="xp-l1card__row"><span class="xp-l1card__name">' + c.name + '</span>' +
            tagHTML(c.lv, 'blue') + '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="' + c.btn + '">' + c.btn + '</button></p>'
            : '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="' + c.btn + '">' + c.btn + '</button>') +
          '</div>';
      }).join('') + '</div>' +
      '</div>';

    return {
      html: notice + '<div class="xp-split xp-split--2">' + overview + dynamic + '</div>' + l1,
      acts: '<a class="pv-btn" href="#">最新功能</a>' +
        '<a class="pv-btn" href="#">帮助文档</a>' +
        '<a class="pv-btn" href="#">客户通学院</a>' +
        '<a class="pv-btn" href="#">名词解释</a>'
    };
  }

  /* ---------- 二级页面 · 建设中 ---------- */
  function pageSoon(m, child) {
    var chips = (m.children || []).filter(function (c) { return c.key !== child.key; })
      .slice(0, 6).map(function (c) {
        return '<button class="pv-chip" type="button" data-open-page="' + c.key + '">' + c.name + '</button>';
      }).join('');
    return '<div class="pv-card"><div class="pv-empty">' +
      '<span class="pv-empty__ico"><svg class="ico"><use href="#' + m.icon + '"/></svg></span>' +
      '<p class="pv-empty__t1">' + child.name + ' · 模块建设中</p>' +
      '<p class="pv-empty__t2">该模块正在按平台最新版本迭代，接入后将提供完整的筛选、列表、批量操作与数据看板。' +
        '您可以先前往同组的其他功能，或订阅上线提醒。</p>' +
      (chips ? '<div class="pv-chips">' + chips + '</div>' : '') +
      '<div class="pv-empty__acts">' +
        '<button class="pv-btn pv-btn--pri" type="button" data-pv-act="订阅上线提醒">订阅上线提醒</button>' +
        '<button class="pv-btn" type="button" data-pv-gohome>返回工作台</button>' +
      '</div></div></div>';
  }

  var PAGES = {
    'customer-list': pageCustomer,
    'order': pageOrder,
    'onetouch': pageOnetouch,
    'mkt-site-promo': pagePlan,
    'crm-home': pageCrmHome,
    'trade-overview': pageTradeOverview,
    'export-orders': pageExportOrders,
    'logi-quote': pageLogiQuote,
    'fund-overview': pageFundOverview,
    'verify-home': pageVerifyHome,
    'grow-level': pageGrowLevel,
    'ai-home': pageAiHome,
    'svc-appeal': pageAppeal
  };

  /* 一级菜单默认进入「第一个已实现页面」，避免落到建设中 */
  function defaultPageFor(m) {
    var i;
    for (i = 0; i < m.children.length; i++) {
      if (PAGES[m.children[i].key]) return m.children[i].key;
    }
    return m.children[0].key;
  }

  function renderPage(m, child) {
    var body = document.getElementById('pageBody');
    var acts = document.getElementById('pageActs');
    if (!body || !acts) return;

    var build = PAGES[child.key];
    var out = build ? build() : { html: pageSoon(m, child) };
    body.innerHTML = out.html;
    acts.innerHTML = out.acts || '';
    bindPage(body);
  }

  function bindPage(root) {
    // 全选
    var all = root.querySelector('#pvAll');
    if (all) {
      all.addEventListener('change', function () {
        Array.prototype.forEach.call(root.querySelectorAll('.pv-rowchk'), function (c) {
          c.checked = all.checked;
        });
      });
    }
    // 重置筛选
    Array.prototype.forEach.call(root.querySelectorAll('[data-pv-reset]'), function (btn) {
      btn.addEventListener('click', function () {
        Array.prototype.forEach.call(root.querySelectorAll('.pv-input'), function (i) { i.value = ''; });
        Array.prototype.forEach.call(root.querySelectorAll('.pv-select'), function (s) { s.selectedIndex = 0; });
        toast('筛选条件已重置');
      });
    });
    // 状态标签页过滤
    var tabs = root.querySelectorAll('.pv-tab');
    Array.prototype.forEach.call(tabs, function (tab) {
      tab.addEventListener('click', function () {
        Array.prototype.forEach.call(tabs, function (t) { t.classList.remove('is-active'); });
        tab.classList.add('is-active');
        var k = tab.dataset.tabk;
        Array.prototype.forEach.call(root.querySelectorAll('.pv-table tbody tr'), function (tr) {
          var tags = (tr.dataset.tab || '').split(' ');
          tr.hidden = !(k === 'all' || tags.indexOf(k) >= 0);
        });
      });
    });
  }

  /* ---------- 渲染 ---------- */
  function renderMetrics(key) {
    var box = document.getElementById('metrics');
    if (!box) return;
    var list = METRICS[key] || METRICS.realtime;
    box.innerHTML = list.map(function (m) {
      var arrow = m.up ? '↑' : '↓';
      var cls = m.up ? 'up' : 'down';
      return '' +
        '<div class="metric">' +
          '<p class="metric__label">' + m.label + '</p>' +
          '<p class="metric__value">' +
            '<b>' + m.value + '</b>' +
            '<span class="metric__delta metric__delta--' + cls + '">' + arrow + ' ' + m.delta + '</span>' +
          '</p>' +
        '</div>';
    }).join('');
  }

  function renderRFQ() {
    var box = document.getElementById('rfq');
    if (!box) return;
    box.innerHTML = RFQ.map(function (r, i) {
      return '' +
        '<li class="rfq__row" data-idx="' + i + '">' +
          '<span class="flag flag--' + r.flag + '"></span>' +
          '<div class="rfq__main">' +
            '<a class="rfq__title" href="#">' + r.title + '</a>' +
            '<p class="rfq__sub">' + r.sub + '</p>' +
          '</div>' +
          '<span class="rfq__price">' + r.price + '</span>' +
          '<span class="rfq__time">' + r.time + '</span>' +
          '<span class="rfq__seats">' + r.seats + '</span>' +
          '<div class="rfq__acts">' +
            '<button class="btn btn--ghost" type="button" data-act="assist">辅助报价</button>' +
            '<button class="btn btn--solid" type="button" data-act="quote">去报价</button>' +
          '</div>' +
        '</li>';
    }).join('');
  }

  function renderTodo() {
    var box = document.getElementById('todo');
    if (!box) return;
    box.innerHTML = TODO.map(function (t) {
      var hot = t.value > 0 ? ' is-hot' : '';
      return '<p class="todo__item"><span>' + t.label + '</span><b class="' +
        hot.trim() + '">' + t.value + '</b></p>';
    }).join('');
  }

  /* ---------- 轻提示 ---------- */
  var toastTimer = null;
  function toast(msg) {
    var el = document.getElementById('toast');
    if (!el) return;
    el.textContent = msg;
    el.classList.add('is-show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () {
      el.classList.remove('is-show');
    }, 1800);
  }

  /* ---------- 事件 ---------- */
  function bind() {
    // 数据卡切换
    var tabs = document.querySelectorAll('.tabs__item');
    Array.prototype.forEach.call(tabs, function (tab) {
      tab.addEventListener('click', function () {
        Array.prototype.forEach.call(tabs, function (t) { t.classList.remove('is-active'); });
        tab.classList.add('is-active');
        renderMetrics(tab.dataset.tab);
      });
    });

    // 一级菜单（含「更多」下拉）/ 二级菜单（事件委托）
    document.addEventListener('click', function (e) {
      if (!e.target.closest) return;
      var more = document.getElementById('navMore');

      var moreBtn = e.target.closest('.nav__morebtn');
      if (moreBtn) {
        e.preventDefault();
        if (more) more.classList.toggle('is-open');
        return;
      }

      var dropItem = e.target.closest('.nav__dropitem');
      if (dropItem) {
        e.preventDefault();
        if (more) more.classList.remove('is-open');
        openGroup(dropItem.dataset.group);
        return;
      }

      var nav = e.target.closest('.nav__item');
      if (nav) {
        e.preventDefault();
        if (more) more.classList.remove('is-open');
        openGroup(nav.dataset.group);
        return;
      }

      if (more) more.classList.remove('is-open');

      var sub = e.target.closest('.subnav__item');
      if (sub) {
        e.preventDefault();
        openPage(sub.dataset.page);
      }
    });

    // 询盘操作
    document.addEventListener('click', function (e) {
      var btn = e.target.closest ? e.target.closest('[data-act]') : null;
      if (!btn) return;
      e.preventDefault();
      var row = btn.closest('.rfq__row');
      var idx = row ? Number(row.dataset.idx) : -1;
      var title = idx >= 0 && RFQ[idx] ? RFQ[idx].title : '';
      toast((btn.dataset.act === 'quote' ? '进入报价：' : '辅助报价：') + title);
    });

    // 二级页面：演示动作 / 返回工作台 / 同组跳转 / 分页
    document.addEventListener('click', function (e) {
      var t = e.target;
      if (!t || !t.closest) return;

      var act = t.closest('[data-pv-act]');
      if (act) {
        e.preventDefault();
        toast(act.dataset.pvAct + '（演示）');
        return;
      }

      var home = t.closest('[data-pv-gohome]');
      if (home) {
        e.preventDefault();
        openGroup('home');
        return;
      }

      var jump = t.closest('[data-open-page]');
      if (jump) {
        e.preventDefault();
        openPage(jump.dataset.openPage);
        return;
      }

      var nav = t.closest('[data-pv-nav]');
      if (nav) {
        e.preventDefault();
        toast(nav.dataset.pvNav === 'prev' ? '已经是第一页' : '已加载下一页（演示）');
        return;
      }

      var pg = t.closest('[data-pv-page]');
      if (pg) {
        e.preventDefault();
        var wrap = pg.parentNode;
        Array.prototype.forEach.call(wrap.querySelectorAll('.pv-page'), function (b) {
          if (b.dataset.pvPage) b.classList.remove('is-active');
        });
        pg.classList.add('is-active');
        toast('已切换到第 ' + pg.textContent + ' 页（演示）');
      }
    });

    // 演示动作
    document.addEventListener('click', function (e) {
      var el = e.target.closest ? e.target.closest('.ai__btn, .promo__btn') : null;
      if (!el) return;
      toast(el.classList.contains('ai__btn') ? 'Accio Work 即将开放体验' : '生意助手新版已开启');
    });
  }

  /* ---------- 启动 ---------- */
  renderNav();
  paintNav();
  layoutNav();
  window.addEventListener('resize', function () { layoutNav(); });
  renderMetrics('realtime');
  renderRFQ();
  renderTodo();
  bind();
})();

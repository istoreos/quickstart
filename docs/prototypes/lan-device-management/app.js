const variants = {
  A: '设备优先 + 详情抽屉（推荐）',
  B: '列表 + 常驻详情双栏',
  C: '任务引导首页'
};

const contexts = {
  main: {
    label: '主路由 · 本机 DHCP', topology: '主路由', dhcp: '本机负责', editable: true,
    lan: '192.168.1.1', upstream: 'WAN 自动获取', defaultRoute: '本机路由', confidence: '高'
  },
  'main-external': {
    label: '主路由拓扑 · 外部 DHCP', topology: '主路由', dhcp: '其他设备负责', editable: false,
    lan: '192.168.1.1', upstream: 'WAN 自动获取', defaultRoute: '本机路由', confidence: '中'
  },
  'side-local': {
    label: '旁路由 · 本机 DHCP', topology: '旁路由', dhcp: '本机负责', editable: true,
    lan: '192.168.1.2', upstream: '192.168.1.1', defaultRoute: '上级路由', confidence: '高'
  },
  'side-external': {
    label: '旁路由 · 外部 DHCP', topology: '旁路由', dhcp: '主路由负责', editable: false,
    lan: '192.168.1.2', upstream: '192.168.1.1', defaultRoute: '上级路由', confidence: '高'
  },
  ambiguous: {
    label: '角色不明 · 本机 DHCP', topology: '无法确定', dhcp: '本机负责', editable: true,
    lan: '192.168.1.1', upstream: '192.168.1.254', defaultRoute: '当前默认路线', confidence: '低'
  }
};

const capabilityScenarios = {
  all: { floating: 'available', speed: 'available' },
  'no-floating': { floating: 'not_installed', speed: 'available' },
  'no-speed': { floating: 'available', speed: 'not_installed' },
  none: { floating: 'not_installed', speed: 'not_installed' },
  error: { floating: 'error', speed: 'error' }
};

const brandIcons = { ASUS: '🖥️', Xiaomi: '📺', Apple: '📱', Huawei: '📡', Synology: '🗄️' };
const typeIcons = { 电脑: '💻', 手机: '📱', 电视: '📺', 游戏设备: '🎮', 智能家居: '🏠', 路由器: '📡', 存储设备: '🗄️', 打印机: '🖨️', 摄像头: '📷', 音箱: '🔊' };
const manualIcons = [
  ['💻','笔记本'],['🖥️','台式电脑'],['⌨️','工作站'],['📱','手机'],['⌚','手表'],['📟','手持设备'],
  ['📺','电视'],['📽️','投影'],['🔊','音箱'],['🎵','音乐设备'],['🎮','游戏机'],['🕹️','街机'],
  ['📡','路由器'],['🌐','网络设备'],['🛜','无线设备'],['🔌','网关'],['🗄️','存储'],['💾','服务器'],
  ['🏠','智能家居'],['💡','灯具'],['🔒','门锁'],['📷','摄像头'],['🌡️','传感器'],['🧹','清洁设备'],
  ['🖨️','打印机'],['📻','收音机'],['🚗','汽车'],['🚲','自行车'],['🤖','机器人'],['⚙️','其他设备']
];

const params = new URLSearchParams(location.search);
const requestedContext = params.get('context') || (params.get('role') === 'side' ? 'side-external' : params.get('role'));
const supportedModals = ['add-device', 'profile', 'network', 'limits', 'usage', 'group', 'context', 'dhcp', 'routes', 'floating', 'bandwidth', 'diagnostics', 'route-target', 'capability-setup'];
const initialModal = supportedModals.includes(params.get('modal')) ? params.get('modal') : null;
let variant = variants[params.get('variant')] ? params.get('variant') : 'A';
let screen = ['device', 'groups', 'network'].includes(params.get('screen')) ? params.get('screen') : 'device';
let contextKey = contexts[requestedContext] ? requestedContext : 'main';
let capabilityScenario = capabilityScenarios[params.get('caps')] ? params.get('caps') : 'all';
let filter = 'online';
let selectedDeviceId = ['asus', 'tv', 'unknown', 'planned'].includes(params.get('device')) ? params.get('device') : null;
let modalKind = null;
let pendingDraft = null;
let capabilityReturnModal = null;
const capabilityOverrides = {};
let toastTimer = null;

const prototypeState = {
  apply: 'applied',
  lastAction: '当前配置已生效'
};

const devices = [
  {
    id: 'asus', iconMode: 'auto', iconKey: '', name: '客厅 ASUS 电脑', brand: 'ASUS', type: '电脑', presence: 'online',
    access: 'allowed', address: '192.168.1.23', connection: '有线', speed: '↓2.4M · ↑80K',
    route: '客厅旁路由', routeKind: 'side', reserved: true, hostname: 'living-room-pc',
    limits: '下载 20 / 上传 5 Mbps', group: '家庭设备', schedule: '未设置', quota: '每月 100 GB · 已用 38%',
    manufacturer: 'ASUSTek COMPUTER INC.', confidence: '高', source: '厂商 + 用户修正',
    labels: ['经旁路由', '限速'], issue: '', today: '3.2 GB', history: ['192.168.1.23', '192.168.1.18']
  },
  {
    id: 'tv', iconMode: 'auto', iconKey: '', name: '客厅电视', brand: 'Xiaomi', type: '电视', presence: 'online',
    access: 'allowed', address: '192.168.1.31', connection: '无线', speed: '↓8.1M · ↑40K',
    route: '跟随网络默认', routeKind: 'default', reserved: false, hostname: '',
    limits: '不限速', group: '家庭设备', schedule: '未设置', quota: '未设置',
    manufacturer: 'Xiaomi Communications Co Ltd', confidence: '中', source: '厂商默认类型',
    labels: [], issue: '', today: '6.8 GB', history: ['192.168.1.31']
  },
  {
    id: 'unknown', iconMode: 'auto', iconKey: '', name: '未命名设备', brand: '未知品牌', type: '电脑', presence: 'offline',
    access: 'allowed', address: '192.168.1.52', connection: '未知', speed: '—',
    route: '跟随网络默认', routeKind: 'default', reserved: true, hostname: '',
    limits: '不限速', group: '未分组', schedule: '未设置', quota: '未设置',
    manufacturer: '—', confidence: '低', source: '安全回退类型',
    labels: ['地址冲突'], issue: '地址与另一台设备冲突', today: '—', history: ['192.168.1.52', '192.168.1.99']
  },
  {
    id: 'planned', iconMode: 'manual', iconKey: '🎮', name: '儿童游戏机', brand: '未知品牌', type: '游戏设备', presence: 'never',
    access: 'paused', address: '尚未分配', connection: '从未连接', speed: '—',
    route: '浮动网关', routeKind: 'floating', reserved: true, hostname: 'kids-console',
    limits: '下载 50 / 上传 10 Mbps', group: '儿童设备', schedule: '工作日 22:00 暂停', quota: '每月 80 GB',
    manufacturer: '尚未观察到', confidence: '—', source: '用户预先创建',
    labels: ['从未上线', '晚间暂停'], issue: '', today: '—', history: []
  }
];

const groups = [
  { id: 'kids', name: '儿童设备', count: 4, summary: '工作日 22:00—次日 07:00 暂停联网', route: '浮动网关', limits: '下载 50 / 上传 10 Mbps', quota: '每月 80 GB', state: '正常', exceptions: 1 },
  { id: 'iot', name: 'IoT 设备', count: 12, summary: '始终允许联网', route: 'IoT 旁路由', limits: '下载 10 / 上传 2 Mbps', quota: '未设置', state: '1 台等待更新', exceptions: 0 }
];

function escapeHtml(value) {
  return String(value ?? '').replace(/[&<>'"]/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[char]));
}

function currentContext() { return contexts[contextKey]; }
function selectedDevice() { return devices.find(device => device.id === selectedDeviceId) || devices[0]; }
function capabilityState(name) { return capabilityOverrides[name] || capabilityScenarios[capabilityScenario][name]; }
function capabilityText(name) {
  const state = capabilityState(name);
  return state === 'available' ? '可用' : state === 'disabled' ? '未启用' : state === 'not_installed' ? '未安装' : '状态异常';
}
function deviceIcon(device) {
  if (device.iconMode === 'manual' && device.iconKey) return device.iconKey;
  if (brandIcons[device.brand]) return brandIcons[device.brand];
  if (typeIcons[device.type]) return typeIcons[device.type];
  return '💻';
}
function iconSourceText(device) {
  if (device.iconMode === 'manual' && device.iconKey) return '手动选择';
  if (brandIcons[device.brand]) return `自动 · ${device.brand} 厂商视觉（原创）`;
  if (typeIcons[device.type]) return `自动 · ${device.type}类型`;
  return '自动 · 通用电脑';
}
function presenceLabel(value) { return value === 'online' ? '在线' : value === 'offline' ? '离线' : '从未上线'; }
function applyLabel() {
  return prototypeState.apply === 'pending' ? '已保存，等待设备更新' :
    prototypeState.apply === 'failed' ? '应用失败，原设置已恢复' : '当前配置已生效';
}

function showToast(message) {
  const toast = document.getElementById('toast');
  toast.textContent = message;
  toast.classList.add('open');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toast.classList.remove('open'), 2600);
}

function updateUrl() {
  const next = new URLSearchParams(location.search);
  next.delete('role');
  next.set('variant', variant);
  next.set('screen', screen);
  next.set('context', contextKey);
  next.set('caps', capabilityScenario);
  if (selectedDeviceId) next.set('device', selectedDeviceId); else next.delete('device');
  history.replaceState(null, '', `${location.pathname}?${next}`);
}

function renderStateStrip() {
  const context = currentContext();
  const stateClass = prototypeState.apply === 'pending' ? 'pending' : prototypeState.apply === 'failed' ? 'failed' : '';
  document.getElementById('prototype-state').innerHTML = `
    <span><b>原型状态</b>　${escapeHtml(context.topology)} · DHCP ${escapeHtml(context.dhcp)} · 设备路线${context.editable ? '可编辑' : '只读'}　｜　浮动网关${capabilityText('floating')} · 限速${capabilityText('speed')}　｜　${escapeHtml(applyLabel())}</span>
    <span>
      ${prototypeState.apply === 'pending' ? '<button class="button" data-action="simulate-renew">模拟设备续租</button>' : ''}
      ${prototypeState.apply === 'failed' ? '<button class="button" data-action="clear-failure">查看恢复结果</button>' : ''}
    </span>`;
  document.getElementById('prototype-state').className = `prototype-state ${stateClass}`;
}

function deviceRows() {
  const visible = devices.filter(device => filter === 'all' || filter === 'issues' && device.issue || filter === 'online' && device.presence === 'online' || filter === 'offline' && device.presence !== 'online');
  return `<table>
    <thead><tr><th>设备</th><th>状态</th><th>地址 / 连接</th><th>实时流量</th><th>当前设置</th></tr></thead>
    <tbody>${visible.map(device => `
      <tr data-action="open-device" data-device-id="${device.id}">
        <td><div class="device"><span class="device-icon">${deviceIcon(device)}</span><div><strong>${escapeHtml(device.name)}</strong><small>${escapeHtml(device.brand)} · ${escapeHtml(device.type)}</small></div></div></td>
        <td><span class="status ${device.presence === 'online' ? '' : 'offline'}">${presenceLabel(device.presence)}</span>${device.access === 'paused' ? '<br><span class="tag warn">已暂停联网</span>' : ''}</td>
        <td>${escapeHtml(device.address)}<br><span class="muted">${escapeHtml(device.connection)}</span></td>
        <td>${escapeHtml(device.speed)}</td>
        <td>${device.issue ? `<span class="tag warn">${escapeHtml(device.issue)}</span>` : device.labels.length ? device.labels.slice(0, 2).map(label => `<span class="tag">${escapeHtml(label)}</span>`).join('') : '<span class="muted">默认</span>'}</td>
      </tr>`).join('') || '<tr><td colspan="5" class="muted">当前筛选下没有设备</td></tr>'}</tbody>
  </table>`;
}

function toolbar() {
  const online = devices.filter(device => device.presence === 'online').length;
  const issues = devices.filter(device => device.issue).length;
  const filterButton = (value, label) => `<button data-filter="${value}" class="${filter === value ? 'active' : ''}">${label}</button>`;
  return `<div class="summary"><span><b>${online}</b> 在线</span><span><b>${devices.length}</b> 全部</span><span><b>${issues}</b> 需要处理</span></div>
    <div class="toolbar"><input class="search" placeholder="搜索名称、品牌、IP 或 MAC"><div class="segmented">${filterButton('online', '在线')}${filterButton('all', '全部')}${filterButton('offline', '离线')}${filterButton('issues', '需要处理')}</div><button class="button" data-action="refresh">↻</button></div>`;
}

function renderDeviceA() {
  return `<div class="header"><div><h2>设备</h2><p class="muted">先找到设备，再管理它怎样联网</p></div></div>${toolbar()}${deviceRows()}
    <p class="coverage-note">列表只突出身份、存在状态、实时概况和非默认结果；完整配置集中在设备详情。</p>`;
}

function renderDeviceB() {
  const device = devices[0];
  return `<div class="header"><div><h2>设备工作台</h2><p class="muted">适合频繁管理多台设备；移动端仍回到单列详情</p></div></div>${toolbar()}
    <div class="split"><div class="split-list">${devices.map((item, index) => `<div class="compact-device ${index === 0 ? 'selected' : ''}" data-action="open-device" data-device-id="${item.id}"><div class="device"><span class="device-icon">${deviceIcon(item)}</span><div><strong>${escapeHtml(item.name)}</strong><small>${escapeHtml(item.address)} · ${presenceLabel(item.presence)}</small></div></div>${item.labels[0] ? `<span class="tag ${item.issue ? 'warn' : ''}">${escapeHtml(item.labels[0])}</span>` : ''}</div>`).join('')}</div>
      <aside class="split-detail"><h3>${escapeHtml(device.name)}</h3><p class="muted">${escapeHtml(device.brand)} · ${escapeHtml(device.type)} · ${presenceLabel(device.presence)}</p>
        <section class="section"><h3>网络与上网</h3><div class="row"><span>地址</span><span>${escapeHtml(device.address)}</span></div><div class="row"><span>路线</span><b>${escapeHtml(device.route)}</b></div><div class="row"><span>DNS</span><span>跟随路线</span></div><button class="button" data-action="open-device" data-device-id="${device.id}">完整设置</button></section>
        <section class="section"><h3>使用限制</h3><div class="row"><span>限速</span><span>${escapeHtml(device.limits)}</span></div><div class="row"><span>流量额度</span><span>${escapeHtml(device.quota)}</span></div></section>
      </aside></div>`;
}

function renderDeviceC() {
  return `<div class="header"><div><h2>今天想做什么？</h2><p class="muted">从常见任务开始，下面仍保留最近活跃设备</p></div></div>
    <div class="task-grid">
      <button class="task" data-action="task-online"><b>查看在线设备</b><span>快速识别当前使用网络的设备</span></button>
      <button class="task" data-action="open-device" data-device-id="asus"><b>指定上网路线</b><span>使用旁路由、浮动网关或指定 IP</span></button>
      <button class="task" data-action="open-limits" data-device-id="asus"><b>限制设备使用</b><span>暂停联网、限速、时段或流量额度</span></button>
    </div><h3>最近活跃</h3>${deviceRows()}`;
}

function renderGroups() {
  return `<div class="header"><div><h2>分组与计划</h2><p class="muted">批量管理设备，先展示最终结果，再按需查看规则来源</p></div><button class="button primary" data-action="new-group">新建分组</button></div>
    <div class="notice"><b>全局默认：</b>允许联网 · 不限速 · 跟随网络默认路线。明确的单设备设置会显示为例外。</div>
    <div class="cards">${groups.map(group => `<article class="card"><div class="card-top"><div><h3>${escapeHtml(group.name)}</h3><p class="muted">${group.count} 台设备 · ${escapeHtml(group.state)}</p></div><button class="button" data-action="manage-group" data-group-id="${group.id}">管理</button></div>
      <div class="facts"><div class="fact"><small>当前结果</small><strong>${escapeHtml(group.summary)}</strong></div><div class="fact"><small>上网路线</small><strong>${escapeHtml(group.route)}</strong></div><div class="fact"><small>限速</small><strong>${escapeHtml(group.limits)}</strong></div><div class="fact"><small>流量额度</small><strong>${escapeHtml(group.quota)}</strong></div></div>
      ${group.route === '浮动网关' && capabilityState('floating') !== 'available' ? `<p class="notice">浮动网关${capabilityText('floating')}；保留分组期望路线，但当前无法执行。</p>` : ''}
      ${group.limits !== '不限速' && capabilityState('speed') !== 'available' ? `<p class="notice">设备限速${capabilityText('speed')}；联网时段和其他规则继续执行。</p>` : ''}
      ${group.exceptions ? `<p class="notice">${group.exceptions} 台设备使用单独设置，不会被静默覆盖。</p>` : ''}</article>`).join('')}</div>`;
}

function renderNetwork() {
  const context = currentContext();
  return `<div class="header"><div><h2>局域网设置</h2><p class="muted">低频、高影响设置；修改前会说明影响范围和恢复方式</p></div></div>
    ${!context.editable ? `<div class="notice"><b>DHCP 由主路由负责。</b><br>本机可以观察设备，但不能给设备下发地址或上网路线。需要分流时，请在主路由使用本机地址 ${context.lan}。</div>` : ''}
    ${context.topology === '无法确定' ? '<div class="notice"><b>当前网络角色无法可靠判断。</b><br>设备路线仍可编辑，因为 DHCP 由本机负责；建议确认拓扑以获得更准确的默认推荐。</div>' : ''}
    <div class="settings-list">
      <article class="setting"><div><h3>网络概况</h3><p>本机角色：${context.topology} · DHCP：${context.dhcp} · 设备路线：${context.editable ? '可配置' : '只读'}</p></div><button class="button" data-action="open-context">查看</button></article>
      <article class="setting"><div><h3>地址分配</h3><p>${context.editable ? 'DHCP 已开启 · 192.168.1.100—249 · 已分配 18 个地址' : '本机 DHCP 已关闭 · 地址由其他设备分配'}</p></div><button class="button" data-action="open-dhcp">设置</button></article>
      <article class="setting"><div><h3>上网路线</h3><p>默认：${context.defaultRoute} · 旁路由 2 个 · 浮动网关${capabilityState('floating') === 'available' ? ' 1 个' : ` ${capabilityText('floating')}`}</p></div><button class="button" data-action="open-routes">管理</button></article>
      <article class="setting"><div><h3>带宽与统计</h3><p>总带宽 1000 / 100 Mbps · 保留 30 天 · 统计正常</p></div><button class="button" data-action="open-bandwidth">设置</button></article>
      <article class="setting"><div><h3>诊断与高级</h3><p>需要处理 2 项 · 最近变更、探测、通知、备份与导出</p></div><button class="button" data-action="open-diagnostics">查看</button></article>
    </div>`;
}

function renderDrawer() {
  if (!selectedDeviceId) return;
  const device = selectedDevice();
  const context = currentContext();
  const applyClass = prototypeState.apply === 'pending' ? 'pending' : prototypeState.apply === 'failed' ? 'failed' : '';
  document.getElementById('drawer-content').innerHTML = `
    <div class="drawer-head"><div class="device"><span class="device-icon">${deviceIcon(device)}</span><div><h2>${escapeHtml(device.name)}</h2><small>${escapeHtml(device.brand)} · ${escapeHtml(device.type)} · ${presenceLabel(device.presence)}</small></div></div><button class="button quiet" data-action="close-drawer">关闭</button></div>
    <div class="quick-actions"><button class="button ${device.access === 'paused' ? 'primary' : 'danger'}" data-action="toggle-access">${device.access === 'paused' ? '恢复联网' : '暂停联网'}</button><button class="button" data-action="edit-profile">编辑设备</button><button class="button" data-action="open-usage">查看用量</button></div>
    <div class="status-line ${applyClass}"><span><b>${escapeHtml(applyLabel())}</b><br><span>${escapeHtml(prototypeState.lastAction)}</span></span>${prototypeState.apply === 'pending' ? '<button class="button small" data-action="simulate-renew">模拟续租</button>' : ''}</div>
    ${device.issue ? `<div class="notice"><b>需要处理：${escapeHtml(device.issue)}</b><br>建议为其中一台设备重新选择地址，保存前会再次检查冲突。 <button class="button small" data-action="edit-network">处理</button></div>` : ''}
    ${!context.editable ? `<div class="notice"><b>上网路线只读</b><br>设备地址由主路由分配。请在主路由中把该设备的网关和 DNS 设置为 <b>${context.lan}</b>。 <button class="button small" data-action="copy-gateway">复制本机地址</button></div>` : ''}
    ${device.routeKind === 'floating' && capabilityState('floating') !== 'available' ? `<div class="notice"><b>浮动网关当前无法执行</b><br>保留原设置，但不会假装已生效。可以安装能力或为设备选择其他路线。 <button class="button small" data-action="setup-capability" data-capability="floating">${capabilityState('floating') === 'error' ? '重试与诊断' : '了解并安装'}</button></div>` : ''}
    ${device.limits !== '不限速' && capabilityState('speed') !== 'available' ? `<div class="notice"><b>设备限速当前无法执行</b><br>保留原限速值，联网权限和其他设置不受影响。 <button class="button small" data-action="setup-capability" data-capability="speed">${capabilityState('speed') === 'error' ? '重试与诊断' : '了解并安装'}</button></div>` : ''}
    <section class="section"><div class="section-head"><h3>设备信息</h3><button class="button small" data-action="edit-profile">编辑</button></div>
      <div class="row"><span>设备备注名</span><b>${escapeHtml(device.name)}</b></div><div class="row"><span>设备图标</span><span>${deviceIcon(device)} · ${escapeHtml(iconSourceText(device))}</span></div><div class="row"><span>存在状态</span><span>${presenceLabel(device.presence)}</span></div><div class="row"><span>联网权限</span><span>${device.access === 'paused' ? '暂停联网' : '允许联网'}</span></div><div class="row"><span>连接</span><span>${escapeHtml(device.connection)}</span></div><div class="row"><span>当前地址</span><span>${escapeHtml(device.address)}</span></div>
    </section>
    <section class="section"><div class="section-head"><h3>网络与上网</h3><button class="button small" data-action="edit-network" ${context.editable ? '' : 'disabled'}>${context.editable ? '编辑' : '只读'}</button></div>
      <div class="row"><span>地址</span><span>${device.reserved ? `保留 ${escapeHtml(device.address)}` : '自动获取'}</span></div><div class="row"><span>上网路线</span><b>${escapeHtml(device.route)}</b></div><div class="row"><span>DNS</span><span>跟随路线</span></div><div class="row"><span>网络主机名</span><span>${escapeHtml(device.hostname || '未设置')}</span></div>
    </section>
    <section class="section"><div class="section-head"><h3>使用限制</h3><button class="button small" data-action="open-limits">编辑</button></div>
      <div class="row"><span>限速</span><span>${escapeHtml(device.limits)}</span></div><div class="row"><span>设备分组</span><span>${escapeHtml(device.group)}</span></div><div class="row"><span>上网时段</span><span>${escapeHtml(device.schedule)}</span></div><div class="row"><span>流量额度</span><span>${escapeHtml(device.quota)}</span></div>
      <div class="source-chain"><span>当前结果来源</span><b>网络默认</b><span>→</span><b>${escapeHtml(device.group)}</b><span>→</span><b>单设备设置</b></div>
    </section>
    <section class="section"><div class="section-head"><h3>用量与诊断</h3><button class="button small" data-action="open-usage">查看详情</button></div>
      <div class="row"><span>当前速度</span><span>${escapeHtml(device.speed)}</span></div><div class="row"><span>今日用量</span><span>${escapeHtml(device.today)}</span></div><div class="row"><span>最近出现</span><span>${device.presence === 'online' ? '刚刚' : device.presence === 'never' ? '从未出现' : '2 小时前'}</span></div>
    </section>`;
}

function modalFrame(title, subtitle, body, footer = '', wide = false) {
  document.getElementById('modal-content').className = `modal ${wide ? 'wide' : ''}`;
  document.getElementById('modal-content').innerHTML = `<div class="modal-head"><div><h2 id="modal-title">${title}</h2><p class="muted">${subtitle}</p></div><button class="button quiet" data-action="close-modal">关闭</button></div>${body}${footer ? `<div class="modal-footer">${footer}</div>` : ''}`;
  document.getElementById('modal-backdrop').classList.add('open');
}

function openModal(kind, targetId) {
  modalKind = kind;
  if (kind !== 'preview') {
    const next = new URLSearchParams(location.search);
    next.set('modal', kind);
    if (kind === 'group' && targetId) next.set('group', targetId); else next.delete('group');
    if (kind === 'capability-setup' && targetId) next.set('capability', targetId); else next.delete('capability');
    history.replaceState(null, '', `${location.pathname}?${next}`);
  }
  let device = selectedDevice();
  if (targetId && ['profile', 'network', 'limits', 'usage'].includes(kind)) {
    device = devices.find(item => item.id === targetId) || device;
    selectedDeviceId = device.id;
  }
  if (kind === 'add-device') {
    modalFrame('添加设备', '只凭 MAC 也可以先建立管理项，设备上线后自动补充信息。', `
      <div class="form-grid"><label class="field full">设备备注名<input id="add-name" value="新设备" maxlength="64"><span class="field-hint">支持中文，只保存在 Quickstart 设备资料中。</span></label><label class="field full">MAC 地址<input id="add-mac" value="AA:BB:CC:DD:EE:FF"><span class="field-hint">保存后显示为“从未上线”。</span></label><label class="field">设备类型<select id="add-type"><option>电脑</option><option>手机</option><option>电视</option><option>游戏设备</option><option>智能家居</option></select></label><label class="field">加入分组<select><option>未分组</option><option>家庭设备</option><option>儿童设备</option><option>IoT 设备</option></select></label></div>`, '<button class="button" data-action="close-modal">取消</button><button class="button primary" data-action="save-new-device">添加设备</button>');
  } else if (kind === 'profile') {
    modalFrame('编辑设备资料', '备注名只用于识别设备，不会写入 DHCP 或触发 dnsmasq。', `
      <div class="form-grid"><label class="field full">设备备注名<input id="profile-name" value="${escapeHtml(device.name)}" maxlength="64"><span class="field-hint">支持中文和 Unicode。</span></label><label class="field">品牌<input id="profile-brand" value="${escapeHtml(device.brand)}"></label><label class="field">设备类型<select id="profile-type">${['电脑','手机','电视','游戏设备','智能家居','路由器','存储设备','打印机','摄像头','音箱'].map(type => `<option ${type === device.type ? 'selected' : ''}>${type}</option>`).join('')}</select></label></div>
      <h3 style="margin-top:18px">设备图标</h3><input id="profile-icon-mode" type="hidden" value="${escapeHtml(device.iconMode || 'auto')}"><input id="profile-icon-key" type="hidden" value="${escapeHtml(device.iconKey || '')}">
      <div class="icon-current"><span class="icon-preview" id="profile-icon-preview">${deviceIcon(device)}</span><div><b id="profile-icon-source">${escapeHtml(iconSourceText(device))}</b><br><span class="muted">图标帮助快速识别，准确厂商仍以品牌文字为准。</span></div></div>
      <button class="button ${device.iconMode !== 'manual' ? 'primary' : ''}" data-action="icon-auto">使用自动推荐</button>
      <details open><summary>从图标库手动选择</summary><div class="icon-grid">${manualIcons.map(([icon,label]) => `<button class="icon-option ${device.iconMode === 'manual' && device.iconKey === icon ? 'selected' : ''}" title="${label}" aria-label="${label}" data-action="select-icon" data-icon-key="${icon}">${icon}</button>`).join('')}</div><p class="field-hint">正式资源将按电脑、移动设备、影音、游戏、网络、智能家居和其他分组；只使用原创或明确授权资产。</p></details>
      <div class="preview"><b>原始识别不会被删除</b><br><span class="muted">诊断信息仍保留原始厂商、识别依据和可信度。</span></div>`, '<button class="button" data-action="close-modal">取消</button><button class="button primary" data-action="save-profile">保存</button>');
  } else if (kind === 'network') {
    const floatingAvailable = capabilityState('floating') === 'available';
    modalFrame('网络与上网', '地址与路线在同一次预览中保存，避免出现部分成功。', `
      <h3>地址</h3><label class="choice"><input type="radio" name="address-mode" value="auto" ${device.reserved ? '' : 'checked'}> 自动获取</label><label class="choice"><input type="radio" name="address-mode" value="reserved" ${device.reserved ? 'checked' : ''}> 保留当前地址 <span class="muted">· ${escapeHtml(device.address)}</span></label><label class="choice"><input type="radio" name="address-mode" value="custom"> 指定地址 <input id="custom-address" value="${device.address.startsWith('192.') ? device.address : '192.168.1.60'}"></label>
      <h3 style="margin-top:18px">上网路线</h3>${[['default','跟随网络默认','本机路由'],['self','本机路由','192.168.1.1'],['upstream','上级路由','192.168.1.254'],['side','客厅旁路由','192.168.1.2'],['floating','浮动网关',floatingAvailable ? '192.168.1.3 · 正常' : `需要${capabilityState('floating') === 'error' ? '恢复能力' : '安装能力'}`],['custom','指定 IP','手动输入']].map(([value,label,detail]) => `<label class="choice"><input type="radio" name="route" value="${value}" ${value === device.routeKind ? 'checked' : ''} ${value === 'floating' && !floatingAvailable ? 'disabled' : ''}> ${label} <span class="muted">· ${detail}</span></label>`).join('')}
      ${!floatingAvailable ? `<div class="capability-inline"><div><b>浮动网关${capabilityText('floating')}</b><p>其他路线仍可正常选择和保存。</p></div><button class="button small" data-action="setup-capability" data-capability="floating">${capabilityState('floating') === 'error' ? '重试与诊断' : '了解并安装'}</button></div>` : ''}
      <label class="field" style="margin-top:10px">指定路线 IP<input id="custom-route" value="192.168.1.8"></label><div class="preview"><b>DNS 跟随路线</b><br><span class="muted">所选路线地址也会作为 DNS 下发。本阶段不提供单独 DNS 选项。</span></div>
      <details><summary>高级设置</summary><label class="field" style="margin-top:10px">网络主机名<input id="network-hostname" value="${escapeHtml(device.hostname)}"><span class="field-hint">用于局域网名称解析，仅支持英文、数字和中间连字符；不支持中文、空格、点或下划线。</span><span id="hostname-error" class="field-error"></span></label></details>`, '<button class="button" data-action="close-modal">取消</button><button class="button primary" data-action="preview-network">预览并保存</button>', true);
  } else if (kind === 'limits') {
    const speedAvailable = capabilityState('speed') === 'available';
    modalFrame('使用限制', '围绕设备设置联网权限、限速、时段和流量额度。', `
      <div class="form-grid"><label class="field">联网权限<select id="limit-access"><option value="allowed" ${device.access === 'allowed' ? 'selected' : ''}>允许联网</option><option value="paused" ${device.access === 'paused' ? 'selected' : ''}>暂停联网</option></select></label><label class="field">设备分组<select id="limit-group">${['未分组','家庭设备','儿童设备','IoT 设备'].map(group => `<option ${group === device.group ? 'selected' : ''}>${group}</option>`).join('')}</select></label><label class="field">下载上限 Mbps<input id="limit-down" type="number" min="0" value="20" ${speedAvailable ? '' : 'disabled'}><span class="field-hint">0 表示不限速</span></label><label class="field">上传上限 Mbps<input id="limit-up" type="number" min="0" value="5" ${speedAvailable ? '' : 'disabled'}><span class="field-hint">0 表示不限速</span></label><label class="field">暂停开始时间<input id="schedule-start" type="time" value="22:00"></label><label class="field">暂停结束时间<input id="schedule-end" type="time" value="07:00"></label><label class="field">每月流量额度 GB<input id="quota-value" type="number" min="0" value="100"></label><label class="field">超限后<select id="quota-action"><option>暂停联网</option><option>仅提醒</option>${speedAvailable ? '<option>限速到 1 Mbps</option>' : ''}</select></label></div>
      ${!speedAvailable ? `<div class="capability-inline"><div><b>设备限速${capabilityText('speed')}</b><p>联网权限、分组、上网时段和额度仍可编辑。</p></div><button class="button small" data-action="setup-capability" data-capability="speed">${capabilityState('speed') === 'error' ? '重试与诊断' : '了解并安装'}</button></div>` : ''}
      <div class="preview"><b>当前结果来源</b><div class="source-chain"><b>网络默认</b><span>→</span><b>${escapeHtml(device.group)}</b><span>→</span><b>本次单设备设置</b></div></div>`, '<button class="button" data-action="close-modal">取消</button><button class="button primary" data-action="preview-limits">预览并保存</button>');
  } else if (kind === 'usage') {
    modalFrame('用量与诊断', '详细信息按需加载，不影响设备列表和基础控制。', `
      <div class="facts"><div class="fact"><small>当前速度</small><strong>${escapeHtml(device.speed)}</strong></div><div class="fact"><small>今日用量</small><strong>${escapeHtml(device.today)}</strong></div><div class="fact"><small>本月额度</small><strong>${escapeHtml(device.quota)}</strong></div><div class="fact"><small>策略状态</small><strong>${escapeHtml(applyLabel())}</strong></div></div>
      <div class="chart" aria-label="最近 12 小时流量趋势">${[22,35,18,48,62,35,76,88,55,34,67,42].map(height => `<i style="height:${height}%"></i>`).join('')}</div><p class="muted">最近 12 小时流量趋势（原型数据）</p>
      <h3>设备识别</h3><div class="row"><span>原始厂商</span><span>${escapeHtml(device.manufacturer)}</span></div><div class="row"><span>识别依据</span><span>${escapeHtml(device.source)}</span></div><div class="row"><span>识别可信度</span><span>${escapeHtml(device.confidence)}</span></div>
      <h3 style="margin-top:18px">地址记录</h3>${device.history.length ? device.history.map((address,index) => `<div class="row"><span>${index === 0 ? '当前地址' : '历史地址'}</span><span>${escapeHtml(address)}</span></div>`).join('') : '<p class="muted">设备尚未上线，没有地址记录。</p>'}
      <details><summary>查看策略来源和最近变更</summary><div class="source-chain"><b>网络默认</b><span>→</span><b>${escapeHtml(device.group)}</b><span>→</span><b>单设备例外</b></div><div class="timeline"><div class="timeline-item"><time>今天 10:32</time><span>上网路线改为“${escapeHtml(device.route)}”</span></div><div class="timeline-item"><time>昨天 21:00</time><span>限速策略自动执行</span></div></div></details>`, '<button class="button" data-action="close-modal">完成</button>', true);
  } else if (kind === 'group') {
    const group = groups.find(item => item.id === targetId) || { name: '新分组', route: '跟随网络默认', limits: '不限速', quota: '未设置', summary: '未设置计划', exceptions: 0 };
    modalFrame(group.id ? `管理“${group.name}”` : '新建分组', '选择设备并设置共同规则；单设备例外不会被静默覆盖。', `
      <label class="field">分组名称<input id="group-name" value="${escapeHtml(group.name)}"></label><h3 style="margin-top:18px">选择设备</h3><div class="check-grid">${devices.map((item,index) => `<label class="check-card"><input type="checkbox" ${index < 2 ? 'checked' : ''}><span><b>${escapeHtml(item.name)}</b><br><span class="muted">${escapeHtml(item.address)}</span></span></label>`).join('')}</div>
      <h3 style="margin-top:18px">共同规则</h3><div class="form-grid"><label class="field">上网路线<select><option>${escapeHtml(group.route)}</option><option>跟随网络默认</option><option>客厅旁路由</option><option ${capabilityState('floating') === 'available' ? '' : 'disabled'}>浮动网关${capabilityState('floating') === 'available' ? '' : '（需要安装）'}</option></select></label><label class="field">联网权限<select><option>允许联网</option><option>暂停联网</option></select></label><label class="field">暂停开始<input type="time" value="22:00"></label><label class="field">暂停结束<input type="time" value="07:00"></label><label class="field">下载 / 上传限速<input value="20 / 5 Mbps" ${capabilityState('speed') === 'available' ? '' : 'disabled'}></label><label class="field">每月流量额度<input value="80 GB"></label></div>
      ${capabilityState('floating') !== 'available' || capabilityState('speed') !== 'available' ? '<div class="notice">不可用能力只锁定相关字段；其他分组规则仍可编辑。选择安装后应保留当前草稿。</div>' : ''}
      ${group.exceptions ? `<div class="notice"><b>${group.exceptions} 台设备存在单独设置。</b><br>保存不会覆盖这些例外；可以稍后进入对应设备详情取消。</div>` : ''}`, '<button class="button" data-action="close-modal">取消</button><button class="button primary" data-action="preview-group">预览受影响设备</button>', true);
  } else if (kind === 'context') {
    const context = currentContext();
    modalFrame('网络概况', '拓扑角色用于解释和推荐，设备路线权限始终由 DHCP 分配权决定。', `
      <div class="row"><span>拓扑位置</span><b>${context.topology}</b></div><div class="row"><span>DHCP 分配权</span><b>${context.dhcp}</b></div><div class="row"><span>本机地址</span><span>${context.lan}</span></div><div class="row"><span>上级网关</span><span>${context.upstream}</span></div><div class="row"><span>设备路线能力</span><span>${context.editable ? '可配置' : '只读'}</span></div><div class="row"><span>判断可信度</span><span>${context.confidence}</span></div>
      ${context.topology === '无法确定' ? '<div class="notice">路由表存在多出口，无法可靠判断角色。你可以确认拓扑，但确认结果不会绕过 DHCP 权限检查。</div>' : ''}
      <details><summary>查看判断依据</summary><p class="muted">LAN 地址、LAN/WAN 默认路由、DHCP 服务状态和浮动网关参与状态。普通页面不直接显示 UCI 字段。</p></details>`, '<button class="button" data-action="close-modal">完成</button>');
  } else if (kind === 'dhcp') {
    const context = currentContext();
    modalFrame('地址分配', '关闭 DHCP 会让设备级地址保留和上网路线变为只读。', `
      <label class="choice"><input id="dhcp-enabled" type="checkbox" ${context.editable ? 'checked' : ''}> 由本机提供地址分配</label><div class="form-grid"><label class="field">地址池开始<input value="192.168.1.100"></label><label class="field">地址池结束<input value="192.168.1.249"></label><label class="field">租期<select><option>12 小时</option><option>24 小时</option><option>7 天</option></select></label><label class="field">已分配地址<input value="18 个" disabled></label></div>
      <div class="notice"><b>关闭前必须确认网络中存在另一个 DHCP 服务器。</b><br>18 台当前设备可能需要重新获取地址；地址保留和设备路线会在本机变成只读。</div>`, '<button class="button" data-action="close-modal">取消</button><button class="button primary" data-action="preview-dhcp">预览影响</button>');
  } else if (kind === 'routes') {
    const floatingAvailable = capabilityState('floating') === 'available';
    modalFrame('上网路线', '维护路线目标；为某台设备选择路线仍在设备详情中完成。', `
      <div class="route-item"><div><b>网络默认</b><p>${currentContext().defaultRoute} · 21 台设备</p></div><span class="tag">默认</span></div>
      <div class="route-item"><div><b>客厅旁路由</b><p>192.168.1.2 · 正常 · 3 台设备 · DNS 跟随路线</p></div><div class="route-actions"><button class="button small" data-action="edit-route-target">编辑</button></div></div>
      <div class="route-item"><div><b>浮动网关</b><p>${floatingAvailable ? '192.168.1.3 · 正常 · 5 台设备 · 当前节点 B' : `能力${capabilityText('floating')} · 已有引用不会被静默删除`}</p></div><div class="route-actions"><button class="button small" data-action="${floatingAvailable ? 'open-floating' : 'setup-capability'}" data-capability="floating">${floatingAvailable ? '详情' : capabilityState('floating') === 'error' ? '重试与诊断' : '了解并安装'}</button></div></div>
      <div class="route-item"><div><b>下载机</b><p>192.168.1.8 · 不可达 · 0 台设备</p></div><div class="route-actions"><button class="button small" data-action="edit-route-target">编辑</button><button class="button small danger" data-action="delete-route">删除</button></div></div>`, '<button class="button" data-action="close-modal">完成</button><button class="button primary" data-action="add-route">添加路线</button>', true);
  } else if (kind === 'floating') {
    if (capabilityState('floating') !== 'available') {
      capabilityReturnModal = 'routes';
      openModal('capability-setup', 'floating');
      return;
    }
    modalFrame('浮动网关', '浮动网关是一种高可用上网路线，节点细节只在这里出现。', `
      <div class="status-line"><span><b>运行正常</b><br>浮动地址当前由客厅旁路由 B 提供</span><span class="tag">5 台设备使用</span></div><div class="row"><span>浮动地址</span><b>192.168.1.3</b></div><div class="row"><span>优先服务节点</span><span>客厅旁路由 B · 在线</span></div><div class="row"><span>故障接管节点</span><span>主路由 A · 待命</span></div><div class="row"><span>最近切换</span><span>3 天前 · 12 秒内恢复</span></div>
      <h3 style="margin-top:18px">使用设备</h3><p>儿童游戏机、客厅电视等 5 台设备</p><details><summary>健康检测与高级参数</summary><p class="muted">检测间隔 3 秒 · 连续失败 3 次后接管。这里验证信息层级，不模拟真实切换。</p></details>`, '<button class="button" data-action="close-modal">完成</button><button class="button" data-action="simulate-failover">模拟一次切换</button>');
  } else if (kind === 'bandwidth') {
    modalFrame('带宽与统计', '统计能力异常时，设备列表和网络设置仍然可用。', `
      <div class="form-grid"><label class="field">总下载带宽 Mbps<input value="1000"></label><label class="field">总上传带宽 Mbps<input value="100"></label><label class="field">历史保留<select><option>30 天</option><option>7 天</option><option>90 天</option></select></label><label class="field">统计来源<select><option>自动选择（推荐）</option><option>基础统计</option><option>增强统计</option></select></label></div><div class="preview"><b>当前资源预算</b><br><span class="muted">批量采样 · 设备详情按需加载 · 预计内存上限 12 MB · 磁盘上限 64 MB。</span></div>`, '<button class="button" data-action="close-modal">取消</button><button class="button primary" data-action="save-generic">保存</button>');
  } else if (kind === 'diagnostics') {
    modalFrame('诊断与高级', '正常情况下保持安静；这里只呈现能够采取行动的问题和主动工具。', `
      <h3>需要处理</h3><div class="notice"><b>未命名设备存在地址冲突</b><br>重新选择地址后可以再次应用。</div><div class="notice"><b>下载机路线不可达</b><br>当前没有设备使用，不影响其他设备。</div>
      <h3>最近变更</h3><div class="timeline"><div class="timeline-item"><time>10:32</time><span>客厅 ASUS 电脑改用客厅旁路由</span></div><div class="timeline-item"><time>昨天</time><span>浮动网关由节点 A 切换到节点 B</span></div><div class="timeline-item"><time>3 天前</time><span>儿童设备计划已更新</span></div></div>
      <h3>主动工具</h3><div class="quick-actions"><button class="button" data-action="probe-entry">探测管理入口</button><button class="button" data-action="export-policy">导出策略</button><button class="button" data-action="import-policy">导入并预览</button><button class="button" data-action="simulate-failure">模拟应用失败</button></div>
      <details><summary>通知和 Webhook</summary><label class="field" style="margin-top:10px">Webhook 地址<input placeholder="https://example.invalid/hook"></label><p class="muted">仅通知需要处理的问题和浮动网关切换。</p></details><details><summary>专家诊断信息</summary><p class="muted">诊断导出可以包含脱敏后的内部映射和能力限制；普通 UI 不显示 DHCP tag、option 3/6 或 UCI section。</p></details>`, '<button class="button" data-action="close-modal">完成</button>', true);
  } else if (kind === 'route-target') {
    modalFrame('添加上网路线', '为用户命名一个可复用的网关目标。', `<div class="form-grid"><label class="field full">路线名称<input value="书房旁路由"></label><label class="field">类型<select><option>旁路由</option><option>指定 IP</option></select></label><label class="field">网关 IP<input value="192.168.1.9"></label></div><div class="preview"><b>DNS 跟随路线</b><br><span class="muted">保存前会检查地址格式、同网段、冲突和可达性。</span></div>`, '<button class="button" data-action="close-modal">取消</button><button class="button primary" data-action="save-route-target">保存路线</button>');
  } else if (kind === 'capability-setup') {
    const capability = targetId || params.get('capability') || 'floating';
    const name = capability === 'floating' ? '浮动网关能力' : '设备限速能力';
    const state = capabilityState(capability);
    modalFrame(name, '能力缺失只影响相关选项，其他设备管理功能可以继续使用。', `
      <div class="status-line ${state === 'error' ? 'failed' : 'pending'}"><span><b>${state === 'error' ? '当前状态读取或运行异常' : '当前尚未安装'}</b><br>${state === 'error' ? '已有配置保持不变，不允许新的应用。' : '安装前不会改变现有设备配置。'}</span></div>
      <div class="row"><span>受影响功能</span><span>${capability === 'floating' ? '选择和运行浮动网关路线' : '设置上传、下载速度上限'}</span></div><div class="row"><span>不受影响</span><span>${capability === 'floating' ? '其他上网路线、地址保留和联网权限' : '联网权限、分组、上网时段和流量额度'}</span></div>
      <div class="preview"><b>安装后的返回方式</b><br><span class="muted">保留当前编辑意图，安装完成后返回原位置；仍需用户预览并保存，不会自动应用。</span></div>
      <details><summary>查看技术组件</summary><p class="muted">${capability === 'floating' ? '浮动网关组件（floatip）' : '设备限速组件（eqos）'}。普通界面默认不显示包名。</p></details>`, `<button class="button" data-action="close-modal">稍后处理</button><button class="button primary" data-action="confirm-capability" data-capability="${capability}">${state === 'error' ? '模拟恢复能力' : '模拟安装完成'}</button>`);
  } else if (kind === 'preview') {
    modalFrame('确认修改', '配置保存和终端实际生效是两个不同状态。', `
      <div class="preview">${pendingDraft.lines.map(line => `<div class="row"><span>${escapeHtml(line[0])}</span><b>${escapeHtml(line[1])}</b></div>`).join('')}</div>
      <div class="notice"><b>${pendingDraft.impact}</b><br>${pendingDraft.recovery}</div>`, '<button class="button" data-action="return-edit">返回修改</button><button class="button primary" data-action="commit-change">确认保存</button>');
  }
}

function closeModal() {
  modalKind = null;
  document.getElementById('modal-backdrop').classList.remove('open');
  const next = new URLSearchParams(location.search);
  next.delete('modal');
  next.delete('group');
  next.delete('capability');
  history.replaceState(null, '', `${location.pathname}?${next}`);
}

function refreshProfileIconPreview() {
  const modeInput = document.getElementById('profile-icon-mode');
  if (!modeInput) return;
  const temporaryDevice = {
    brand: document.getElementById('profile-brand').value.trim() || '未知品牌',
    type: document.getElementById('profile-type').value,
    iconMode: modeInput.value,
    iconKey: document.getElementById('profile-icon-key').value
  };
  document.getElementById('profile-icon-preview').textContent = deviceIcon(temporaryDevice);
  document.getElementById('profile-icon-source').textContent = iconSourceText(temporaryDevice);
  document.querySelectorAll('.icon-option').forEach(button => button.classList.toggle('selected', temporaryDevice.iconMode === 'manual' && button.dataset.iconKey === temporaryDevice.iconKey));
}

function render() {
  document.getElementById('variant-label').textContent = `${variant} — ${variants[variant]}`;
  document.getElementById('capability-scenario').value = capabilityScenario;
  document.querySelectorAll('[data-screen]').forEach(button => button.classList.toggle('active', button.dataset.screen === screen));
  document.querySelectorAll('[data-context]').forEach(button => button.classList.toggle('active', button.dataset.context === contextKey));
  document.getElementById('screen-device').innerHTML = variant === 'A' ? renderDeviceA() : variant === 'B' ? renderDeviceB() : renderDeviceC();
  document.getElementById('screen-groups').innerHTML = renderGroups();
  document.getElementById('screen-network').innerHTML = renderNetwork();
  document.getElementById('screen-device').classList.toggle('hidden', screen !== 'device');
  document.getElementById('screen-groups').classList.toggle('hidden', screen !== 'groups');
  document.getElementById('screen-network').classList.toggle('hidden', screen !== 'network');
  document.getElementById('global-add-device').classList.toggle('hidden', screen !== 'device');
  document.getElementById('drawer-backdrop').classList.toggle('open', Boolean(selectedDeviceId));
  renderStateStrip();
  renderDrawer();
  updateUrl();
}

function cycle(direction) {
  const keys = Object.keys(variants);
  variant = keys[(keys.indexOf(variant) + direction + keys.length) % keys.length];
  render();
}

function previewNetwork() {
  const hostname = document.getElementById('network-hostname').value.trim();
  const error = document.getElementById('hostname-error');
  if (hostname && !/^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$/.test(hostname)) {
    error.textContent = '只允许英文、数字和中间连字符，且不能以连字符开头或结尾。';
    return;
  }
  const device = selectedDevice();
  const addressMode = document.querySelector('input[name="address-mode"]:checked').value;
  const routeKind = document.querySelector('input[name="route"]:checked').value;
  if (routeKind === 'floating' && capabilityState('floating') !== 'available') {
    showToast('请先安装浮动网关能力，或选择其他上网路线');
    return;
  }
  const routeLabels = { default: '跟随网络默认', self: '本机路由', upstream: '上级路由', side: '客厅旁路由', floating: '浮动网关', custom: `指定 IP ${document.getElementById('custom-route').value}` };
  pendingDraft = {
    kind: 'network', addressMode, routeKind, hostname, route: routeLabels[routeKind],
    lines: [['地址', addressMode === 'auto' ? '自动获取' : addressMode === 'reserved' ? `保留 ${device.address}` : `指定 ${document.getElementById('custom-address').value}`], ['上网路线', `${device.route} → ${routeLabels[routeKind]}`], ['DNS', '跟随所选路线'], ['网络主机名', hostname || '未设置']],
    impact: '设备可能短暂断网，并需要重新联网或续租后才会生效。', recovery: '如果应用失败，原地址与路线将整体恢复。'
  };
  openModal('preview');
}

function previewLimits() {
  const speedAvailable = capabilityState('speed') === 'available';
  const access = document.getElementById('limit-access').value;
  const group = document.getElementById('limit-group').value;
  const down = document.getElementById('limit-down').value;
  const up = document.getElementById('limit-up').value;
  const quota = document.getElementById('quota-value').value;
  pendingDraft = { kind: 'limits', access, group, down, up, quota, changeSpeed: speedAvailable, lines: [['联网权限', access === 'paused' ? '暂停联网' : '允许联网'], ['设备分组', group], ['限速', speedAvailable ? (down === '0' && up === '0' ? '不限速' : `下载 ${down} / 上传 ${up} Mbps`) : '保持原设置（能力不可用）'], ['上网时段', `${document.getElementById('schedule-start').value}—${document.getElementById('schedule-end').value}`], ['流量额度', quota === '0' ? '未设置' : `每月 ${quota} GB`]], impact: '规则保存后立即进入运行状态；不可用且未修改的能力不会阻断其他设置。', recovery: '如果应用失败，本次修改整体回滚。' };
  openModal('preview');
}

function commitChange() {
  const device = selectedDevice();
  let subject = device.name;
  if (pendingDraft?.kind === 'network') {
    device.reserved = pendingDraft.addressMode !== 'auto';
    device.routeKind = pendingDraft.routeKind;
    device.route = pendingDraft.route;
    device.hostname = pendingDraft.hostname;
    device.labels = device.routeKind === 'default' ? device.labels.filter(label => !label.startsWith('经')) : [`经${device.route}`, ...device.labels.filter(label => !label.startsWith('经'))];
  } else if (pendingDraft?.kind === 'limits') {
    device.access = pendingDraft.access;
    device.group = pendingDraft.group;
    if (pendingDraft.changeSpeed) device.limits = pendingDraft.down === '0' && pendingDraft.up === '0' ? '不限速' : `下载 ${pendingDraft.down} / 上传 ${pendingDraft.up} Mbps`;
    device.quota = pendingDraft.quota === '0' ? '未设置' : `每月 ${pendingDraft.quota} GB`;
  } else if (pendingDraft?.kind === 'group') {
    subject = '设备分组规则';
  } else if (pendingDraft?.kind === 'dhcp') {
    subject = '地址分配设置';
    if (pendingDraft.enabled) {
      contextKey = currentContext().topology === '旁路由' ? 'side-local' : currentContext().topology === '无法确定' ? 'ambiguous' : 'main';
    } else {
      contextKey = currentContext().topology === '旁路由' ? 'side-external' : 'main-external';
    }
  }
  prototypeState.apply = 'pending';
  prototypeState.lastAction = `${subject}已保存，等待相关设备更新`;
  pendingDraft = null;
  closeModal();
  render();
  showToast('配置已保存，设备续租后生效');
}

function handleAction(target) {
  const action = target.dataset.action;
  if (!action) return;
  if (action === 'open-device') { selectedDeviceId = target.dataset.deviceId || 'asus'; render(); }
  else if (action === 'close-drawer') { selectedDeviceId = null; render(); }
  else if (action === 'add-device') openModal('add-device');
  else if (action === 'edit-profile') openModal('profile');
  else if (action === 'edit-network' && currentContext().editable) openModal('network');
  else if (action === 'open-limits') openModal('limits', target.dataset.deviceId);
  else if (action === 'open-usage') openModal('usage');
  else if (action === 'close-modal') closeModal();
  else if (action === 'preview-network') previewNetwork();
  else if (action === 'preview-limits') previewLimits();
  else if (action === 'commit-change') commitChange();
  else if (action === 'return-edit') {
    const returnKind = pendingDraft?.kind === 'network' ? 'network' : pendingDraft?.kind === 'limits' ? 'limits' : pendingDraft?.kind === 'group' ? 'group' : 'dhcp';
    openModal(returnKind);
  }
  else if (action === 'toggle-access') {
    const device = selectedDevice(); device.access = device.access === 'paused' ? 'allowed' : 'paused';
    prototypeState.lastAction = `${device.name}已${device.access === 'paused' ? '暂停' : '恢复'}联网`;
    render(); showToast(prototypeState.lastAction);
  } else if (action === 'copy-gateway') {
    navigator.clipboard?.writeText(currentContext().lan).catch(() => {}); showToast(`已复制 ${currentContext().lan}`);
  } else if (action === 'icon-auto') {
    document.getElementById('profile-icon-mode').value = 'auto';
    document.getElementById('profile-icon-key').value = '';
    refreshProfileIconPreview();
  } else if (action === 'select-icon') {
    document.getElementById('profile-icon-mode').value = 'manual';
    document.getElementById('profile-icon-key').value = target.dataset.iconKey;
    refreshProfileIconPreview();
  } else if (action === 'save-profile') {
    const device = selectedDevice(); device.name = document.getElementById('profile-name').value.trim() || device.name; device.brand = document.getElementById('profile-brand').value.trim() || '未知品牌'; device.type = document.getElementById('profile-type').value; device.iconMode = document.getElementById('profile-icon-mode').value; device.iconKey = document.getElementById('profile-icon-key').value; closeModal(); render(); showToast('设备资料和图标已保存，不会写入 DHCP');
  } else if (action === 'save-new-device') {
    const mac = document.getElementById('add-mac').value.trim();
    if (!/^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$/.test(mac)) { showToast('请输入有效的 MAC 地址'); return; }
    devices.push({ id: `manual-${devices.length}`, iconMode: 'auto', iconKey: '', name: document.getElementById('add-name').value.trim() || '未命名设备', brand: '未知品牌', type: document.getElementById('add-type').value, presence: 'never', access: 'allowed', address: '尚未分配', connection: '从未连接', speed: '—', route: '跟随网络默认', routeKind: 'default', reserved: false, hostname: '', limits: '不限速', group: '未分组', schedule: '未设置', quota: '未设置', manufacturer: '尚未观察到', confidence: '—', source: `用户通过 ${mac} 创建`, labels: ['从未上线'], issue: '', today: '—', history: [] });
    closeModal(); filter = 'all'; render(); showToast('设备已添加，上线后会自动补充信息');
  } else if (action === 'new-group') openModal('group');
  else if (action === 'manage-group') openModal('group', target.dataset.groupId);
  else if (action === 'preview-group') {
    pendingDraft = { kind: 'group', lines: [['分组', document.getElementById('group-name').value], ['受影响设备', '2 台'], ['例外设备', '保留单独设置'], ['规则', '路线、时段、限速和额度一起保存']], impact: '组内设备将在下一次策略计算时更新。', recovery: '已有单设备例外不会被覆盖。' }; openModal('preview');
  } else if (action === 'open-context') openModal('context');
  else if (action === 'open-dhcp') openModal('dhcp');
  else if (action === 'open-routes') openModal('routes');
  else if (action === 'open-floating') openModal('floating');
  else if (action === 'open-bandwidth') openModal('bandwidth');
  else if (action === 'open-diagnostics') openModal('diagnostics');
  else if (action === 'add-route' || action === 'edit-route-target') openModal('route-target');
  else if (action === 'setup-capability') {
    capabilityReturnModal = modalKind && modalKind !== 'capability-setup' ? modalKind : target.dataset.capability === 'floating' ? 'network' : 'limits';
    openModal('capability-setup', target.dataset.capability);
  }
  else if (action === 'confirm-capability') {
    const capability = target.dataset.capability;
    capabilityOverrides[capability] = 'available';
    const returnModal = capabilityReturnModal;
    capabilityReturnModal = null;
    closeModal(); render();
    showToast(`${capability === 'floating' ? '浮动网关' : '设备限速'}能力已可用，尚未自动应用设置`);
    if (returnModal) openModal(returnModal);
  }
  else if (action === 'delete-route') showToast('该路线没有设备使用，可以安全删除（原型未实际删除）');
  else if (action === 'save-route-target') { closeModal(); showToast('路线已保存，DNS 默认跟随路线'); }
  else if (action === 'save-generic') { closeModal(); showToast('设置已保存'); }
  else if (action === 'preview-dhcp') {
    const enabled = document.getElementById('dhcp-enabled').checked;
    pendingDraft = { kind: 'dhcp', enabled, lines: [['本机 DHCP', enabled ? '保持开启' : '关闭'], ['设备路线权限', enabled ? '保持可编辑' : '变为只读'], ['当前租约', '18 台设备需要重新获取地址']], impact: enabled ? '地址池变化可能使部分设备重新获取地址。' : '关闭前必须确认网络中存在另一个 DHCP 服务器。', recovery: '保存失败时恢复当前 DHCP 配置和地址池。' }; openModal('preview');
  } else if (action === 'simulate-renew') { prototypeState.apply = 'applied'; prototypeState.lastAction = '设备已续租，新地址和路线已生效'; render(); showToast('已模拟设备续租，配置生效'); }
  else if (action === 'simulate-failure') { prototypeState.apply = 'failed'; prototypeState.lastAction = '模拟应用失败；原配置已恢复'; closeModal(); render(); showToast('已模拟失败和自动恢复'); }
  else if (action === 'clear-failure') { prototypeState.apply = 'applied'; prototypeState.lastAction = '原配置运行正常'; render(); }
  else if (action === 'simulate-failover') { closeModal(); prototypeState.lastAction = '浮动网关已从节点 B 切换到节点 A，设备路线保持不变'; render(); showToast('已模拟浮动网关切换'); }
  else if (action === 'probe-entry') showToast('探测完成：主路由管理入口可访问');
  else if (action === 'export-policy') showToast('原型：将导出脱敏后的策略与诊断摘要');
  else if (action === 'import-policy') showToast('原型：导入后必须先预览差异，不会直接应用');
  else if (action === 'refresh') showToast('设备观察数据已刷新，当前编辑不会被打断');
  else if (action === 'task-online') { variant = 'A'; filter = 'online'; render(); }
}

document.addEventListener('click', event => {
  const target = event.target.closest('[data-action], [data-screen], [data-context], [data-filter]');
  if (!target) return;
  if (target.dataset.screen) { screen = target.dataset.screen; selectedDeviceId = null; closeModal(); render(); return; }
  if (target.dataset.context) { contextKey = target.dataset.context; closeModal(); render(); return; }
  if (target.dataset.filter) { filter = target.dataset.filter; render(); return; }
  handleAction(target);
});

document.getElementById('drawer-backdrop').addEventListener('click', event => {
  if (event.target.id === 'drawer-backdrop') { selectedDeviceId = null; render(); }
});
document.getElementById('modal-backdrop').addEventListener('click', event => {
  if (event.target.id === 'modal-backdrop') closeModal();
});
document.getElementById('previous').addEventListener('click', () => cycle(-1));
document.getElementById('next').addEventListener('click', () => cycle(1));
document.getElementById('capability-scenario').addEventListener('change', event => {
  capabilityScenario = event.target.value;
  Object.keys(capabilityOverrides).forEach(key => delete capabilityOverrides[key]);
  closeModal();
  render();
});
document.addEventListener('change', event => {
  if (['profile-brand', 'profile-type'].includes(event.target.id) && document.getElementById('profile-icon-mode')?.value === 'auto') refreshProfileIconPreview();
});
document.addEventListener('keydown', event => {
  if (event.key === 'Escape') { if (modalKind) closeModal(); else { selectedDeviceId = null; render(); } return; }
  if (['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement.tagName) || document.activeElement.isContentEditable) return;
  if (event.key === 'ArrowLeft') cycle(-1);
  if (event.key === 'ArrowRight') cycle(1);
});

render();
if (initialModal) openModal(initialModal, params.get('group') || params.get('capability') || undefined);

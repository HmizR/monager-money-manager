/* ============================================================
   Money Manager — app.js
   ============================================================ */

'use strict';

// ── Config ─────────────────────────────────────────────────────

// ── State ──────────────────────────────────────────────────────
let categoryChart     = null;
let trendChart        = null;
let balanceTrendChart = null;
let globalRange       = '30D';
let currentCategoryContext = null;
let allCategories     = [];
/** @type {Array<{uuid:string,name:string,type:string,currency_code:string}>} */
let accountsCache     = [];

// ── Utility: Toast notification ────────────────────────────────
function toast(message, type = 'success') {
  const icons = { success: 'fa-check-circle', error: 'fa-times-circle', info: 'fa-info-circle' };
  const container = document.getElementById('toastContainer');
  const el = document.createElement('div');
  el.className = `toast ${type}`;
  el.innerHTML = `<i class="fas ${icons[type]}"></i><span>${message}</span>`;
  container.appendChild(el);
  setTimeout(() => { el.style.opacity = '0'; el.style.transform = 'translateX(20px)'; el.style.transition = '0.3s'; setTimeout(() => el.remove(), 320); }, 3000);
}

function escapeHtml(s) {
  if (s == null) return '';
  const d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}

function formatMoney(amount, currencyCode = 'IDR') {
  const code = (currencyCode || 'IDR').toUpperCase();
  try {
    return new Intl.NumberFormat('id-ID', { style: 'currency', currency: code, maximumFractionDigits: 2 }).format(Number(amount) || 0);
  } catch {
    return `${code} ${(Number(amount) || 0).toLocaleString('id-ID')}`;
  }
}

function accountDisplayName(uuid) {
  if (!uuid) return '—';
  const a = accountsCache.find(x => x.uuid === uuid);
  return a ? a.name : uuid.slice(0, 8) + '…';
}

function accountTypeIconClass(type) {
  if (type === 'bank') return ['fa-university', 'bank'];
  if (type === 'ewallet') return ['fa-mobile-screen-button', 'ewallet'];
  return ['fa-money-bill-wave', 'cash'];
}

function setDateInputIfEmpty(id) {
  const el = document.getElementById(id);
  if (!el || el.value) return;
  const d = new Date();
  el.value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function fillAccountSelectsFromCache() {
  const preserve = {
    txnAccountSelect: document.getElementById('txnAccountSelect')?.value,
    txnAccountFilter: document.getElementById('txnAccountFilter')?.value,
    transferFrom: document.getElementById('transferFrom')?.value,
    transferTo: document.getElementById('transferTo')?.value,
  };

  const selTxn = document.getElementById('txnAccountSelect');
  if (selTxn) {
    selTxn.innerHTML = '<option value="">Default (first account)</option>';
    accountsCache.forEach(a => {
      const o = document.createElement('option');
      o.value = a.uuid;
      o.textContent = `${a.name} (${a.currency_code})`;
      selTxn.appendChild(o);
    });
    if (preserve.txnAccountSelect && [...selTxn.options].some(o => o.value === preserve.txnAccountSelect)) selTxn.value = preserve.txnAccountSelect;
  }

  const fil = document.getElementById('txnAccountFilter');
  if (fil) {
    fil.innerHTML = '<option value="">All accounts</option>';
    accountsCache.forEach(a => {
      const o = document.createElement('option');
      o.value = a.uuid;
      o.textContent = `${a.name} (${a.currency_code})`;
      fil.appendChild(o);
    });
    if (preserve.txnAccountFilter && [...fil.options].some(o => o.value === preserve.txnAccountFilter)) fil.value = preserve.txnAccountFilter;
  }

  ['transferFrom', 'transferTo'].forEach(id => {
    const s = document.getElementById(id);
    if (!s) return;
    s.innerHTML = '';
    if (!accountsCache.length) {
      const o = document.createElement('option');
      o.value = '';
      o.textContent = 'No accounts';
      s.appendChild(o);
      return;
    }
    accountsCache.forEach(a => {
      const o = document.createElement('option');
      o.value = a.uuid;
      o.textContent = `${a.name} (${a.currency_code})`;
      s.appendChild(o);
    });
    const prev = preserve[id];
    if (prev && [...s.options].some(o => o.value === prev)) s.value = prev;
  });
}

async function loadAccounts() {
  try {
    const res = await fetch(`${API_URL}/accounts`, { credentials: 'include' });
    const data = await res.json();
    accountsCache = res.ok && data.accounts ? data.accounts : [];
  } catch {
    accountsCache = [];
  }
  fillAccountSelectsFromCache();
  return accountsCache;
}

function computeBalancesPerAccount(transactions) {
  /** @type {Record<string, { sum: number, currency: string }>} */
  const m = {};
  (transactions || []).forEach(t => {
    const key = t.account_uuid || '_unset';
    if (!m[key]) m[key] = { sum: 0, currency: (t.currency_code || 'IDR').toUpperCase() };
    m[key].sum += t.type === 'income' ? Number(t.amount) : -Number(t.amount);
    if (t.currency_code) m[key].currency = String(t.currency_code).toUpperCase();
  });
  return m;
}

async function loadAccountsPage() {
  await loadAccounts();
  let balances = {};
  try {
    const tr = await fetch(`${API_URL}/transactions`, { credentials: 'include' });
    const td = await tr.json();
    balances = computeBalancesPerAccount(td.transactions || []);
  } catch (e) { console.error(e); }

  const grid = document.getElementById('accountsGrid');
  if (!grid) return;

  if (!accountsCache.length) {
    grid.innerHTML = '<div class="empty-state"><i class="fas fa-wallet"></i><p>Belum ada akun. Tambahkan akun pertama di bawah.</p></div>';
    setDateInputIfEmpty('transferDate');
    return;
  }

  grid.innerHTML = accountsCache.map(a => {
    const b = balances[a.uuid] || { sum: 0, currency: a.currency_code };
    const cur = b.currency || a.currency_code;
    const [icon, icls] = accountTypeIconClass(a.type);
    const typeLabel = a.type === 'ewallet' ? 'E-wallet' : a.type.charAt(0).toUpperCase() + a.type.slice(1);
    const balStr = formatMoney(b.sum, cur);
    const neg = b.sum < 0;
    return `
      <div class="account-card">
        <div class="account-card-top">
          <div style="display:flex;gap:12px;align-items:flex-start;min-width:0">
            <div class="account-card-icon ${icls}"><i class="fas ${icon}"></i></div>
            <div style="min-width:0">
              <div class="account-card-name">${escapeHtml(a.name)}</div>
              <div class="account-card-meta">${escapeHtml(typeLabel)} · ${escapeHtml(a.currency_code)}</div>
            </div>
          </div>
        </div>
        <div class="account-card-balance" style="color:${neg ? 'var(--red)' : 'var(--accent)'}">${balStr}</div>
        <div class="account-card-actions">
          <button type="button" class="btn btn-sm btn-danger" onclick="deleteAccount('${a.uuid}')" title="Hapus jika belum ada transaksi">
            <i class="fas fa-trash"></i> Hapus
          </button>
        </div>
      </div>`;
  }).join('');

  setDateInputIfEmpty('transferDate');
}

async function createAccount() {
  const name = document.getElementById('newAccountName')?.value.trim();
  const type = document.getElementById('newAccountType')?.value;
  let currency = (document.getElementById('newAccountCurrency')?.value || 'IDR').trim().toUpperCase();
  if (!name) { toast('Isi nama akun', 'error'); return; }
  if (currency.length !== 3) { toast('Kode mata uang harus 3 huruf (mis. IDR, USD)', 'error'); return; }
  try {
    const res = await fetch(`${API_URL}/accounts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, type, currency_code: currency }),
      credentials: 'include',
    });
    const data = await res.json().catch(() => ({}));
    if (res.ok) {
      toast('Akun berhasil dibuat');
      document.getElementById('newAccountName').value = '';
      document.getElementById('newAccountCurrency').value = '';
      await loadAccountsPage();
    } else toast(data.error || 'Gagal membuat akun', 'error');
  } catch { toast('Koneksi error', 'error'); }
}

async function deleteAccount(uuid) {
  if (!confirm('Hapus akun ini? Hanya bisa jika belum ada transaksi.')) return;
  try {
    const res = await fetch(`${API_URL}/accounts/${uuid}`, { method: 'DELETE', credentials: 'include' });
    const data = await res.json().catch(() => ({}));
    if (res.ok) {
      toast('Akun dihapus');
      await loadAccountsPage();
      await loadTransactions();
    } else toast(data.error || 'Gagal menghapus', 'error');
  } catch { toast('Koneksi error', 'error'); }
}

async function createTransfer() {
  const from = document.getElementById('transferFrom')?.value;
  const to = document.getElementById('transferTo')?.value;
  const amount = parseFloat(document.getElementById('transferAmount')?.value || '');
  const transaction_date = document.getElementById('transferDate')?.value;
  const description = document.getElementById('transferDesc')?.value || '';
  if (!from || !to) { toast('Pilih akun asal dan tujuan', 'error'); return; }
  if (from === to) { toast('Akun asal dan tujuan harus berbeda', 'error'); return; }
  if (!amount || amount <= 0) { toast('Jumlah tidak valid', 'error'); return; }
  if (!transaction_date) { toast('Pilih tanggal', 'error'); return; }
  try {
    const res = await fetch(`${API_URL}/transfers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        from_account_uuid: from,
        to_account_uuid: to,
        amount,
        description,
        transaction_date,
      }),
      credentials: 'include',
    });
    const data = await res.json().catch(() => ({}));
    if (res.ok) {
      toast('Transfer berhasil');
      document.getElementById('transferAmount').value = '';
      document.getElementById('transferDesc').value = '';
      await loadAccountsPage();
      await loadTransactions();
      loadDashboard();
    } else toast(data.error || 'Transfer gagal', 'error');
  } catch { toast('Koneksi error', 'error'); }
}

// ── Auth: Login ────────────────────────────────────────────────
document.getElementById('loginForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const username = document.getElementById('loginUsername').value;
  const password = document.getElementById('loginPassword').value;
  try {
    const res  = await fetch(`${API_URL}/auth/login`, { 
        method: 'POST', 
        headers: { 'Content-Type': 'application/json' }, 
        body: JSON.stringify({ username, password }), 
        credentials: 'include'
    });
    const data = await res.json();
    if (res.ok) {
      localStorage.setItem('user', JSON.stringify(data.user));
      showDashboard();
      loadDashboard();
    } else { toast(data.error || 'Login failed', 'error'); }
  } catch { toast('Connection error', 'error'); }
});

// ── Auth: Register ─────────────────────────────────────────────
document.getElementById('registerForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const username = document.getElementById('regUsername').value;
  const email    = document.getElementById('regEmail').value;
  const password = document.getElementById('regPassword').value;
  try {
    const res  = await fetch(`${API_URL}/auth/register`, { 
        method: 'POST', 
        headers: { 'Content-Type': 'application/json' }, 
        body: JSON.stringify({ username, email, password }),
        credentials: 'include'
    });
    const data = await res.json();
    if (res.ok) {
      localStorage.setItem('user', JSON.stringify(data.user));
      showDashboard();
      loadDashboard();
    } else { toast(data.error || 'Registration failed', 'error'); }
  } catch { toast('Connection error', 'error'); }
});

function showRegister() {
  document.getElementById('loginPage').style.display = 'none';
  document.getElementById('registerPage').style.display = 'flex';
}
function showLogin() {
  document.getElementById('registerPage').style.display = 'none';
  document.getElementById('loginPage').style.display = 'flex';
}

async function logout() {
  try {
    const res  = await fetch(`${API_URL}/auth/logout`, { 
        method: 'POST', 
        headers: { 'Content-Type': 'application/json' }, 
        credentials: 'include'
    });
    const data = await res.json();
    if (res.ok) {
      localStorage.removeItem('user');
      document.getElementById('dashboard').style.display = 'none';
      document.getElementById('loginPage').style.display = 'flex';
    } else { toast(data.error || 'Login failed', 'error'); }
  } catch { toast('Connection error', 'error'); }
}

function showDashboard() {
  document.getElementById('loginPage').style.display   = 'none';
  document.getElementById('registerPage').style.display = 'none';
  document.getElementById('dashboard').style.display   = 'block';
  const user = JSON.parse(localStorage.getItem('user') || '{}');
  document.querySelectorAll('.user-display').forEach(el => el.textContent = user.username || 'User');
}

// ── Navigation ─────────────────────────────────────────────────
document.querySelectorAll('.menu-item').forEach(item => {
  item.addEventListener('click', () => {
    const page = item.dataset.page;
    document.querySelectorAll('.menu-item').forEach(i => i.classList.remove('active'));
    item.classList.add('active');
    document.querySelectorAll('.page-content').forEach(p => p.classList.remove('active-page'));
    const pageEl = document.getElementById(`${page}Page`);
    if (pageEl) pageEl.classList.add('active-page');
    const titles = { dashboard: 'Dashboard', transactions: 'Transactions', accounts: 'Accounts & Transfer', budgets: 'Budgets', recommendations: 'Recommendations', export: 'Export / Import' };
    document.getElementById('pageTitle').textContent = titles[page] || page;
    closeMobileSidebar();
    if (page === 'dashboard')       loadDashboard();
    else if (page === 'transactions') { loadCategories(); loadAccounts().then(() => loadTransactions()); }
    else if (page === 'accounts')    loadAccountsPage();
    else if (page === 'budgets')      { loadCategories(); loadBudgets(); }
    else if (page === 'recommendations') loadRecommendations();
  });
});

// ── Mobile sidebar ──────────────────────────────────────────────
document.getElementById('hamburger').addEventListener('click', () => {
  document.getElementById('sidebar').classList.add('open');
  document.getElementById('sidebarBackdrop').classList.add('open');
});
function closeMobileSidebar() {
  document.getElementById('sidebar').classList.remove('open');
  document.getElementById('sidebarBackdrop').classList.remove('open');
}
document.getElementById('sidebarBackdrop').addEventListener('click', closeMobileSidebar);

// ── Range buttons ───────────────────────────────────────────────
document.querySelectorAll('.range-btn').forEach(btn => {
  btn.addEventListener('click', async () => {
    document.querySelectorAll('.range-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    globalRange = btn.dataset.range;
    await loadDashboard();
  });
});

// ── Dashboard ───────────────────────────────────────────────────
async function loadDashboard() {
  try {
    await loadCategories();

    // Stats with range
    const statsRes  = await fetch(`${API_URL}/statistics?range=${globalRange}`, { credentials: 'include' });
    const stats     = await statsRes.json();
    const net = (stats.total_income || 0) - (stats.total_expense || 0);
    const netClass  = net >= 0 ? 'netchange' : 'expense';

    document.getElementById('statIncome').textContent  = `Rp ${(stats.total_income || 0).toLocaleString()}`;
    document.getElementById('statExpense').textContent = `Rp ${(stats.total_expense || 0).toLocaleString()}`;
    document.getElementById('statNet').textContent     = `Rp ${net.toLocaleString()}`;

    // Current balance
    const balRes  = await fetch(`${API_URL}/balance/current`, { credentials: 'include' });
    const balData = await balRes.json();
    document.getElementById('currentBalance').textContent = `Rp ${(balData.balance || 0).toLocaleString()}`;

    await loadTrendChart();
    await loadCategoryChart();
    await loadBalanceTrend();
    await loadBalanceForecast();
    await loadMonthlyProjection();
  } catch (err) { console.error('Dashboard error:', err); }
}

// ── Charts ──────────────────────────────────────────────────────
const chartDefaults = {
  color: '#94a3b8',
  plugins: { legend: { labels: { color: '#94a3b8', font: { family: 'DM Sans' } } } },
};

async function loadTrendChart() {
  try {
    const res  = await fetch(`${API_URL}/charts?type=trend&range=${globalRange}`, { credentials: 'include' });
    const data = await res.json();
    if (trendChart) trendChart.destroy();
    const ctx = document.getElementById('trendChart').getContext('2d');
    trendChart = new Chart(ctx, {
      type: 'line',
      data: {
        labels: data.months.map(d => d.split('T')[0]),
        datasets: [
          { label: 'Income',  data: data.income,  borderColor: '#00e5a0', backgroundColor: 'rgba(0,229,160,0.08)', fill: true, tension: 0.4, pointRadius: 3 },
          { label: 'Expense', data: data.expense, borderColor: '#ff5370', backgroundColor: 'rgba(255,83,112,0.08)', fill: true, tension: 0.4, pointRadius: 3 }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: { ...chartDefaults.plugins, tooltip: { callbacks: { label: ctx => `${ctx.dataset.label}: Rp ${ctx.parsed.y.toLocaleString()}` } } },
        scales: {
          x: { ticks: { color: '#64748b', maxTicksLimit: 8, maxRotation: 30 }, grid: { color: 'rgba(255,255,255,0.04)' } },
          y: { ticks: { color: '#64748b', callback: v => 'Rp ' + (v >= 1000000 ? (v/1000000).toFixed(1)+'M' : v.toLocaleString()) }, grid: { color: 'rgba(255,255,255,0.04)' } }
        }
      }
    });
  } catch (err) { console.error(err); }
}

async function loadCategoryChart() {
  try {
    const res  = await fetch(`${API_URL}/charts?type=category&range=${globalRange}`, { credentials: 'include' });
    const data = await res.json();
    if (categoryChart) categoryChart.destroy();
    const ctx = document.getElementById('categoryChart').getContext('2d');
    categoryChart = new Chart(ctx, {
      type: 'doughnut',
      data: {
        labels: data.labels,
        datasets: [{ data: data.data, backgroundColor: ['#00e5a0','#60a5fa','#f59e0b','#ff5370','#a78bfa','#fb923c','#34d399','#f472b6'], borderWidth: 0, hoverOffset: 8 }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        cutout: '60%',
        plugins: {
          ...chartDefaults.plugins,
          legend: { position: 'bottom', labels: { color: '#94a3b8', font: { family: 'DM Sans', size: 12 }, padding: 12, boxWidth: 12 } },
          tooltip: { callbacks: { label: ctx => `${ctx.label}: Rp ${ctx.parsed.toLocaleString()}` } }
        }
      }
    });
  } catch (err) { console.error(err); }
}

async function loadBalanceTrend() {
  try {
    const res  = await fetch(`${API_URL}/balance/trend?range=${globalRange}`, { credentials: 'include' });
    const data = await res.json();
    if (balanceTrendChart) balanceTrendChart.destroy();
    const ctx = document.getElementById('balanceTrendChart').getContext('2d');
    balanceTrendChart = new Chart(ctx, {
      type: 'line',
      data: {
        labels: data.trends.map(t => t.date),
        datasets: [
          { label: 'Balance',            data: data.trends.map(t => t.balance),            borderColor: '#60a5fa', backgroundColor: 'rgba(96,165,250,0.08)', fill: true, tension: 0.4, borderWidth: 2.5 },
          { label: 'Cumulative Income',  data: data.trends.map(t => t.cumulative_income),  borderColor: '#00e5a0', fill: false, borderDash: [5,5], borderWidth: 1.5, pointRadius: 0 },
          { label: 'Cumulative Expense', data: data.trends.map(t => t.cumulative_expense), borderColor: '#ff5370', fill: false, borderDash: [5,5], borderWidth: 1.5, pointRadius: 0 }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: 'index', intersect: false },
        plugins: { ...chartDefaults.plugins, tooltip: { callbacks: { label: ctx => `${ctx.dataset.label}: Rp ${ctx.parsed.y.toLocaleString()}` } } },
        scales: {
          x: { ticks: { color: '#64748b', maxTicksLimit: 8, maxRotation: 30 }, grid: { color: 'rgba(255,255,255,0.04)' } },
          y: { ticks: { color: '#64748b', callback: v => 'Rp ' + (v >= 1000000 ? (v/1000000).toFixed(1)+'M' : v.toLocaleString()) }, grid: { color: 'rgba(255,255,255,0.04)' } }
        }
      }
    });
  } catch (err) { console.error(err); }
}

// ── Balance Forecast ────────────────────────────────────────────
async function loadBalanceForecast() {
  try {
    const res      = await fetch(`${API_URL}/balance/forecast`, { credentials: 'include' });
    const forecast = await res.json();

    document.getElementById('forecastCards').innerHTML = `
      <div class="forecast-grid">
        <div class="stat-card balance">
          <div class="stat-icon"><i class="fas fa-wallet"></i></div>
          <div class="stat-label">Starting Balance</div>
          <div class="stat-value">Rp ${(forecast.starting_balance||0).toLocaleString()}</div>
          <div class="stat-sub">as of ${forecast.start_date||''}</div>
        </div>
        <div class="stat-card income">
          <div class="stat-icon"><i class="fas fa-arrow-trend-up"></i></div>
          <div class="stat-label">Expected Income (30d)</div>
          <div class="stat-value">Rp ${(forecast.expected_income||0).toLocaleString()}</div>
          <div class="stat-sub">Rp ${(forecast.daily_average_income||0).toLocaleString()}/day avg</div>
        </div>
        <div class="stat-card expense">
          <div class="stat-icon"><i class="fas fa-arrow-trend-down"></i></div>
          <div class="stat-label">Expected Expense (30d)</div>
          <div class="stat-value">Rp ${(forecast.expected_expense||0).toLocaleString()}</div>
          <div class="stat-sub">Rp ${(forecast.daily_average_expense||0).toLocaleString()}/day avg</div>
        </div>
        <div class="stat-card ${forecast.ending_balance >= 0 ? 'balance' : 'expense'}">
          <div class="stat-icon"><i class="fas fa-flag-checkered"></i></div>
          <div class="stat-label">Ending Balance Forecast</div>
          <div class="stat-value">Rp ${(forecast.ending_balance||0).toLocaleString()}</div>
          <div class="stat-sub">by ${forecast.end_date||''}</div>
        </div>
      </div>
    `;

    const recsHtml = forecast.recommendations?.length
      ? forecast.recommendations.map(r => `<div class="rec-item"><i class="fas fa-chart-line"></i><span>${r}</span></div>`).join('')
      : `<div class="empty-state"><i class="fas fa-check-circle" style="color:var(--accent)"></i><p>No recommendations at this time</p></div>`;
    document.getElementById('forecastRecommendations').innerHTML = recsHtml;
  } catch (err) { console.error(err); }
}

// ── Monthly Projection ──────────────────────────────────────────
async function loadMonthlyProjection() {
  try {
    const res        = await fetch(`${API_URL}/balance/projection`, { credentials: 'include' });
    const projection = await res.json();
    const endBal     = projection.projected_ending_balance || 0;
    const isPos      = endBal >= 0;
    const incomeGrowth = projection.actual_income
      ? (((projection.projected_income - projection.actual_income) / projection.actual_income) * 100).toFixed(1)
      : '0.0';

    const html = `
      <div class="projection-grid">
        <div class="projection-cell">
          <div class="pcell-label">Month</div>
          <div class="pcell-value">${projection.month} ${projection.year}</div>
          <div class="pcell-sub">${projection.days_remaining} days remaining</div>
        </div>
        <div class="projection-cell">
          <div class="pcell-label">Starting Balance</div>
          <div class="pcell-value">Rp ${(projection.starting_balance||0).toLocaleString()}</div>
        </div>
        <div class="projection-cell">
          <div class="pcell-label">Actual / Projected Income</div>
          <div class="pcell-value" style="color:var(--accent)">Rp ${(projection.projected_income||0).toLocaleString()}</div>
          <div class="pcell-sub">Actual: Rp ${(projection.actual_income||0).toLocaleString()} &nbsp;·&nbsp; Growth: ${incomeGrowth}%</div>
        </div>
        <div class="projection-cell">
          <div class="pcell-label">Actual / Projected Expense</div>
          <div class="pcell-value" style="color:var(--red)">Rp ${(projection.projected_expense||0).toLocaleString()}</div>
          <div class="pcell-sub">Actual: Rp ${(projection.actual_expense||0).toLocaleString()}</div>
        </div>
        <div class="projection-cell">
          <div class="pcell-label">Budget vs Actual</div>
          <div class="pcell-value" style="color:${(projection.budget_vs_actual||0)>=0?'var(--accent)':'var(--red)'}">
            Rp ${(projection.budget_vs_actual||0).toLocaleString()}
          </div>
        </div>
        <div class="projection-cell ${isPos ? 'highlight-pos' : 'highlight-neg'}">
          <div class="pcell-label">Projected Ending Balance</div>
          <div class="pcell-value" style="font-size:22px; color:${isPos?'var(--accent)':'var(--red)'}">
            Rp ${endBal.toLocaleString()}
          </div>
        </div>
      </div>
      ${projection.budgets?.length ? `
        <div style="margin-top:16px">
          <div style="font-size:12px;color:var(--text-muted);margin-bottom:8px;text-transform:uppercase;letter-spacing:.05em">Budget Categories</div>
          <div>${projection.budgets.map(b => `<span class="budget-tag">${b.category}: Rp ${(b.budget||0).toLocaleString()}</span>`).join('')}</div>
        </div>` : ''}
    `;
    document.getElementById('monthlyProjection').innerHTML = html;
  } catch (err) {
    document.getElementById('monthlyProjection').innerHTML = '<div class="empty-state"><i class="fas fa-chart-bar"></i><p>Add some transactions to see projection</p></div>';
  }
}

// ── Categories ──────────────────────────────────────────────────
async function loadCategories() {
  try {
    const res  = await fetch(`${API_URL}/categories`, { credentials: 'include' });
    const data = await res.json();
    allCategories = data.categories || [];
  } catch {
    allCategories = ['Food & Dining','Transportation','Shopping','Entertainment','Bills & Utilities','Healthcare','Education','Rent','Salary','Investment','Other'];
  }
  updateCategoryDropdowns();
}

function updateCategoryDropdowns() {
  ['category','budgetCategory'].forEach(id => {
    const sel = document.getElementById(id);
    if (!sel) return;
    const cur = sel.value;
    sel.innerHTML = '<option value="">Select category</option>';
    allCategories.forEach(cat => {
      const opt = document.createElement('option');
      opt.value = opt.textContent = cat;
      if (cat === cur) opt.selected = true;
      sel.appendChild(opt);
    });
  });
}

function showAddCategoryModal(context) {
  currentCategoryContext = context;
  document.getElementById('newCategoryName').value = '';
  document.getElementById('categoryModal').classList.add('open');
}
function closeCategoryModal() {
  document.getElementById('categoryModal').classList.remove('open');
  currentCategoryContext = null;
}
document.getElementById('categoryModalClose').addEventListener('click', closeCategoryModal);
document.getElementById('cancelCategoryBtn').addEventListener('click', closeCategoryModal);

async function addNewCategory() {
  const name = document.getElementById('newCategoryName').value.trim();
  if (!name) { toast('Please enter a category name', 'error'); return; }
  if (allCategories.includes(name)) { toast('Category already exists', 'error'); closeCategoryModal(); return; }
  allCategories.push(name);
  allCategories.sort();
  updateCategoryDropdowns();
  if (currentCategoryContext === 'transaction') document.getElementById('category').value = name;
  else if (currentCategoryContext === 'budget') document.getElementById('budgetCategory').value = name;
  closeCategoryModal();
  toast(`Category "${name}" added`);
}

// ── Transactions ────────────────────────────────────────────────
async function loadTransactions() {
  try {
    await loadAccounts();
    setDateInputIfEmpty('date');
    const filter = document.getElementById('txnAccountFilter')?.value || '';
    const url = filter
      ? `${API_URL}/transactions?account_uuid=${encodeURIComponent(filter)}`
      : `${API_URL}/transactions`;
    const res  = await fetch(url, { credentials: 'include' });
    const data = await res.json();
    const tbody = document.getElementById('transactionsList');
    tbody.innerHTML = '';
    if (!data.transactions?.length) {
      tbody.innerHTML = `<tr><td colspan="6"><div class="empty-state"><i class="fas fa-receipt"></i><p>Belum ada transaksi</p></div></td></tr>`;
      return;
    }
    const transactionCategories = new Set();
    const cur = (t) => (t.currency_code || 'IDR').toUpperCase();
    data.transactions.forEach(t => {
      transactionCategories.add(t.category);
      const row = tbody.insertRow();
      row.insertCell(0).textContent = new Date(t.transaction_date).toLocaleDateString('id-ID');
      row.insertCell(1).textContent = accountDisplayName(t.account_uuid);
      const amtCell = row.insertCell(2);
      const sign = t.type === 'income' ? '+' : '−';
      const isXfer = String(t.category).toLowerCase() === 'transfer';
      amtCell.innerHTML = `<span class="amount-${t.type}">${sign} ${formatMoney(t.amount, cur(t))}</span>`;
      const catCell = row.insertCell(3);
      if (isXfer) {
        catCell.innerHTML = `<span class="badge badge-${t.type} badge-transfer"><i class="fas fa-random" style="margin-right:4px"></i>${escapeHtml(t.category)}</span>`;
      } else {
        catCell.innerHTML = `<span class="badge badge-${t.type}">${escapeHtml(t.category)}</span>`;
      }
      row.insertCell(4).textContent = t.description || '—';
      const act = row.insertCell(5);
      if (isXfer) {
        if (t.transfer_uuid) {
          act.innerHTML = `<button class="btn btn-sm btn-danger" onclick="deleteTransfer('${t.transfer_uuid}')"><i class="fas fa-trash"></i></button>`;
        } else {
          act.innerHTML = '<span style="font-size:12px;color:var(--text-muted)">—</span>';
        }
      } else {
        act.innerHTML = `<button class="btn btn-sm btn-danger" onclick="deleteTransaction('${t.uuid}')"><i class="fas fa-trash"></i></button>`;
      }
    });
    transactionCategories.forEach(cat => { if (!allCategories.includes(cat)) allCategories.push(cat); });
    allCategories.sort();
    updateCategoryDropdowns();
  } catch (err) { console.error(err); }
}

async function addTransaction() {
  const amount      = document.getElementById('amount').value;
  const type        = document.getElementById('type').value;
  const category    = document.getElementById('category').value;
  const description = document.getElementById('description').value;
  const date        = document.getElementById('date').value;
  if (!amount || !category || !date) { toast('Please fill all required fields', 'error'); return; }
  if (!allCategories.includes(category)) { allCategories.push(category); allCategories.sort(); updateCategoryDropdowns(); }
  const accountUuid = document.getElementById('txnAccountSelect')?.value || '';
  const payload = { amount: parseFloat(amount), type, category, description, transaction_date: date };
  if (accountUuid) payload.account_uuid = accountUuid;
  try {
    const res = await fetch(`${API_URL}/transactions`, { 
        method: 'POST', 
        headers: { 'Content-Type': 'application/json' }, 
        body: JSON.stringify(payload),
        credentials: 'include'
    });
    if (res.ok) {
      toast('Transaction added successfully');
      clearTransactionForm();
      loadTransactions();
      loadDashboard();
    } else {
      const data = await res.json();
      toast(data.error || 'Failed to add transaction', 'error');
    }
  } catch { toast('Error adding transaction', 'error'); }
}

async function deleteTransfer(transferUUID) {
  if (!transferUUID) return;
  if (!confirm('Hapus transfer ini? Ini akan menghapus 2 transaksi (debit dan kredit).')) return;
  try {
    const res = await fetch(`${API_URL}/transfers/${transferUUID}`, { method: 'DELETE', credentials: 'include' });
    const data = await res.json().catch(() => ({}));
    if (res.ok) {
      toast('Transfer dihapus');
      await loadTransactions();
      await loadAccountsPage();
      loadDashboard();
    } else {
      toast(data.error || 'Gagal menghapus transfer', 'error');
    }
  } catch {
    toast('Koneksi error', 'error');
  }
}

async function deleteTransaction(uuid) {
  if (!confirm('Delete this transaction?')) return;
  try {
    const res = await fetch(`${API_URL}/transactions/${uuid}`, { method: 'DELETE', credentials: 'include' });
    if (res.ok) { toast('Transaction deleted'); loadTransactions(); }
  } catch { toast('Error deleting transaction', 'error'); }
}

function clearTransactionForm() {
  ['amount','description','date'].forEach(id => document.getElementById(id).value = '');
  document.getElementById('category').value = '';
}

// ── Budgets ─────────────────────────────────────────────────────
async function setBudget() {
  const category = document.getElementById('budgetCategory').value;
  const amount   = document.getElementById('budgetAmount').value;
  const month    = document.getElementById('budgetMonth').value;
  const year     = document.getElementById('budgetYear').value;
  if (!category || !amount || !month || !year) { toast('Please fill all fields', 'error'); return; }
  if (!allCategories.includes(category)) { allCategories.push(category); allCategories.sort(); updateCategoryDropdowns(); }
  try {
    const res = await fetch(`${API_URL}/budgets`, { method: 'POST', 
        headers: { 'Content-Type': 'application/json' }, 
        body: JSON.stringify({ category, amount: parseFloat(amount), month: parseInt(month), year: parseInt(year) }),
        credentials: 'include'
    });
    if (res.ok) { toast('Budget set successfully'); loadBudgets(); clearBudgetForm(); }
    else { const d = await res.json(); toast(d.error || 'Failed to set budget', 'error'); }
  } catch { toast('Error setting budget', 'error'); }
}

async function deleteBudget(uuid) {
  if (!confirm('Delete this budget?')) return;
  try {
    const res = await fetch(`${API_URL}/budgets/${uuid}`, { method: 'DELETE', credentials: 'include' });
    if (res.ok) { toast('Budget deleted'); loadBudgets(); }
    else { const d = await res.json(); toast(d.error || 'Failed to delete', 'error'); }
  } catch { toast('Error deleting budget', 'error'); }
}

function editBudget(uuid, oldCategory, oldAmount, oldMonth, oldYear) {
  window.editingBudgetUUID        = uuid;
  window.editingBudgetOldCategory = oldCategory;
  document.getElementById('budgetCategory').value = oldCategory;
  document.getElementById('budgetAmount').value   = oldAmount;
  document.getElementById('budgetMonth').value    = oldMonth;
  document.getElementById('budgetYear').value     = oldYear;
  const btn = document.getElementById('setBudgetBtn');
  btn.innerHTML = '<i class="fas fa-save"></i> Update Budget';
  btn.className = 'btn btn-warning';
  btn.onclick   = () => updateBudget(uuid);
  if (!document.getElementById('cancelEditBudgetBtn')) {
    const cancel = document.createElement('button');
    cancel.id        = 'cancelEditBudgetBtn';
    cancel.className = 'btn btn-muted';
    cancel.innerHTML = '<i class="fas fa-times"></i> Cancel';
    cancel.onclick   = resetBudgetFormButton;
    btn.parentNode.insertBefore(cancel, btn.nextSibling);
  }
}

async function updateBudget(uuid) {
  const category = document.getElementById('budgetCategory').value;
  const amount   = document.getElementById('budgetAmount').value;
  const month    = document.getElementById('budgetMonth').value;
  const year     = document.getElementById('budgetYear').value;
  if (!category || !amount || !month || !year) { toast('Please fill all fields', 'error'); return; }
  try {
    const res = await fetch(`${API_URL}/budgets/${uuid}`, { 
        method: 'PUT', headers: { 'Content-Type': 'application/json' }, 
        body: JSON.stringify({ category, amount: parseFloat(amount), month: parseInt(month), year: parseInt(year) }),
        credentials: 'include'
    });
    if (res.ok) {
      toast('Budget updated');
      if (window.editingBudgetOldCategory !== category) await loadCategories();
      loadBudgets(); clearBudgetForm(); resetBudgetFormButton();
    } else { const d = await res.json(); toast(d.error || 'Failed to update', 'error'); }
  } catch { toast('Error updating budget', 'error'); }
}

function resetBudgetFormButton() {
  const btn = document.getElementById('setBudgetBtn');
  btn.innerHTML = '<i class="fas fa-plus"></i> Set Budget';
  btn.className = 'btn btn-accent';
  btn.onclick   = setBudget;
  document.getElementById('cancelEditBudgetBtn')?.remove();
  window.editingBudgetUUID = null;
  window.editingBudgetOldCategory = null;
  clearBudgetForm();
}

function clearBudgetForm() {
  ['budgetCategory','budgetAmount','budgetMonth','budgetYear'].forEach(id => document.getElementById(id).value = '');
}

async function loadBudgets() {
  try {
    const now   = new Date();
    const res   = await fetch(`${API_URL}/budgets?month=${now.getMonth()+1}&year=${now.getFullYear()}`, { credentials: 'include' });
    const data  = await res.json();
    const div   = document.getElementById('budgetsList');
    div.innerHTML = '';
    if (!data.budgets?.length) {
      div.innerHTML = '<div class="empty-state"><i class="fas fa-piggy-bank"></i><p>No budgets for this month. Create one above!</p></div>';
      return;
    }
    data.budgets.forEach(budget => {
      const pct = budget.percentage || 0;
      const cls = pct >= 90 ? 'danger' : pct >= 70 ? 'warning' : '';
      div.innerHTML += `
        <div class="budget-item">
          <div class="budget-meta">
            <div>
              <span class="budget-cat">${budget.category}</span>
              <span class="budget-date-badge" style="margin-left:8px">${budget.month}/${budget.year}</span>
            </div>
            <div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap">
              <span class="budget-amounts">Rp ${budget.spent.toLocaleString()} / Rp ${budget.amount.toLocaleString()}</span>
              <div class="budget-actions">
                <button class="btn btn-sm btn-edit" onclick="editBudget('${budget.uuid}','${budget.category}',${budget.amount},${budget.month},${budget.year})">
                  <i class="fas fa-edit"></i>
                </button>
                <button class="btn btn-sm btn-danger" onclick="deleteBudget('${budget.uuid}')">
                  <i class="fas fa-trash"></i>
                </button>
              </div>
            </div>
          </div>
          <div class="progress-track">
            <div class="progress-fill ${cls}" style="width:${Math.min(pct,100)}%"></div>
          </div>
          <div class="progress-pct">${pct.toFixed(1)}%</div>
        </div>`;
    });
  } catch (err) { console.error(err); }
}

// ── Recommendations ─────────────────────────────────────────────
async function generateRecommendations() {
  try {
    const res = await fetch(`${API_URL}/recommendations/generate`, { method: 'POST', credentials: 'include' });
    if (res.ok) { toast('Recommendations generated'); loadRecommendations(); }
  } catch { toast('Error generating recommendations', 'error'); }
}

async function loadRecommendations() {
  try {
    const res  = await fetch(`${API_URL}/recommendations`, { credentials: 'include' });
    const data = await res.json();
    const div  = document.getElementById('recommendationsList');
    div.innerHTML = '';
    if (!data.recommendations?.length) {
      div.innerHTML = '<div class="empty-state"><i class="fas fa-lightbulb"></i><p>No recommendations yet. Click "Generate" to get insights!</p></div>';
      return;
    }
    data.recommendations.forEach(rec => {
      div.innerHTML += `
        <div class="rec-item">
          <i class="fas fa-lightbulb"></i>
          <div>
            ${rec.recommendation}
            <div class="rec-date">${new Date(rec.created_at).toLocaleDateString('id-ID')}</div>
          </div>
        </div>`;
    });
  } catch (err) { console.error(err); }
}

// ── Export / Import ─────────────────────────────────────────────
function exportData(format) {
  window.open(`${API_URL}/export?format=${format}`, '_blank');
}

async function importData() {
  const fileInput = document.getElementById('importFile');
  const file      = fileInput.files[0];
  if (!file) { toast('Please select a file', 'error'); return; }
  const formData  = new FormData();
  formData.append('file', file);
  try {
    const res  = await fetch(`${API_URL}/import`, { method: 'POST', credentials: 'include', body: formData });
    const data = await res.json();
    if (res.ok) { toast(`Successfully imported ${data.count} transactions`); loadTransactions(); }
    else toast(data.error || 'Import failed', 'error');
  } catch { toast('Import error', 'error'); }
}

async function checkAuth() {
  try {
    const statsRes  = await fetch(`${API_URL}/balance/current`, { credentials: 'include' });
    if (statsRes.ok) {
        showDashboard();
        loadDashboard();
    }
  } catch (err) {
    toast('Check auth Error', 'error');
  }
}

// ── Init ────────────────────────────────────────────────────────
document.getElementById('txnAccountFilter')?.addEventListener('change', () => loadTransactions());

checkAuth();
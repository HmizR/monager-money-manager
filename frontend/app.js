/* ============================================================
   Money Manager — app.js
   ============================================================ */

'use strict';

// ── Config ─────────────────────────────────────────────────────
const API_URL = 'http://localhost:8084/api';

// ── State ──────────────────────────────────────────────────────
let token             = localStorage.getItem('token');
let categoryChart     = null;
let trendChart        = null;
let balanceTrendChart = null;
let globalRange       = '30D';
let currentCategoryContext = null;
let allCategories     = [];

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

// ── Auth: Login ────────────────────────────────────────────────
document.getElementById('loginForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const username = document.getElementById('loginUsername').value;
  const password = document.getElementById('loginPassword').value;
  try {
    const res  = await fetch(`${API_URL}/auth/login`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }) });
    const data = await res.json();
    if (res.ok) {
      localStorage.setItem('token', data.token);
      localStorage.setItem('user', JSON.stringify(data.user));
      token = data.token;
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
    const res  = await fetch(`${API_URL}/auth/register`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, email, password }) });
    const data = await res.json();
    if (res.ok) {
      localStorage.setItem('token', data.token);
      localStorage.setItem('user', JSON.stringify(data.user));
      token = data.token;
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

function logout() {
  localStorage.removeItem('token');
  localStorage.removeItem('user');
  token = null;
  document.getElementById('dashboard').style.display = 'none';
  document.getElementById('loginPage').style.display = 'flex';
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
    const titles = { dashboard: 'Dashboard', transactions: 'Transactions', budgets: 'Budgets', recommendations: 'Recommendations', export: 'Export / Import' };
    document.getElementById('pageTitle').textContent = titles[page] || page;
    closeMobileSidebar();
    if (page === 'dashboard')       loadDashboard();
    else if (page === 'transactions') { loadCategories(); loadTransactions(); }
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
    const statsRes  = await fetch(`${API_URL}/statistics?range=${globalRange}`, { headers: { Authorization: `Bearer ${token}` } });
    const stats     = await statsRes.json();
    const net = (stats.total_income || 0) - (stats.total_expense || 0);
    const netClass  = net >= 0 ? 'netchange' : 'expense';

    document.getElementById('statIncome').textContent  = `Rp ${(stats.total_income || 0).toLocaleString()}`;
    document.getElementById('statExpense').textContent = `Rp ${(stats.total_expense || 0).toLocaleString()}`;
    document.getElementById('statNet').textContent     = `Rp ${net.toLocaleString()}`;

    // Current balance
    const balRes  = await fetch(`${API_URL}/balance/current`, { headers: { Authorization: `Bearer ${token}` } });
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
    const res  = await fetch(`${API_URL}/charts?type=trend&range=${globalRange}`, { headers: { Authorization: `Bearer ${token}` } });
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
    const res  = await fetch(`${API_URL}/charts?type=category`, { headers: { Authorization: `Bearer ${token}` } });
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
    const res  = await fetch(`${API_URL}/balance/trend?range=${globalRange}`, { headers: { Authorization: `Bearer ${token}` } });
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
    const res      = await fetch(`${API_URL}/balance/forecast`, { headers: { Authorization: `Bearer ${token}` } });
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
    const res        = await fetch(`${API_URL}/balance/projection`, { headers: { Authorization: `Bearer ${token}` } });
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
    const res  = await fetch(`${API_URL}/categories`, { headers: { Authorization: `Bearer ${token}` } });
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
    const res  = await fetch(`${API_URL}/transactions`, { headers: { Authorization: `Bearer ${token}` } });
    const data = await res.json();
    const tbody = document.getElementById('transactionsList');
    tbody.innerHTML = '';
    if (!data.transactions?.length) {
      tbody.innerHTML = `<tr><td colspan="5"><div class="empty-state"><i class="fas fa-receipt"></i><p>No transactions yet</p></div></td></tr>`;
      return;
    }
    const transactionCategories = new Set();
    data.transactions.forEach(t => {
      transactionCategories.add(t.category);
      const row = tbody.insertRow();
      row.insertCell(0).textContent = new Date(t.transaction_date).toLocaleDateString('id-ID');
      row.insertCell(1).innerHTML = `<span class="amount-${t.type}">${t.type === 'income' ? '+' : '−'} Rp ${t.amount.toLocaleString()}</span>`;
      row.insertCell(2).innerHTML = `<span class="badge badge-${t.type}">${t.category}</span>`;
      row.insertCell(3).textContent = t.description || '—';
      row.insertCell(4).innerHTML = `<button class="btn btn-sm btn-danger" onclick="deleteTransaction('${t.uuid}')"><i class="fas fa-trash"></i></button>`;
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
  try {
    const res = await fetch(`${API_URL}/transactions`, { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ amount: parseFloat(amount), type, category, description, transaction_date: date }) });
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

async function deleteTransaction(uuid) {
  if (!confirm('Delete this transaction?')) return;
  try {
    const res = await fetch(`${API_URL}/transactions/${uuid}`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } });
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
    const res = await fetch(`${API_URL}/budgets`, { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ category, amount: parseFloat(amount), month: parseInt(month), year: parseInt(year) }) });
    if (res.ok) { toast('Budget set successfully'); loadBudgets(); clearBudgetForm(); }
    else { const d = await res.json(); toast(d.error || 'Failed to set budget', 'error'); }
  } catch { toast('Error setting budget', 'error'); }
}

async function deleteBudget(uuid) {
  if (!confirm('Delete this budget?')) return;
  try {
    const res = await fetch(`${API_URL}/budgets/${uuid}`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } });
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
    const res = await fetch(`${API_URL}/budgets/${uuid}`, { method: 'PUT', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ category, amount: parseFloat(amount), month: parseInt(month), year: parseInt(year) }) });
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
    const res   = await fetch(`${API_URL}/budgets?month=${now.getMonth()+1}&year=${now.getFullYear()}`, { headers: { Authorization: `Bearer ${token}` } });
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
    const res = await fetch(`${API_URL}/recommendations/generate`, { method: 'POST', headers: { Authorization: `Bearer ${token}` } });
    if (res.ok) { toast('Recommendations generated'); loadRecommendations(); }
  } catch { toast('Error generating recommendations', 'error'); }
}

async function loadRecommendations() {
  try {
    const res  = await fetch(`${API_URL}/recommendations`, { headers: { Authorization: `Bearer ${token}` } });
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
  window.open(`${API_URL}/export?format=${format}&token=${token}`, '_blank');
}

async function importData() {
  const fileInput = document.getElementById('importFile');
  const file      = fileInput.files[0];
  if (!file) { toast('Please select a file', 'error'); return; }
  const formData  = new FormData();
  formData.append('file', file);
  try {
    const res  = await fetch(`${API_URL}/import`, { method: 'POST', headers: { Authorization: `Bearer ${token}` }, body: formData });
    const data = await res.json();
    if (res.ok) { toast(`Successfully imported ${data.count} transactions`); loadTransactions(); }
    else toast(data.error || 'Import failed', 'error');
  } catch { toast('Import error', 'error'); }
}

// ── Init ────────────────────────────────────────────────────────
if (token) {
  showDashboard();
  loadDashboard();
}

// API Configuration
const API_BASE = '/api';
const API_ENDPOINTS = {
	dashboard: `${API_BASE}/dashboard`,
	workflows: `${API_BASE}/workflows`,
	triage: `${API_BASE}/github/triage`,
	health: `${API_BASE}/health`,
};

// Global state
let currentPage = 'dashboard';
let charts = {};
let workflows = [];
let triageData = {};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
	initializeNavigation();
	initializeButtons();
	loadDashboard();
	startAutoRefresh();
	checkHealth();
});

// Navigation
function initializeNavigation() {
	document.querySelectorAll('.nav-item').forEach(item => {
		item.addEventListener('click', (e) => {
			e.preventDefault();
			const page = item.dataset.page;
			navigateTo(page);
		});
	});
}

function navigateTo(page) {
	// Hide all pages
	document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));

	// Show selected page
	const pageEl = document.getElementById(`page-${page}`);
	if (pageEl) {
		pageEl.classList.add('active');
	}

	// Update nav
	document.querySelectorAll('.nav-item').forEach(item => {
		item.classList.toggle('active', item.dataset.page === page);
	});

	// Update header
	const titles = {
		dashboard: 'Dashboard',
		workflows: 'Workflows',
		triage: 'Issue Triage',
		reviews: 'Code Reviews',
		bugs: 'Bug Detection',
		docs: 'Documentation',
		settings: 'Settings',
	};

	document.getElementById('page-title').textContent = titles[page] || 'Dashboard';
	document.getElementById('page-subtitle').textContent = getSubtitle(page);

	// Load page data
	loadPageData(page);
	currentPage = page;
}

function getSubtitle(page) {
	const subtitles = {
		dashboard: 'Real-time factory operations',
		workflows: 'View and manage workflows',
		triage: 'GitHub issue triage results',
		reviews: 'Code review findings',
		bugs: 'Detected and fixed bugs',
		docs: 'Generated documentation',
		settings: 'System configuration',
	};
	return subtitles[page] || '';
}

// Buttons
function initializeButtons() {
	document.getElementById('refresh-btn').addEventListener('click', () => {
		loadDashboard();
	});

	document.getElementById('new-workflow-btn').addEventListener('click', () => {
		showNewWorkflowModal();
	});

	document.getElementById('modal-close').addEventListener('click', () => {
		closeModal();
	});

	document.getElementById('save-settings-btn')?.addEventListener('click', () => {
		saveSettings();
	});

	// Modal close on outside click
	document.getElementById('modal').addEventListener('click', (e) => {
		if (e.target.id === 'modal') {
			closeModal();
		}
	});
}

// Load Data
async function loadDashboard() {
	try {
		const response = await fetch(API_ENDPOINTS.dashboard);
		const data = await response.json();

		// Update metrics
		document.getElementById('metric-total-issues').textContent = data.triage?.total_analyzed || 0;
		document.getElementById('metric-ready-count').textContent = data.triage?.ready_count || 0;
		document.getElementById('metric-active-workflows').textContent = data.active_workflows || 0;
		document.getElementById('metric-success-rate').textContent = calculateSuccessRate(data) + '%';

		// Update charts
		updateIssueChart(data.triage?.by_category || {});
		updateWorkflowChart(data.workflow_summary || {});

		// Update recent workflows
		updateRecentWorkflows(data.recent_workflows || []);

		// Update priority issues
		updatePriorityIssues(data.triage?.by_priority || {});

		triageData = data;
	} catch (error) {
		console.error('Failed to load dashboard:', error);
		showError('Failed to load dashboard data');
	}
}

async function loadPageData(page) {
	switch (page) {
		case 'workflows':
			loadWorkflows();
			break;
		case 'triage':
			loadTriageData();
			break;
		case 'reviews':
			loadReviews();
			break;
		case 'bugs':
			loadBugs();
			break;
		case 'docs':
			loadDocs();
			break;
		case 'settings':
			loadSettings();
			break;
	}
}

async function loadWorkflows() {
	try {
		const grid = document.getElementById('workflows-grid');
		grid.innerHTML = '<div class="loading">Loading workflows...</div>';

		const response = await fetch(`${API_ENDPOINTS.workflows}`);
		const data = await response.json();
		workflows = data.workflows || [];

		if (workflows.length === 0) {
			grid.innerHTML = '<div class="empty-state">No workflows found</div>';
			return;
		}

		grid.innerHTML = workflows.map(w => `
			<div class="workflow-card" onclick="showWorkflowDetail('${w.id}')">
				<div class="card-title">${w.type}</div>
				<div class="card-text">${w.description || 'No description'}</div>
				<div style="margin: 1rem 0;">
					<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
						<span style="font-size: 0.875rem; color: var(--text-secondary);">Progress</span>
						<span style="font-weight: 600;">${w.progress || 0}%</span>
					</div>
					<div class="progress-bar">
						<div class="progress-fill" style="width: ${w.progress || 0}%"></div>
					</div>
				</div>
				<div style="display: flex; justify-content: space-between; align-items: center;">
					<span class="status-badge status-${w.status}">${w.status}</span>
					<span style="font-size: 0.875rem; color: var(--text-secondary);">${formatDuration(w.duration)}</span>
				</div>
			</div>
		`).join('');
	} catch (error) {
		console.error('Failed to load workflows:', error);
		document.getElementById('workflows-grid').innerHTML = '<div class="empty-state">Failed to load workflows</div>';
	}
}

async function loadTriageData() {
	try {
		const grid = document.getElementById('triage-grid');
		grid.innerHTML = '<div class="loading">Loading triage data...</div>';

		if (!triageData || !triageData.triage) {
			await loadDashboard();
		}

		const data = triageData.triage || {};
		document.getElementById('triage-total').textContent = data.total_analyzed || 0;

		const priorityStr = `🔴 ${data.by_priority?.critical || 0} | 🟠 ${data.by_priority?.high || 0} | 🟡 ${data.by_priority?.medium || 0}`;
		document.getElementById('triage-priority').textContent = priorityStr;

		const issues = [];
		for (const [priority, count] of Object.entries(data.by_priority || {})) {
			for (let i = 0; i < Math.min(count, 2); i++) {
				issues.push({ priority, count });
			}
		}

		if (issues.length === 0) {
			grid.innerHTML = '<div class="empty-state">No issues to triage</div>';
			return;
		}

		grid.innerHTML = issues.map((issue, idx) => `
			<div class="issue-card">
				<div class="card-title">Issue #${idx + 1}</div>
				<div class="card-text">Priority: ${issue.priority}</div>
				<div style="display: flex; gap: 0.5rem;">
					<button class="btn btn-primary" style="flex: 1; font-size: 0.875rem;">Analyze</button>
					<button class="btn btn-secondary" style="flex: 1; font-size: 0.875rem;">Skip</button>
				</div>
			</div>
		`).join('');
	} catch (error) {
		console.error('Failed to load triage data:', error);
	}
}

async function loadReviews() {
	try {
		const list = document.getElementById('reviews-list');
		list.innerHTML = '<div class="loading">Loading reviews...</div>';

		// Fetch recent reviews from dashboard
		if (!triageData) await loadDashboard();

		const reviews = triageData.recent_reviews || [];
		if (reviews.length === 0) {
			list.innerHTML = '<div class="empty-state">No reviews found</div>';
			return;
		}

		list.innerHTML = reviews.map(r => `
			<div class="card" style="margin-bottom: 1rem;">
				<div class="card-header">
					<div>
						<h4>${r.file_name}</h4>
						<p style="font-size: 0.875rem; color: var(--text-secondary);">Score: ${r.score}/100</p>
					</div>
					<span class="status-badge status-${r.approved ? 'success' : 'running'}">${r.approved ? 'Approved' : 'Needs Review'}</span>
				</div>
				<div style="padding: 1rem; border-bottom: 1px solid var(--border);">
					<p>${r.summary || 'No summary available'}</p>
				</div>
				<div style="padding: 1rem;">
					<button class="btn btn-primary" onclick="showReviewDetail('${r.id}')">View Details</button>
				</div>
			</div>
		`).join('');
	} catch (error) {
		console.error('Failed to load reviews:', error);
	}
}

async function loadBugs() {
	try {
		const grid = document.getElementById('bugs-grid');
		grid.innerHTML = '<div class="loading">Loading bugs...</div>';

		if (!triageData) await loadDashboard();

		const bugs = triageData.bugs || [];
		document.getElementById('bugs-total').textContent = bugs.length;
		document.getElementById('bugs-fixed').textContent = bugs.filter(b => b.fixed).length;
		document.getElementById('bugs-critical').textContent = bugs.filter(b => b.severity === 'critical').length;

		if (bugs.length === 0) {
			grid.innerHTML = '<div class="empty-state">No bugs detected</div>';
			return;
		}

		grid.innerHTML = bugs.map(b => `
			<div class="bug-card">
				<div class="card-title">${b.type || 'Bug'}</div>
				<div class="card-text">${b.description || 'No description'}</div>
				<div style="margin: 1rem 0; font-size: 0.875rem;">
					<div>Severity: <strong>${b.severity}</strong></div>
					<div>Confidence: <strong>${b.confidence || 0}%</strong></div>
				</div>
				<button class="btn btn-primary" style="width: 100%; font-size: 0.875rem;">Fix</button>
			</div>
		`).join('');
	} catch (error) {
		console.error('Failed to load bugs:', error);
	}
}

async function loadDocs() {
	try {
		const grid = document.getElementById('docs-grid');
		grid.innerHTML = '<div class="loading">Loading documentation...</div>';

		if (!triageData) await loadDashboard();

		const docs = triageData.documentation || [];
		if (docs.length === 0) {
			grid.innerHTML = '<div class="empty-state">No documentation found</div>';
			return;
		}

		grid.innerHTML = docs.map(d => `
			<div class="doc-card">
				<div class="card-title">${d.title}</div>
				<div class="card-text">${d.description || 'Documentation file'}</div>
				<div style="display: flex; gap: 0.5rem;">
					<button class="btn btn-primary" style="flex: 1; font-size: 0.875rem;">View</button>
					<button class="btn btn-secondary" style="flex: 1; font-size: 0.875rem;">Edit</button>
				</div>
			</div>
		`).join('');
	} catch (error) {
		console.error('Failed to load docs:', error);
	}
}

async function loadSettings() {
	try {
		// Load system status
		const response = await fetch(API_ENDPOINTS.health);
		const health = await response.json();

		const statusGrid = document.getElementById('system-status');
		statusGrid.innerHTML = `
			<div class="status-item">
				<span class="dot ${health.checks?.mcp ? 'healthy' : 'error'}"></span>
				<div>
					<div style="font-weight: 500;">MCP Server</div>
					<div style="font-size: 0.75rem; color: var(--text-secondary);">${health.checks?.mcp ? 'Connected' : 'Disconnected'}</div>
				</div>
			</div>
			<div class="status-item">
				<span class="dot ${health.checks?.github ? 'healthy' : 'error'}"></span>
				<div>
					<div style="font-weight: 500;">GitHub API</div>
					<div style="font-size: 0.75rem; color: var(--text-secondary);">${health.checks?.github ? 'Connected' : 'Disconnected'}</div>
				</div>
			</div>
			<div class="status-item">
				<span class="dot ${health.checks?.agents ? 'healthy' : 'error'}"></span>
				<div>
					<div style="font-weight: 500;">Agents</div>
					<div style="font-size: 0.75rem; color: var(--text-secondary);">${health.checks?.agents ? 'Ready' : 'Offline'}</div>
				</div>
			</div>
			<div class="status-item">
				<span class="dot ${health.checks?.storage ? 'healthy' : 'error'}"></span>
				<div>
					<div style="font-weight: 500;">Storage</div>
					<div style="font-size: 0.75rem; color: var(--text-secondary);">${health.checks?.storage ? 'Connected' : 'Disconnected'}</div>
				</div>
			</div>
		`;
	} catch (error) {
		console.error('Failed to load settings:', error);
	}
}

// Charts
function updateIssueChart(categories) {
	const ctx = document.getElementById('issues-chart')?.getContext('2d');
	if (!ctx) return;

	if (charts.issues) charts.issues.destroy();

	charts.issues = new Chart(ctx, {
		type: 'doughnut',
		data: {
			labels: ['Feature', 'Bug', 'Documentation', 'Refactor'],
			datasets: [{
				data: [
					categories.feature || 0,
					categories.bug || 0,
					categories.documentation || 0,
					categories.refactor || 0,
				],
				backgroundColor: ['#3b82f6', '#ef4444', '#10b981', '#f59e0b'],
				borderColor: '#1e293b',
				borderWidth: 2,
			}],
		},
		options: {
			responsive: true,
			maintainAspectRatio: false,
			plugins: {
				legend: {
					position: 'bottom',
					labels: { color: '#cbd5e1', font: { size: 12 } },
				},
			},
		},
	});
}

function updateWorkflowChart(summary) {
	const ctx = document.getElementById('workflow-chart')?.getContext('2d');
	if (!ctx) return;

	if (charts.workflow) charts.workflow.destroy();

	const days = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
	const data = days.map(() => Math.floor(Math.random() * 20));

	charts.workflow = new Chart(ctx, {
		type: 'bar',
		data: {
			labels: days,
			datasets: [
				{
					label: 'Success',
					data: data.map(d => Math.floor(d * 0.8)),
					backgroundColor: '#10b981',
				},
				{
					label: 'Failed',
					data: data.map(d => Math.floor(d * 0.2)),
					backgroundColor: '#ef4444',
				},
			],
		},
		options: {
			responsive: true,
			maintainAspectRatio: false,
			plugins: {
				legend: {
					labels: { color: '#cbd5e1', font: { size: 12 } },
				},
			},
			scales: {
				x: { ticks: { color: '#cbd5e1' }, grid: { color: '#475569' } },
				y: { ticks: { color: '#cbd5e1' }, grid: { color: '#475569' } },
			},
			indexAxis: undefined,
			stacked: true,
		},
	});
}

// Update DOM elements
function updateRecentWorkflows(workflows) {
	const tbody = document.getElementById('recent-workflows');
	if (workflows.length === 0) {
		tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No workflows yet</td></tr>';
		return;
	}

	tbody.innerHTML = workflows.slice(0, 5).map(w => `
		<tr>
			<td><strong>${w.type}</strong></td>
			<td><span class="status-badge status-${w.status}">${w.status}</span></td>
			<td>
				<div style="display: flex; align-items: center; gap: 0.5rem;">
					<div class="progress-bar" style="flex: 1; height: 6px;">
						<div class="progress-fill" style="width: ${w.progress || 0}%"></div>
					</div>
					<span style="font-size: 0.75rem;">${w.progress || 0}%</span>
				</div>
			</td>
			<td>${formatDuration(w.duration)}</td>
			<td><button class="btn btn-secondary" style="font-size: 0.75rem;" onclick="showWorkflowDetail('${w.id}')">View</button></td>
		</tr>
	`).join('');
}

function updatePriorityIssues(priorities) {
	const container = document.getElementById('priority-issues');
	const count = (priorities.critical || 0) + (priorities.high || 0);
	document.getElementById('priority-count').textContent = count;

	if (count === 0) {
		container.innerHTML = '<div class="empty-state">No priority issues</div>';
		return;
	}

	let html = '';
	for (const [priority, num] of Object.entries(priorities)) {
		if (priority === 'critical' || priority === 'high') {
			const icon = priority === 'critical' ? '🔴' : '🟠';
			html += `
				<div class="issue-item">
					<div>
						<div class="issue-title">${icon} ${priority.toUpperCase()}</div>
						<div class="issue-meta">${num} ${num === 1 ? 'issue' : 'issues'}</div>
					</div>
					<button class="btn btn-secondary" style="font-size: 0.75rem;">Review</button>
				</div>
			`;
		}
	}
	container.innerHTML = html;
}

// Modals
function showNewWorkflowModal() {
	const modal = document.getElementById('modal');
	document.getElementById('modal-title').textContent = 'New Workflow';
	document.getElementById('modal-body').innerHTML = `
		<div class="settings-group">
			<label>Workflow Type</label>
			<select class="input" id="workflow-type">
				<option value="spec-to-code">Spec to Code</option>
				<option value="code-review">Code Review</option>
				<option value="bug-detection">Bug Detection</option>
				<option value="docs">Generate Docs</option>
			</select>
			<label>Description</label>
			<textarea class="input" id="workflow-desc" style="min-height: 100px;"></textarea>
		</div>
	`;
	document.getElementById('modal-footer').innerHTML = `
		<button class="btn btn-secondary" onclick="closeModal()">Cancel</button>
		<button class="btn btn-primary" onclick="createWorkflow()">Create</button>
	`;
	modal.classList.add('active');
}

function showWorkflowDetail(id) {
	const workflow = workflows.find(w => w.id === id);
	if (!workflow) return;

	const modal = document.getElementById('modal');
	document.getElementById('modal-title').textContent = workflow.type;
	document.getElementById('modal-body').innerHTML = `
		<div style="display: grid; gap: 1rem;">
			<div>
				<strong>Status:</strong>
				<span class="status-badge status-${workflow.status}">${workflow.status}</span>
			</div>
			<div>
				<strong>Progress:</strong>
				<div class="progress-bar" style="margin-top: 0.5rem;">
					<div class="progress-fill" style="width: ${workflow.progress || 0}%"></div>
				</div>
				${workflow.progress || 0}%
			</div>
			<div>
				<strong>Duration:</strong>
				${formatDuration(workflow.duration)}
			</div>
			<div>
				<strong>Results:</strong>
				<pre style="background: var(--bg-tertiary); padding: 1rem; border-radius: 0.5rem; overflow-x: auto; max-height: 300px;">${JSON.stringify(workflow.result || {}, null, 2)}</pre>
			</div>
		</div>
	`;
	document.getElementById('modal-footer').innerHTML = `
		<button class="btn btn-secondary" onclick="closeModal()">Close</button>
	`;
	modal.classList.add('active');
}

function showReviewDetail(id) {
	const modal = document.getElementById('modal');
	document.getElementById('modal-title').textContent = 'Review Details';
	document.getElementById('modal-body').innerHTML = `
		<div style="display: grid; gap: 1rem;">
			<p>Review ID: ${id}</p>
			<p>Detailed review information would appear here.</p>
		</div>
	`;
	document.getElementById('modal-footer').innerHTML = `
		<button class="btn btn-secondary" onclick="closeModal()">Close</button>
	`;
	modal.classList.add('active');
}

function closeModal() {
	document.getElementById('modal').classList.remove('active');
}

// Settings
async function saveSettings() {
	const settings = {
		ollama_url: document.getElementById('setting-ollama-url').value,
		ollama_model: document.getElementById('setting-ollama-model').value,
	};

	try {
		// In a real implementation, send to server
		localStorage.setItem('asf-settings', JSON.stringify(settings));
		showSuccess('Settings saved!');
	} catch (error) {
		showError('Failed to save settings');
	}
}

// Workflow Operations
async function createWorkflow() {
	const type = document.getElementById('workflow-type').value;
	const desc = document.getElementById('workflow-desc').value;

	try {
		const response = await fetch(`${API_ENDPOINTS.workflows}`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ type, description: desc }),
		});

		if (response.ok) {
			showSuccess('Workflow created!');
			closeModal();
			loadWorkflows();
		} else {
			showError('Failed to create workflow');
		}
	} catch (error) {
		console.error('Failed to create workflow:', error);
		showError('Failed to create workflow');
	}
}

// Health Check
async function checkHealth() {
	try {
		const response = await fetch(API_ENDPOINTS.health);
		const health = await response.json();
		const status = document.querySelector('.status-indicator');
		const isHealthy = health.status === 'healthy';
		status.querySelector('.dot').style.background = isHealthy ? '#10b981' : '#ef4444';
		status.querySelector('span:last-child').textContent = isHealthy ? 'Healthy' : 'Offline';
	} catch (error) {
		const status = document.querySelector('.status-indicator');
		status.querySelector('.dot').style.background = '#ef4444';
		status.querySelector('span:last-child').textContent = 'Offline';
	}
}

// Auto Refresh
function startAutoRefresh() {
	setInterval(() => {
		if (currentPage === 'dashboard') {
			loadDashboard();
		}
		checkHealth();
	}, 30000); // Every 30 seconds
}

// Utilities
function calculateSuccessRate(data) {
	const total = data.workflow_summary?.total || 1;
	const success = data.workflow_summary?.success || 0;
	return Math.round((success / total) * 100) || 0;
}

function formatDuration(ms) {
	if (!ms) return '—';
	const seconds = Math.floor(ms / 1000);
	if (seconds < 60) return `${seconds}s`;
	const minutes = Math.floor(seconds / 60);
	if (minutes < 60) return `${minutes}m`;
	const hours = Math.floor(minutes / 60);
	return `${hours}h`;
}

function showSuccess(message) {
	showToast(message, 'success');
}

function showError(message) {
	showToast(message, 'error');
}

function showToast(message, type) {
	// Create toast element
	const toast = document.createElement('div');
	toast.style.cssText = `
		position: fixed;
		bottom: 1rem;
		right: 1rem;
		padding: 1rem 1.5rem;
		background: ${type === 'success' ? '#10b981' : '#ef4444'};
		color: white;
		border-radius: 0.5rem;
		z-index: 9999;
		animation: slideIn 0.3s ease;
	`;
	toast.textContent = message;
	document.body.appendChild(toast);

	setTimeout(() => {
		toast.style.animation = 'slideOut 0.3s ease';
		setTimeout(() => toast.remove(), 300);
	}, 3000);
}

// Add animations
const style = document.createElement('style');
style.textContent = `
	@keyframes slideIn {
		from { transform: translateX(400px); opacity: 0; }
		to { transform: translateX(0); opacity: 1; }
	}
	@keyframes slideOut {
		from { transform: translateX(0); opacity: 1; }
		to { transform: translateX(400px); opacity: 0; }
	}
`;
document.head.appendChild(style);

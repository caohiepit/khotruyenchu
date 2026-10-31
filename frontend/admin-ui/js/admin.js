/**
 * COMICS ADMIN PORTAL - CLIENT-SIDE CONTROLLER
 * Real-time monitoring, database explorer, Two-Phase Queue Crawler, and Universal Source Manager
 */

const API_BASE = (window.location.origin.includes('localhost:8000') || window.location.origin.includes('127.0.0.1:8000'))
  ? `${window.location.origin}/api`
  : 'http://localhost:8000/api';

// State
let appState = {
  activeTab: 'overview',
  searchQuery: '',
  filterStatus: '',
  filterQueueStatus: '',
  currentPage: 1,
  pageSize: 20,
  pollTimer: null
};

// DOM Elements
const elements = {
  navItems: document.querySelectorAll('.nav-item'),
  tabPanes: document.querySelectorAll('.tab-pane'),
  pageTitle: document.getElementById('page-title'),
  btnRefresh: document.getElementById('btn-refresh'),
  refreshIcon: document.getElementById('refresh-icon'),
  btnQuickCrawl: document.getElementById('btn-quick-crawl'),
  btnGoToQueue: document.getElementById('btn-go-to-queue'),
  
  // Overview Stats
  statTotalNovels: document.getElementById('stat-total-novels'),
  statTotalChapters: document.getElementById('stat-total-chapters'),
  statTotalWords: document.getElementById('stat-total-words'),
  statQueuePending: document.getElementById('stat-queue-pending'),
  statQueueHint: document.getElementById('stat-queue-hint'),
  navNovelsCount: document.getElementById('nav-novels-count'),
  navQueueCount: document.getElementById('nav-queue-count'),
  apiEndpointText: document.getElementById('api-endpoint-text'),

  // Banner
  workerPulseIndicator: document.getElementById('worker-pulse-indicator'),
  workerStatusTitle: document.getElementById('worker-status-title'),
  workerStatusDesc: document.getElementById('worker-status-desc'),

  // Queue Section
  queueCountPending: document.getElementById('queue-count-pending'),
  queueCountCrawling: document.getElementById('queue-count-crawling'),
  queueCountCompleted: document.getElementById('queue-count-completed'),
  queueCountFailed: document.getElementById('queue-count-failed'),
  workerBadge: document.getElementById('worker-badge'),
  formDiscover: document.getElementById('form-discover'),
  discoverUrl: document.getElementById('discover-url'),
  discoverPages: document.getElementById('discover-pages'),
  btnStartDiscover: document.getElementById('btn-start-discover'),
  workerLimit: document.getElementById('worker-limit'),
  workerDelay: document.getElementById('worker-delay'),
  btnStartWorker: document.getElementById('btn-start-worker'),
  btnStopWorker: document.getElementById('btn-stop-worker'),
  btnResetFailed: document.getElementById('btn-reset-failed'),
  filterQueueStatus: document.getElementById('filter-queue-status'),
  btnReloadQueue: document.getElementById('btn-reload-queue'),
  queueTableBody: document.getElementById('queue-table-body'),
  
  // Novels Table
  novelsTableBody: document.getElementById('novels-table-body'),
  novelSearchInput: document.getElementById('novel-search-input'),
  filterStatus: document.getElementById('filter-status'),
  btnReloadNovels: document.getElementById('btn-reload-novels'),
  tableEntriesInfo: document.getElementById('table-entries-info'),
  novelsPagination: document.getElementById('novels-pagination'),
  
  // Universal Crawler Form
  crawlerQuickForm: document.getElementById('crawler-quick-form'),
  crawlUrlInput: document.getElementById('crawl-url'),
  crawlLimitInput: document.getElementById('crawl-limit'),
  crawlPagesInput: document.getElementById('crawl-pages'),
  btnSubmitCrawl: document.getElementById('btn-submit-crawl'),
  crawlJobStatusBox: document.getElementById('crawl-job-status'),
  jobBadgeStatus: document.getElementById('job-badge-status'),
  jobMsgText: document.getElementById('job-msg-text'),

  // Sources Management
  sourcesTableBody: document.getElementById('sources-table-body'),
  btnRefreshSources: document.getElementById('btn-refresh-sources'),

  // Detail Modal
  novelDetailModal: document.getElementById('novel-detail-modal'),
  btnCloseModal: document.getElementById('btn-close-modal'),
  modalNovelTitle: document.getElementById('modal-novel-title'),
  modalNovelMeta: document.getElementById('modal-novel-meta'),
  modalNovelCover: document.getElementById('modal-novel-cover'),
  modalNovelStatus: document.getElementById('modal-novel-status'),
  modalNovelChaptersCount: document.getElementById('modal-novel-chapters-count'),
  modalNovelSource: document.getElementById('modal-novel-source'),
  modalNovelDesc: document.getElementById('modal-novel-desc'),
  modalActualChaptersBadge: document.getElementById('modal-actual-chapters-badge'),
  modalChaptersList: document.getElementById('modal-chapters-list'),

  // Reader Modal
  readerModal: document.getElementById('reader-modal'),
  btnCloseReader: document.getElementById('btn-close-reader'),
  readerChapterTitle: document.getElementById('reader-chapter-title'),
  readerChapterMeta: document.getElementById('reader-chapter-meta'),
  readerContentText: document.getElementById('reader-content-text')
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
  if (elements.apiEndpointText) {
    elements.apiEndpointText.textContent = API_BASE;
  }
  initTabs();
  initEventListeners();
  loadAllData();
  loadAffiliates();

  // Auto-refresh stats and queue status every 2 seconds
  setInterval(() => {
    loadStats();
    if (appState.activeTab === 'queue') {
      loadQueue();
    } else if (appState.activeTab === 'affiliate') {
      loadAffiliates();
    }
  }, 2000);
});

// Tab Switching
function initTabs() {
  elements.navItems.forEach(item => {
    item.addEventListener('click', (e) => {
      e.preventDefault();
      const tabId = item.getAttribute('data-tab');
      switchTab(tabId);
    });
  });
}

function switchTab(tabId) {
  appState.activeTab = tabId;
  elements.navItems.forEach(n => n.classList.toggle('active', n.getAttribute('data-tab') === tabId));
  elements.tabPanes.forEach(pane => {
    pane.classList.toggle('active', pane.id === `pane-${tabId}`);
  });

  const titles = {
    overview: 'Dashboard Tổng Quan',
    queue: 'Hàng Đợi Cào Truyện (Crawl Queue)',
    novels: 'Kho Dữ Liệu Truyện Trong Database',
    sources: 'Quản Lý Nguồn Web & Lập Lịch Quét Hàng Ngày',
    database: 'Hạ Tầng & Sơ Đồ Cơ Sở Dữ Liệu'
  };
  elements.pageTitle.textContent = titles[tabId] || 'Dashboard Tổng Quan';

  if (tabId === 'queue') {
    loadQueue();
  } else if (tabId === 'novels') {
    loadNovels();
  } else if (tabId === 'sources') {
    loadSources();
  }
}

// Event Listeners
function initEventListeners() {
  elements.btnRefresh.addEventListener('click', () => {
    elements.refreshIcon.classList.add('fa-spin');
    loadAllData().finally(() => {
      setTimeout(() => elements.refreshIcon.classList.remove('fa-spin'), 600);
    });
  });

  if (elements.btnQuickCrawl) {
    elements.btnQuickCrawl.addEventListener('click', () => switchTab('queue'));
  }
  if (elements.btnGoToQueue) {
    elements.btnGoToQueue.addEventListener('click', () => switchTab('queue'));
  }

  // Queue Events
  if (elements.formDiscover) {
    elements.formDiscover.addEventListener('submit', (e) => {
      e.preventDefault();
      triggerCatalogDiscovery();
    });
  }

  if (elements.btnStartWorker) {
    elements.btnStartWorker.addEventListener('click', toggleQueueWorker);
  }
  if (elements.btnStopWorker) {
    elements.btnStopWorker.addEventListener('click', toggleQueueWorker);
  }
  if (elements.btnResetFailed) {
    elements.btnResetFailed.addEventListener('click', resetFailedQueue);
  }
  if (elements.btnReloadQueue) {
    elements.btnReloadQueue.addEventListener('click', loadQueue);
  }
  if (elements.filterQueueStatus) {
    elements.filterQueueStatus.addEventListener('change', (e) => {
      appState.filterQueueStatus = e.target.value;
      loadQueue();
    });
  }

  // Search Debounce
  let searchTimeout;
  if (elements.novelSearchInput) {
    elements.novelSearchInput.addEventListener('input', (e) => {
      clearTimeout(searchTimeout);
      searchTimeout = setTimeout(() => {
        appState.searchQuery = e.target.value.trim();
        appState.currentPage = 1;
        loadNovels();
      }, 300);
    });
  }

  if (elements.filterStatus) {
    elements.filterStatus.addEventListener('change', (e) => {
      appState.filterStatus = e.target.value;
      appState.currentPage = 1;
      loadNovels();
    });
  }

  if (elements.btnReloadNovels) {
    elements.btnReloadNovels.addEventListener('click', loadNovels);
  }

  // Sources Refresh
  if (elements.btnRefreshSources) {
    elements.btnRefreshSources.addEventListener('click', loadSources);
  }

  // Universal Crawler Form Submit (Paste ANY URL -> Auto Queue & Auto Crawl!)
  if (elements.crawlerQuickForm) {
    elements.crawlerQuickForm.addEventListener('submit', (e) => {
      e.preventDefault();
      triggerUniversalCrawl();
    });
  }

  // Modal Closers
  if (elements.btnCloseModal) {
    elements.btnCloseModal.addEventListener('click', () => {
      elements.novelDetailModal.classList.remove('open');
    });
  }

  if (elements.btnCloseReader) {
    elements.btnCloseReader.addEventListener('click', () => {
      elements.readerModal.classList.remove('open');
    });
  }

  window.addEventListener('click', (e) => {
    if (e.target === elements.novelDetailModal) elements.novelDetailModal.classList.remove('open');
    if (e.target === elements.readerModal) elements.readerModal.classList.remove('open');
  });
}

// Data Loaders
async function loadAllData() {
  await Promise.allSettled([
    loadStats(),
    loadQueue(),
    loadNovels(),
    loadSources()
  ]);
}

// Fetch Overview Stats
async function loadStats() {
  try {
    const res = await fetch(`${API_BASE}/admin/stats`);
    if (!res.ok) throw new Error('API Error');
    const data = await res.json();

    elements.statTotalNovels.textContent = (data.total_novels || 0).toLocaleString();
    elements.statTotalChapters.textContent = (data.total_chapters || 0).toLocaleString();
    elements.statTotalWords.textContent = data.total_words ? data.total_words.toLocaleString() : '0';
    elements.statQueuePending.textContent = (data.queue_pending || 0).toLocaleString();
    elements.statQueueHint.textContent = `${data.queue_completed || 0} đã xong &bull; ${data.queue_crawling || 0} đang cào`;
    
    elements.navNovelsCount.textContent = data.total_novels || 0;
    elements.navQueueCount.textContent = (data.queue_pending || 0);

    // Update banner
    if (data.worker_running) {
      elements.workerPulseIndicator.className = 'worker-pulse active';
      elements.workerStatusTitle.textContent = 'Tiến trình Hàng Đợi (Queue Worker): ĐANG CHẠY LIÊN TỤC';
      elements.workerStatusDesc.textContent = `Worker nền đang tự động xử lý các truyện trong hàng đợi và cào ngầm vào PostgreSQL...`;
      if (elements.workerBadge) {
        elements.workerBadge.className = 'badge-tag cyan';
        elements.workerBadge.textContent = 'DAEMON RUNNING';
      }
    } else {
      elements.workerPulseIndicator.className = 'worker-pulse';
      elements.workerStatusTitle.textContent = 'Tiến trình Hàng Đợi (Queue Worker): TẠM DỪNG';
      elements.workerStatusDesc.textContent = `Có ${data.queue_pending || 0} truyện đang chờ cào. Bấm "Bắt đầu cào hàng đợi" để kích hoạt lại.`;
      if (elements.workerBadge) {
        elements.workerBadge.className = 'badge-tag amber';
        elements.workerBadge.textContent = 'PAUSED';
      }
    }
  } catch (err) {
    console.warn('Error fetching stats:', err);
  }
}

// Fetch Queue Data
async function loadQueue() {
  try {
    let url = `${API_BASE}/admin/crawler/queue`;
    if (appState.filterQueueStatus) {
      url += `?status=${appState.filterQueueStatus}`;
    }

    const res = await fetch(url);
    if (!res.ok) return;
    const data = await res.json();
    const { counts, items, current_job } = data;

    elements.queueCountPending.textContent = counts.pending;
    elements.queueCountCrawling.textContent = counts.crawling;
    elements.queueCountCompleted.textContent = counts.completed;
    elements.queueCountFailed.textContent = counts.failed;

    // Realtime Worker Telemetry Monitor
    const live = data.live_status;
    const liveTitle = document.getElementById('live-current-novel-title');
    const liveChap = document.getElementById('live-current-chapter');
    const liveSource = document.getElementById('live-current-source');
    const liveLogs = document.getElementById('live-log-console');
    const liveBadge = document.getElementById('live-worker-status-badge');

    if (live && live.is_running && live.current_job) {
      if (liveBadge) {
        liveBadge.className = 'badge-tag green';
        liveBadge.innerHTML = `<i class="fa-solid fa-spinner fa-spin"></i> Đang Cào Ngầm`;
      }
      if (liveTitle) liveTitle.textContent = live.current_job.title || 'Đang bóc tách...';
      if (liveChap) liveChap.textContent = `Chương ${live.current_chapter || 0}: ${live.current_chapter_title || ''}`;
      if (liveSource) liveSource.innerHTML = `<i class="fa-solid fa-globe"></i> ${escapeHtml(live.current_job.source_domain || 'Web')}`;
      if (liveLogs && live.recent_logs && live.recent_logs.length > 0) {
        liveLogs.innerHTML = live.recent_logs.map(l => `<div style="margin-bottom: 2px;">${escapeHtml(l)}</div>`).join('');
        liveLogs.scrollTop = liveLogs.scrollHeight;
      }
    } else if (live && !live.is_running) {
      if (liveBadge) {
        liveBadge.className = 'badge-tag amber';
        liveBadge.innerHTML = `<i class="fa-solid fa-pause"></i> Tạm Dừng`;
      }
      if (liveTitle) liveTitle.textContent = 'Worker đang tạm dừng.';
    } else {
      if (liveBadge) {
        liveBadge.className = 'badge-tag green';
        liveBadge.innerHTML = `<i class="fa-solid fa-check"></i> Đang Chờ Việc`;
      }
      if (liveTitle) liveTitle.textContent = 'Hàng đợi đang rảnh hoặc đang quét nguồn mới...';
    }

    // Show current job in banner if running
    if (current_job && elements.crawlJobStatusBox) {
      elements.crawlJobStatusBox.classList.remove('hidden');
      elements.jobBadgeStatus.className = 'job-badge running';
      elements.jobBadgeStatus.textContent = 'Crawling';
      elements.jobMsgText.textContent = `Đang cào: 【${current_job.title}】 (${current_job.crawled_chapters || 0} chương)`;
    }

    if (!items || items.length === 0) {
      elements.queueTableBody.innerHTML = `
        <tr>
          <td colspan="7" style="text-align: center; color: var(--text-dim); padding: 32px;">
            <i class="fa-solid fa-list-check" style="font-size: 1.8rem; margin-bottom: 8px; display: block;"></i>
            Hàng đợi trống. Hãy dán link truyện hoặc link danh mục ở trên để nạp tự động!
          </td>
        </tr>
      `;
      return;
    }

    elements.queueTableBody.innerHTML = items.map(q => {
      const coverImg = q.cover_url ? `${API_BASE}/proxy-cover?url=${encodeURIComponent(q.cover_url)}` : 'https://images.unsplash.com/photo-1544716278-ca5e3f4abd8c?w=100&auto=format&fit=crop&q=80';
      const updatedTime = new Date(q.updated_at).toLocaleTimeString();
      return `
        <tr>
          <td>
            <img src="${coverImg}" alt="${escapeHtml(q.title)}" class="novel-thumb" style="width: 42px; height: 58px;" onerror="this.src='https://via.placeholder.com/50x70?text=No+Cover'">
          </td>
          <td class="novel-title-cell">
            <strong>${escapeHtml(q.title)}</strong>
            <span class="novel-slug">${escapeHtml(q.slug)}</span>
            ${q.error_message ? `<div style="color: #EF4444; font-size: 0.72rem; max-width: 300px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">${escapeHtml(q.error_message)}</div>` : ''}
          </td>
          <td>
            <small style="color: var(--text-muted);">${escapeHtml(q.author_name || 'Khuyết danh')}</small>
          </td>
          <td style="text-align: center;">
            <span class="status-badge ${q.status}">
              ${q.status === 'crawling' ? '<i class="fa-solid fa-spinner fa-spin"></i>' : ''}
              ${escapeHtml(q.status)}
            </span>
          </td>
          <td style="text-align: center;">
            <span class="chapter-count-badge" style="font-size: 0.8rem; padding: 2px 8px;">
              ${q.crawled_chapters} chương
            </span>
          </td>
          <td>
            <small style="color: var(--text-dim); font-family: var(--font-mono);">${updatedTime}</small>
          </td>
          <td style="text-align: right;">
            <a href="${q.source_url}" target="_blank" class="btn btn-outline btn-sm" title="Xem trên web nguồn">
              <i class="fa-solid fa-arrow-up-right-from-square"></i>
            </a>
          </td>
        </tr>
      `;
    }).join('');
  } catch (err) {
    console.warn('Error loading queue:', err);
  }
}

// 🚀 Universal Auto Ingestion Trigger: Paste ANY URL -> Enqueue -> Auto Worker Crawl!
async function triggerUniversalCrawl() {
  const url = elements.crawlUrlInput.value.trim();
  const limit = parseInt(elements.crawlLimitInput.value) || 0;
  const pages = elements.crawlPagesInput ? (parseInt(elements.crawlPagesInput.value) || 2) : 2;

  if (!url) {
    alert('Vui lòng dán link truyện hoặc danh mục');
    return;
  }

  elements.btnSubmitCrawl.disabled = true;
  elements.crawlJobStatusBox.classList.remove('hidden');
  elements.jobBadgeStatus.className = 'job-badge';
  elements.jobBadgeStatus.textContent = 'Dispatched';
  elements.jobMsgText.textContent = `Đang phân tích link và gửi tới hệ thống worker...`;

  try {
    const res = await fetch(`${API_BASE}/admin/crawler/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url, limit, pages })
    });

    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Lỗi gửi yêu cầu');

    elements.jobBadgeStatus.className = 'job-badge completed';
    elements.jobBadgeStatus.textContent = 'Auto Running';
    elements.jobMsgText.textContent = data.message;
    
    // Auto refresh queue & stats
    setTimeout(() => {
      loadStats();
      loadQueue();
      elements.btnSubmitCrawl.disabled = false;
    }, 1200);
  } catch (err) {
    elements.jobBadgeStatus.className = 'job-badge failed';
    elements.jobBadgeStatus.textContent = 'Error';
    elements.jobMsgText.textContent = err.message;
    elements.btnSubmitCrawl.disabled = false;
  }
}

// Catalog Discovery Trigger from Queue Tab
async function triggerCatalogDiscovery() {
  const url = elements.discoverUrl.value.trim();
  const pages = parseInt(elements.discoverPages.value) || 1;

  if (!url) return;

  elements.btnStartDiscover.disabled = true;
  elements.btnStartDiscover.innerHTML = '<i class="fa-solid fa-spinner fa-spin"></i> Đang quét...';

  try {
    const res = await fetch(`${API_BASE}/admin/crawler/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url, pages })
    });

    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Lỗi gửi lệnh quét');
    
    setTimeout(() => {
      loadQueue();
      loadStats();
      elements.btnStartDiscover.disabled = false;
      elements.btnStartDiscover.innerHTML = '<i class="fa-solid fa-magnifying-glass"></i> Quét & Nạp Vào Queue';
    }, 2000);
  } catch (err) {
    alert('Lỗi: ' + err.message);
    elements.btnStartDiscover.disabled = false;
    elements.btnStartDiscover.innerHTML = '<i class="fa-solid fa-magnifying-glass"></i> Quét & Nạp Vào Queue';
  }
}

// Toggle Worker Pause/Resume
async function toggleQueueWorker() {
  try {
    const res = await fetch(`${API_BASE}/admin/crawler/queue/toggle`, { method: 'POST' });
    const data = await res.json();
    loadStats();
    loadQueue();
  } catch (err) {
    console.warn(err);
  }
}

// Reset Failed Items
async function resetFailedQueue() {
  try {
    await fetch(`${API_BASE}/admin/crawler/queue/reset-failed`, { method: 'POST' });
    loadQueue();
    loadStats();
  } catch (err) {
    console.warn(err);
  }
}

// Fetch Novels Table Data
async function loadNovels() {
  if (!elements.novelsTableBody) return;
  elements.novelsTableBody.innerHTML = `
    <tr>
      <td colspan="7" class="loading-row">
        <i class="fa-solid fa-circle-notch fa-spin"></i> Đang truy vấn từ PostgreSQL 16...
      </td>
    </tr>
  `;

  try {
    let url = `${API_BASE}/admin/novels?page=${appState.currentPage}&limit=${appState.pageSize}`;
    if (appState.searchQuery) {
      url += `&q=${encodeURIComponent(appState.searchQuery)}`;
    }

    const res = await fetch(url);
    if (!res.ok) throw new Error('Failed to load novels');
    const data = await res.json();

    renderNovelsTable(data);
    renderNovelsPagination(data.total || 0);
  } catch (err) {
    elements.novelsTableBody.innerHTML = `
      <tr>
        <td colspan="7" style="text-align: center; color: #EF4444; padding: 24px;">
          <i class="fa-solid fa-triangle-exclamation"></i> Không thể kết nối tới Backend API.
        </td>
      </tr>
    `;
  }
}

function renderNovelsTable(data) {
  const { data: novels, total } = data;
  if (elements.tableEntriesInfo) {
    elements.tableEntriesInfo.textContent = `Tổng cộng ${total || 0} bộ truyện trong database`;
  }

  if (!novels || novels.length === 0) {
    elements.novelsTableBody.innerHTML = `
      <tr>
        <td colspan="7" style="text-align: center; color: var(--text-dim); padding: 36px;">
          <i class="fa-solid fa-inbox" style="font-size: 2rem; margin-bottom: 8px; display: block;"></i>
          Chưa tìm thấy bộ truyện nào. Hãy dùng form cào để thêm dữ liệu vào DB!
        </td>
      </tr>
    `;
    return;
  }

  elements.novelsTableBody.innerHTML = novels.map(n => {
    const coverImg = n.cover_url ? `${API_BASE}/proxy-cover?url=${encodeURIComponent(n.cover_url)}` : 'https://images.unsplash.com/photo-1544716278-ca5e3f4abd8c?w=120&auto=format&fit=crop&q=80';
    const statusClass = (n.status || '').toLowerCase().includes('full') ? 'full' : 'ongoing';
    const statusText = n.status || 'Đang ra';
    const genresBadges = (n.genres || []).slice(0, 3).map(g => `<span class="genre-badge">${escapeHtml(g)}</span>`).join('');

    return `
      <tr>
        <td>
          <img src="${coverImg}" alt="${escapeHtml(n.title)}" class="novel-thumb" onerror="this.src='https://via.placeholder.com/60x85?text=No+Cover'">
        </td>
        <td class="novel-title-cell">
          <strong>${escapeHtml(n.title)}</strong>
          <span class="novel-slug">${escapeHtml(n.slug)}</span>
        </td>
        <td>
          <i class="fa-solid fa-pen-nib" style="color: var(--text-dim); font-size: 0.75rem;"></i>
          ${escapeHtml(n.author_name || 'Khuyết danh')}
        </td>
        <td>
          <div class="genre-tag-list">
            ${genresBadges || '<span class="text-dim">Chưa phân loại</span>'}
          </div>
        </td>
        <td style="text-align: center;">
          <span class="chapter-count-badge" title="Số chương thực tế lưu trong bảng chapters">
            ${n.actual_chapters || 0} chương
          </span>
        </td>
        <td style="text-align: center;">
          <span class="status-pill ${statusClass}">
            <i class="fa-solid fa-circle" style="font-size: 0.5rem;"></i>
            ${escapeHtml(statusText)}
          </span>
        </td>
        <td>
          <div class="actions-cell">
            <button class="btn btn-secondary btn-sm" onclick="viewNovelDetail('${n.id}')">
              <i class="fa-solid fa-list-ol"></i> Mục Lục
            </button>
            <a href="${n.source_url}" target="_blank" class="btn btn-outline btn-sm" title="Xem trên web nguồn">
              <i class="fa-solid fa-arrow-up-right-from-square"></i>
            </a>
          </div>
        </td>
      </tr>
    `;
  }).join('');
}

// 🌐 Load Web Sources List
async function loadSources() {
  if (!elements.sourcesTableBody) return;
  try {
    const res = await fetch(`${API_BASE}/admin/sources`);
    if (!res.ok) return;
    const sources = await res.json();

    if (!sources || sources.length === 0) {
      elements.sourcesTableBody.innerHTML = `<tr><td colspan="7" class="empty-state">Chưa có nguồn web nào được đăng ký.</td></tr>`;
      return;
    }

    elements.sourcesTableBody.innerHTML = sources.map(s => {
      const lastScanned = s.last_scanned_at ? new Date(s.last_scanned_at).toLocaleString() : 'Chưa quét';
      const statusBadge = s.is_active 
        ? `<span class="badge-tag green">Đang Hoạt Động</span>` 
        : `<span class="badge-tag red">Tạm Dừng</span>`;

      return `
        <tr>
          <td>
            <strong>${escapeHtml(s.name)}</strong>
          </td>
          <td>
            <a href="${s.base_url}" target="_blank" style="color: var(--cyan-primary); text-decoration: none;">
              ${escapeHtml(s.domain)} <i class="fa-solid fa-arrow-up-right-from-square" style="font-size: 0.75rem;"></i>
            </a>
          </td>
          <td style="font-size: 0.8rem; color: var(--text-muted); max-width: 260px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
            ${escapeHtml(s.catalog_url)}
          </td>
          <td style="text-align: center;">
            <span class="badge-tag cyan">Mỗi ${s.scan_interval_hours || 24}h</span>
          </td>
          <td>
            <small style="color: var(--text-dim);">${lastScanned}</small>
          </td>
          <td style="text-align: center;">
            ${statusBadge}
          </td>
          <td style="text-align: right;">
            <div style="display: flex; gap: 6px; justify-content: flex-end;">
              <button class="btn btn-primary btn-sm" onclick="scanSourceNow('${s.id}')" title="Quét danh mục ngay">
                <i class="fa-solid fa-bolt"></i> Quét Ngay
              </button>
              <button class="btn btn-secondary btn-sm" onclick="toggleSource('${s.id}', ${!s.is_active})" title="Bật / Tắt nguồn">
                ${s.is_active ? '<i class="fa-solid fa-pause"></i> Tắt' : '<i class="fa-solid fa-play"></i> Bật'}
              </button>
            </div>
          </td>
        </tr>
      `;
    }).join('');
  } catch (err) {
    console.warn('Error loading sources:', err);
  }
}

// Global actions for sources
window.toggleSource = async function(id, isActive) {
  try {
    const res = await fetch(`${API_BASE}/admin/sources/toggle`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, is_active: isActive })
    });
    if (res.ok) {
      loadSources();
    }
  } catch (e) {
    alert('Lỗi cập nhật nguồn: ' + e.message);
  }
};

window.scanSourceNow = async function(id) {
  try {
    const res = await fetch(`${API_BASE}/admin/sources/scan`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id })
    });
    const data = await res.json();
    alert(data.message || 'Đã phát lệnh quét nguồn thành công!');
    setTimeout(() => {
      loadQueue();
      loadStats();
      loadSources();
    }, 1500);
  } catch (e) {
    alert('Lỗi: ' + e.message);
  }
};

// View Novel Detail Modal
window.viewNovelDetail = async function(novelId) {
  elements.novelDetailModal.classList.add('open');
  elements.modalNovelTitle.textContent = 'Đang tải thông tin...';
  elements.modalChaptersList.innerHTML = `<div style="text-align: center; padding: 20px;"><i class="fa-solid fa-circle-notch fa-spin"></i> Đang tải danh sách chương...</div>`;

  try {
    const res = await fetch(`${API_BASE}/admin/novels/${novelId}`);
    if (!res.ok) throw new Error('Failed to load novel');
    const data = await res.json();
    const { novel, chapters } = data;

    elements.modalNovelTitle.textContent = novel.title;
    elements.modalNovelMeta.textContent = `Tác giả: ${novel.author_name} | Thể loại: ${novel.genres.join(', ') || 'N/A'}`;
    elements.modalNovelCover.src = novel.cover_url || 'https://via.placeholder.com/260x350?text=Cover';
    elements.modalNovelStatus.textContent = novel.status;
    elements.modalNovelChaptersCount.textContent = `${chapters.length} chương`;
    elements.modalActualChaptersBadge.textContent = `${chapters.length} chương trong DB`;
    elements.modalNovelSource.href = novel.source_url || '#';
    elements.modalNovelDesc.textContent = novel.description || 'Không có mô tả.';

    if (!chapters || chapters.length === 0) {
      elements.modalChaptersList.innerHTML = `<p style="text-align: center; color: var(--text-dim); padding: 20px;">Chưa có chương nào được cào cho bộ truyện này.</p>`;
      return;
    }

    elements.modalChaptersList.innerHTML = chapters.map(c => `
      <div class="chapter-row-item" onclick="readChapterContent('${c.id}')">
        <div class="chapter-row-title">
          <i class="fa-solid fa-file-lines text-cyan" style="margin-right: 6px;"></i>
          ${escapeHtml(c.title)}
        </div>
        <div class="chapter-row-words">
          ${c.word_count ? c.word_count.toLocaleString() : 0} từ &bull; <span style="color: var(--cyan-primary);">Đọc thử &rarr;</span>
        </div>
      </div>
    `).join('');
  } catch (err) {
    elements.modalChaptersList.innerHTML = `<p style="color: #EF4444; padding: 20px;">Lỗi tải chi tiết: ${err.message}</p>`;
  }
};

// Read Chapter Content Drawer
window.readChapterContent = async function(chapterId) {
  elements.readerModal.classList.add('open');
  elements.readerChapterTitle.textContent = 'Đang tải nội dung...';
  elements.readerContentText.innerHTML = '<i class="fa-solid fa-circle-notch fa-spin"></i> Đang giải nén LZ4 từ PostgreSQL...';

  try {
    const res = await fetch(`${API_BASE}/admin/chapters/${chapterId}`);
    if (!res.ok) throw new Error('Failed to load chapter content');
    const chap = await res.json();

    elements.readerChapterTitle.textContent = chap.title;
    elements.readerChapterMeta.textContent = `${chap.word_count ? chap.word_count.toLocaleString() : 0} từ | Trích xuất từ bảng chapter_contents (Nén LZ4 TOAST)`;
    elements.readerContentText.textContent = chap.content;
  } catch (err) {
    elements.readerContentText.innerHTML = `<span style="color: #EF4444;">Không tìm thấy nội dung chương: ${err.message}</span>`;
  }
};

// Helper: Escape HTML to avoid XSS
function escapeHtml(str) {
  if (!str) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

// =====================================================================
// SHOPEE AFFILIATE MANAGEMENT MODULE
// =====================================================================
let affiliatesCache = [];

async function loadAffiliates() {
  const tbody = document.getElementById('affiliate-table-body');
  if (!tbody) return;

  try {
    const res = await fetch(`${API_BASE}/admin/affiliates`);
    if (!res.ok) throw new Error('Không thể tải danh sách affiliate');
    affiliatesCache = await res.json();

    const totalEl = document.getElementById('affiliate-count-total');
    const activeEl = document.getElementById('affiliate-count-active');
    const clicksEl = document.getElementById('affiliate-count-clicks');
    const navPill = document.getElementById('nav-affiliate-count');

    const total = affiliatesCache.length;
    const active = affiliatesCache.filter(a => a.is_active).length;
    const clicks = affiliatesCache.reduce((sum, a) => sum + (a.clicks_count || 0), 0);

    if (totalEl) totalEl.textContent = total;
    if (activeEl) activeEl.textContent = active;
    if (clicksEl) clicksEl.textContent = clicks;
    if (navPill) navPill.textContent = active;

    if (affiliatesCache.length === 0) {
      tbody.innerHTML = `
        <tr>
          <td colspan="8" style="text-align: center; color: var(--text-dim); padding: 32px;">
            <i class="fa-solid fa-cart-shopping" style="font-size: 1.8rem; margin-bottom: 8px; display: block;"></i>
            Chưa có liên kết Shopee Affiliate nào. Bấm <strong>"Thêm Link Shopee Mới"</strong> để tạo!
          </td>
        </tr>
      `;
      return;
    }

    tbody.innerHTML = affiliatesCache.map(a => {
      const positionName = a.position === 'ticker' ? 'Thanh chạy Ticker' : (a.position === 'reader_footer' ? 'Chân trang Đọc' : 'Thanh bên');
      return `
        <tr>
          <td>
            <img src="${escapeHtml(a.image_url)}" alt="${escapeHtml(a.title)}" style="width: 44px; height: 44px; object-fit: cover; border-radius: 6px; border: 1px solid rgba(255,255,255,0.1);" onerror="this.src='https://cf.shopee.vn/file/vn-11134207-7r98o-ls5h19875e5f0d'">
          </td>
          <td>
            <div style="font-weight: 600; color: #fff;">${escapeHtml(a.title)}</div>
            ${a.tag ? `<span class="badge-tag cyan" style="margin-top: 4px; display: inline-block;">${escapeHtml(a.tag)}</span>` : ''}
          </td>
          <td>
            <a href="${escapeHtml(a.affiliate_url)}" target="_blank" rel="noopener noreferrer" style="color: #38bdf8; font-size: 0.85rem; display: flex; align-items: center; gap: 4px;">
              <span>Mở Shopee</span> <i class="fa-solid fa-arrow-up-right-from-square" style="font-size: 0.72rem;"></i>
            </a>
          </td>
          <td>
            <span style="color: #f59e0b; font-weight: 700;">${escapeHtml(a.price_discount || '--')}</span>
            ${a.price_original ? `<span style="font-size: 0.75rem; color: #64748b; text-decoration: line-through; margin-left: 4px;">${escapeHtml(a.price_original)}</span>` : ''}
          </td>
          <td>
            <span class="badge-pill" style="font-size: 0.78rem;">${positionName}</span>
          </td>
          <td style="text-align: center; font-weight: 700; color: #10b981;">
            ${a.clicks_count || 0}
          </td>
          <td style="text-align: center;">
            <button class="btn btn-sm ${a.is_active ? 'btn-primary' : 'btn-secondary'}" onclick="toggleAffiliate('${a.id}', ${!a.is_active})" style="padding: 4px 10px; font-size: 0.75rem;">
              ${a.is_active ? '<i class="fa-solid fa-check"></i> Đang Bật' : '<i class="fa-solid fa-pause"></i> Tắt'}
            </button>
          </td>
          <td style="text-align: right;">
            <button class="btn btn-secondary btn-sm" onclick="editAffiliate('${a.id}')" title="Sửa" style="padding: 4px 8px; margin-right: 4px;">
              <i class="fa-solid fa-pen-to-square"></i>
            </button>
            <button class="btn btn-outline btn-sm" onclick="deleteAffiliate('${a.id}')" title="Xóa" style="padding: 4px 8px; color: #ef4444; border-color: rgba(239,68,68,0.3);">
              <i class="fa-solid fa-trash"></i>
            </button>
          </td>
        </tr>
      `;
    }).join('');
  } catch (err) {
    if (tbody) {
      tbody.innerHTML = `<tr><td colspan="8" style="color: #ef4444; padding: 20px;">Lỗi tải dữ liệu Shopee: ${err.message}</td></tr>`;
    }
  }
}

// Modal open/close
const affiliateModal = document.getElementById('affiliate-modal');
const formAffiliate = document.getElementById('form-affiliate');
const btnOpenAddAffiliate = document.getElementById('btn-open-add-affiliate');
const btnCloseAffiliateModal = document.getElementById('btn-close-affiliate-modal');
const btnCancelAffiliate = document.getElementById('btn-cancel-affiliate');
const btnReloadAffiliates = document.getElementById('btn-reload-affiliates');

if (btnOpenAddAffiliate) {
  btnOpenAddAffiliate.addEventListener('click', () => {
    formAffiliate.reset();
    document.getElementById('affiliate-id').value = '';
    document.getElementById('modal-affiliate-title').textContent = 'Thêm Link Shopee Affiliate';
    affiliateModal.classList.add('open');
  });
}

if (btnCloseAffiliateModal) {
  btnCloseAffiliateModal.addEventListener('click', () => affiliateModal.classList.remove('open'));
}
if (btnCancelAffiliate) {
  btnCancelAffiliate.addEventListener('click', () => affiliateModal.classList.remove('open'));
}
if (btnReloadAffiliates) {
  btnReloadAffiliates.addEventListener('click', loadAffiliates);
}

if (formAffiliate) {
  formAffiliate.addEventListener('submit', async (e) => {
    e.preventDefault();
    const id = document.getElementById('affiliate-id').value;
    const title = document.getElementById('affiliate-title-input').value.trim();
    const affiliate_url = document.getElementById('affiliate-url-input').value.trim();
    const image_url = document.getElementById('affiliate-image-input').value.trim();
    const price_discount = document.getElementById('affiliate-discount-input').value.trim();
    const price_original = document.getElementById('affiliate-original-input').value.trim();
    const tag = document.getElementById('affiliate-tag-input').value.trim();
    const position = document.getElementById('affiliate-position-select').value;

    const payload = {
      title,
      affiliate_url,
      image_url: image_url || 'https://cf.shopee.vn/file/vn-11134207-7r98o-ls5h19875e5f0d',
      price_discount,
      price_original,
      tag,
      position,
      is_active: true
    };

    try {
      const url = id ? `${API_BASE}/admin/affiliates/${id}` : `${API_BASE}/admin/affiliates`;
      const method = id ? 'PUT' : 'POST';
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error || 'Lỗi lưu affiliate');
      }
      affiliateModal.classList.remove('open');
      loadAffiliates();
    } catch (err) {
      alert(`Lỗi: ${err.message}`);
    }
  });
}

window.toggleAffiliate = async function(id, newState) {
  try {
    const res = await fetch(`${API_BASE}/admin/affiliates/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ is_active: newState })
    });
    if (res.ok) loadAffiliates();
  } catch (err) {
    console.error('Error toggling affiliate:', err);
  }
};

window.editAffiliate = function(id) {
  const item = affiliatesCache.find(a => a.id === id);
  if (!item) return;

  document.getElementById('affiliate-id').value = item.id;
  document.getElementById('affiliate-title-input').value = item.title;
  document.getElementById('affiliate-url-input').value = item.affiliate_url;
  document.getElementById('affiliate-image-input').value = item.image_url;
  document.getElementById('affiliate-discount-input').value = item.price_discount || '';
  document.getElementById('affiliate-original-input').value = item.price_original || '';
  document.getElementById('affiliate-tag-input').value = item.tag || '';
  document.getElementById('affiliate-position-select').value = item.position || 'ticker';

  document.getElementById('modal-affiliate-title').textContent = 'Chỉnh Sửa Link Shopee Affiliate';
  affiliateModal.classList.add('open');
};

window.deleteAffiliate = async function(id) {
  if (!confirm('Bạn có chắc chắn muốn xóa liên kết Shopee này?')) return;
  try {
    const res = await fetch(`${API_BASE}/admin/affiliates/${id}`, { method: 'DELETE' });
    if (res.ok) loadAffiliates();
  } catch (err) {
    alert(`Lỗi xóa: ${err.message}`);
  }
};


function renderNovelsPagination(total) {
  if (!elements.novelsPagination) return;
  const totalPages = Math.ceil(total / appState.pageSize);
  if (totalPages <= 1) {
    elements.novelsPagination.innerHTML = '';
    return;
  }
  
  let html = '<div style="display:flex; align-items:center; gap:6px;">';
  html += `<button class="btn btn-outline btn-sm" ${appState.currentPage === 1 ? 'disabled style="opacity:0.4;cursor:not-allowed;"' : ''} onclick="goToNovelsPage(${appState.currentPage - 1})"><i class="fa-solid fa-chevron-left"></i></button>`;
  
  let pages = [];
  if (totalPages <= 7) {
    pages = Array.from({length: totalPages}, (_, i) => i + 1);
  } else {
    pages.push(1);
    let start = Math.max(2, appState.currentPage - 1);
    let end = Math.min(totalPages - 1, appState.currentPage + 1);
    if (appState.currentPage <= 3) { start = 2; end = 4; }
    else if (appState.currentPage >= totalPages - 2) { start = totalPages - 3; end = totalPages - 1; }
    if (start > 2) pages.push('...');
    for (let i = start; i <= end; i++) pages.push(i);
    if (end < totalPages - 1) pages.push('...');
    pages.push(totalPages);
  }

  pages.forEach(p => {
    if (p === '...') {
      html += '<span style="padding:0 4px; color:var(--text-muted); font-weight:700;">...</span>';
    } else {
      const active = p === appState.currentPage;
      html += `<button class="btn btn-sm ${active ? 'btn-primary' : 'btn-outline'}" onclick="goToNovelsPage(${p})">${p}</button>`;
    }
  });

  html += `<button class="btn btn-outline btn-sm" ${appState.currentPage === totalPages ? 'disabled style="opacity:0.4;cursor:not-allowed;"' : ''} onclick="goToNovelsPage(${appState.currentPage + 1})"><i class="fa-solid fa-chevron-right"></i></button>`;
  html += '</div>';
  elements.novelsPagination.innerHTML = html;
}

window.goToNovelsPage = function(p) {
  appState.currentPage = p;
  loadNovels();
};

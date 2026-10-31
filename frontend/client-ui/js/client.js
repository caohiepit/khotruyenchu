function getNovelRating(novel) {

  if (!novel) return '4.8';

  const raw = novel.rating_score || novel.rating;

  if (!raw || isNaN(Number(raw))) return '4.8';

  let num = Number(raw);

  if (num > 5) {

    num = num / 2;

  }

  return num.toFixed(1);

}





function renderPaginationComponent(containerEl, currentPage, totalPages, onPageChange) {

  if (!containerEl) return;

  if (!totalPages || totalPages <= 1) {

    containerEl.innerHTML = '';

    return;

  }



  // Determine window of page numbers

  let items = [];

  if (totalPages <= 7) {

    for (let i = 1; i <= totalPages; i++) items.push(i);

  } else {

    items.push(1);

    let start = Math.max(2, currentPage - 2);

    let end = Math.min(totalPages - 1, currentPage + 2);



    if (currentPage <= 4) {

      start = 2;

      end = 5;

    } else if (currentPage >= totalPages - 3) {

      start = totalPages - 4;

      end = totalPages - 1;

    }



    if (start > 2) items.push('...');

    for (let p = start; p <= end; p++) items.push(p);

    if (end < totalPages - 1) items.push('...');

    items.push(totalPages);

  }



  let html = '<div class="smart-pagination-bar">';



  // First & Prev buttons

  const isFirst = currentPage === 1;

  const isLast = currentPage === totalPages;



  html += `

    <button class="page-btn page-btn--nav" data-target-page="1" ${isFirst ? 'disabled' : ''} title="Trang đầu">

      <i class="fa-solid fa-angles-left"></i>

    </button>

    <button class="page-btn page-btn--nav" data-target-page="${currentPage - 1}" ${isFirst ? 'disabled' : ''} title="Trang trước">

      <i class="fa-solid fa-chevron-left"></i>

    </button>

  `;



  // Number items & ellipses

  items.forEach(it => {

    if (it === '...') {

      html += `<span class="page-ellipsis">...</span>`;

    } else {

      const isActive = it === currentPage;

      html += `<button class="page-btn ${isActive ? 'active' : ''}" data-target-page="${it}" ${isActive ? 'aria-current="page"' : ''}>${it}</button>`;

    }

  });



  // Next & Last buttons

  html += `

    <button class="page-btn page-btn--nav" data-target-page="${currentPage + 1}" ${isLast ? 'disabled' : ''} title="Trang kế tiếp">

      <i class="fa-solid fa-chevron-right"></i>

    </button>

    <button class="page-btn page-btn--nav" data-target-page="${totalPages}" ${isLast ? 'disabled' : ''} title="Trang cuối">

      <i class="fa-solid fa-angles-right"></i>

    </button>

  `;



  // Quick jump input if totalPages > 5

  if (totalPages > 5) {

    html += `

      <div class="page-jump-box">

        <span class="jump-label">Đến trang:</span>

        <input type="number" min="1" max="${totalPages}" value="${currentPage}" class="page-jump-input" aria-label="Nhập số trang">

        <span class="jump-total">/ ${totalPages}</span>

        <button class="page-btn-jump">Đi</button>

      </div>

    `;

  }



  html += '</div>';



  containerEl.innerHTML = html;



  // Bind click handlers on page buttons

  containerEl.querySelectorAll('button[data-target-page]').forEach(btn => {

    btn.addEventListener('click', (e) => {

      e.preventDefault();

      if (btn.disabled) return;

      const target = Number(btn.getAttribute('data-target-page'));

      if (target && target >= 1 && target <= totalPages && target !== currentPage) {

        onPageChange(target);

      }

    });

  });



  // Bind Quick Jump

  const jumpInput = containerEl.querySelector('.page-jump-input');

  const jumpBtn = containerEl.querySelector('.page-btn-jump');

  if (jumpInput && jumpBtn) {

    const doJump = () => {

      let val = parseInt(jumpInput.value, 10);

      if (isNaN(val) || val < 1) val = 1;

      if (val > totalPages) val = totalPages;

      if (val !== currentPage) {

        onPageChange(val);

      }

    };

    jumpBtn.addEventListener('click', (e) => {

      e.preventDefault();

      doJump();

    });

    jumpInput.addEventListener('keydown', (e) => {

      if (e.key === 'Enter') {

        e.preventDefault();

        doJump();

      }

    });

  }

}





function formatNovelDetailDescription(novelOrRaw, actualCount = 0) {

  let raw = '';

  let novel = null;

  if (novelOrRaw && typeof novelOrRaw === 'object') {

    novel = novelOrRaw;

    raw = novel.description || '';

  } else {

    raw = typeof novelOrRaw === 'string' ? novelOrRaw : '';

  }



  if (!raw && !novel) {

    return '<p>Nội dung bộ truyện đang được cập nhật, kính mời quý độc giả đón đọc...</p>';

  }



  // 1. Remove truyenfull / affiliate SEO spam strings and replace with newline

  let t = raw.replace(/(?:truyen\s*full\s*(?:live|vn)?|truyenfulllive|truyenfullvn|truyenfull)[,\s.;:-]*/gi, '\n');



  // 2. Insert newline before known metadata keys (Số chương, Dịch, Biên, Nguồn, Tác giả, Thể loại, Trạng thái)

  t = t.replace(/([^\n])\s*(Số\s+chương|Dịch\s*giả|Dịch|Biên\s*tập|Biên|Nguồn|Tác\s+giả|Thể\s+loại|Trạng\s+thái)\s*:/gi, '$1\n$2:');



  // 3. Remove bullets like • or ·

  t = t.replace(/[•·]\s*/g, '\n');



  // 4. Split sentences when multiple dots or commas with dots are immediately followed by capital letters

  t = t.replace(/(\_{2,}|\.{2,}|,\s*\.{2,})(?=[A-ZÀÁẢÃẠĂẮẰẲẴẶÂẤẦẨẪẬĐÈÉẺẼẸÊẾỀỂỄỆÌÍỈĨỊÒÓỎÕỌÔỐỒỔỖỘƠỚỜỞỠỢÙÚỦŨỤƯỨỪỬỮỰỲÝỶỸỴ])/g, '$1\n\n');



  // 5. If translator line is stuck to start of synopsis (e.g. 'Dịch giả: Thảo ĐànVề kết hôn...'), break it

  t = t.replace(/(dịch\s*(?:giả)?\s*:[^\n\r]+?)([A-ZÀÁẢÃẠĂẮẰẲẴẶÂẤẦẨẪẬĐÈÉẺẼẸÊẾỀỂỄỆÌÍỈĨỊÒÓỎÕỌÔỐỒỔỖỘƠỚỜỞỠỢÙÚỦŨỤƯỨỪỬỮỰỲÝỶỸỴ][a-zàáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđ]{2,}\s+[a-zàáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđ]+)/gi, '$1\n$2');



  const rawLines = t.split(/\r?\n/).map(l => l.trim()).filter(Boolean);

  const bodyLines = [];

  const extracted = {};



  const prefixDefs = [

    { key: 'dich', reg: /^(?:dịch\s*giả|dịch|nhóm\s*dịch)\s*:\s*(.*)$/i },

    { key: 'bien', reg: /^(?:biên\s*tập|biên)\s*:\s*(.*)$/i },

    { key: 'nguon', reg: /^(?:nguồn)\s*:\s*(.*)$/i },

    { key: 'author', reg: /^(?:tác\s*giả)\s*:\s*(.*)$/i },

    { key: 'genres', reg: /^(?:thể\s*loại)\s*:\s*(.*)$/i },

  ];



  rawLines.forEach(line => {

    if (/^(?:số\s+chương|trạng\s+thái)\s*:/i.test(line)) {

      return;

    }

    let matched = false;

    for (const def of prefixDefs) {

      const m = line.match(def.reg);

      if (m) {

        matched = true;

        if (!extracted[def.key] && m[1].trim()) {

          extracted[def.key] = m[1].trim();

        }

        break;

      }

    }

    if (!matched) {

      bodyLines.push(line);

    }

  });



  // Standardized metadata lines for ALL novels

  const metaLines = [];



  // 1. Tác giả

  const author = (novel && (novel.author_name || novel.author)) || extracted['author'] || 'Đang cập nhật';

  metaLines.push({ label: 'Tác giả:', value: author });



  // 2. Thể loại

  let genresStr = '';

  if (novel && Array.isArray(novel.genres) && novel.genres.length) {

    const unique = Array.from(new Set(novel.genres.map(g => (g || '').trim()).filter(Boolean)));

    genresStr = unique.slice(0, 5).join(', ');

  } else if (extracted['genres']) {

    genresStr = extracted['genres'];

  }

  if (!genresStr) genresStr = 'Tiên Hiệp, Huyền Huyễn';

  metaLines.push({ label: 'Thể loại:', value: genresStr });



  // 3. Số chương

  const totalChaps = actualCount || (novel && (novel.actual_chapters || novel.total_chapters)) || 0;

  const chapsDisplay = totalChaps > 0 ? `${Number(totalChaps).toLocaleString('en-US')} chương` : 'Đang cập nhật';

  metaLines.push({ label: 'Số chương:', value: chapsDisplay });



  // 4. Trạng thái

  const rawStatus = (novel && novel.status) || '';

  const isFull = rawStatus.toLowerCase().includes('full') || rawStatus.toLowerCase().includes('hoàn');

  const statusDisplay = isFull ? 'Full (Hoàn thành)' : 'Đang ra';

  metaLines.push({ label: 'Trạng thái:', value: statusDisplay });



  // 5. Dịch / Biên / Nguồn

  if (extracted['dich']) {

    metaLines.push({ label: 'Dịch:', value: extracted['dich'] });

  }

  if (extracted['bien']) {

    metaLines.push({ label: 'Biên:', value: extracted['bien'] });

  }

  if (extracted['nguon']) {

    metaLines.push({ label: 'Nguồn:', value: extracted['nguon'] });

  } else if (!extracted['dich'] && !extracted['bien']) {

    metaLines.push({ label: 'Nguồn:', value: 'Sưu tầm' });

  }



  let html = '<div class="detail-meta-group">';

  metaLines.forEach(item => {

    html += `<div class="detail-meta-line"><strong>${escapeHtml(item.label)}</strong><span>${escapeHtml(item.value)}</span></div>`;

  });

  html += '</div>';



  if (bodyLines.length > 0) {

    html += '<div class="detail-synopsis-body">';

    bodyLines.forEach(p => {

      html += `<p>${escapeHtml(p)}</p>`;

    });

    html += '</div>';

  } else {

    html += '<div class="detail-synopsis-body"><p>Nội dung bộ truyện đang được cập nhật, kính mời quý độc giả đón đọc...</p></div>';

  }



  return html;

}



/**

 * TIÊN KIẾM TÔNG - CLIENT UI CONTROLLER

 * Pixel-faithful to 01_trang_chu_homepage.jpg, 02_chi_tiet_truyen_detail.jpg, 03_trinh_doc_truyen_reader.jpg

 */



const API_BASE = (window.location.origin.includes('localhost:8000') || window.location.origin.includes('127.0.0.1:8000'))

  ? `${window.location.origin}/api`

  : 'http://localhost:8000/api';



const STORAGE_KEYS = {

  READING_HISTORY: 'tkt_reading_history_v1',

  READER_SETTINGS: 'tkt_reader_settings_v1',

  AUTH_USER: 'tkt_auth_user_v1',

  AUTH_TOKEN: 'tkt_auth_token_v1',

};



// 12 Distinct Cultivation Cover Themes for custom SVG covers when upstream cover fails

const COVER_PALETTES = [

  { bg1: '#091E2C', bg2: '#050B12', accent: '#22D3EE', gold: '#FBBF24', symbol: '劍' },

  { bg1: '#281018', bg2: '#0B0609', accent: '#F87171', gold: '#F59E0B', symbol: '魔' },

  { bg1: '#0D261C', bg2: '#060F0B', accent: '#34D399', gold: '#FDE68A', symbol: '丹' },

  { bg1: '#1E1436', bg2: '#090712', accent: '#A78BFA', gold: '#FBBF24', symbol: '道' },

  { bg1: '#2B1D0A', bg2: '#0D0904', accent: '#FBBF24', gold: '#FEF08A', symbol: '帝' },

  { bg1: '#0F2537', bg2: '#060D14', accent: '#38BDF8', gold: '#F59E0B', symbol: '仙' },

  { bg1: '#261229', bg2: '#0B060D', accent: '#E879F9', gold: '#FDE68A', symbol: '神' },

  { bg1: '#162919', bg2: '#070E08', accent: '#4ADE80', gold: '#FBBF24', symbol: '靈' },

  { bg1: '#291811', bg2: '#0C0705', accent: '#FB923C', gold: '#FDE68A', symbol: '武' },

  { bg1: '#101F33', bg2: '#050A11', accent: '#60A5FA', gold: '#F59E0B', symbol: '天' },

  { bg1: '#241B10', bg2: '#0A0805', accent: '#F59E0B', gold: '#FEF3C7', symbol: '尊' },

  { bg1: '#13262A', bg2: '#060D0F', accent: '#2DD4BF', gold: '#FBBF24', symbol: '玄' },

];



// Curated Shopee Affiliate Products

const DEFAULT_AFFILIATES = [

  {

    id: 'aff-1',

    title: 'Thác Nước Phong Thủy Tu Tiên',

    price_discount: '400.000đ',

    price_original: '500.000đ',

    image_url: 'https://images.unsplash.com/photo-1544787219-7f47ccb76574?auto=format&fit=crop&w=300&q=80',

    affiliate_url: 'https://shopee.vn/search?keyword=thac%20nuoc%20phong%20thuy%20tram%20huong',

    tag: 'Linh Bảo Quán'

  },

  {

    id: 'aff-2',

    title: 'Bộ Ấm Trà Tử Sa Ngũ Hành',

    price_discount: '200.000đ',

    price_original: '500.000đ',

    image_url: 'https://images.unsplash.com/photo-1576092768241-dec231879fc3?auto=format&fit=crop&w=300&q=80',

    affiliate_url: 'https://shopee.vn/search?keyword=bo%20am%20tra%20tu%20sa',

    tag: 'Linh Bảo Quán'

  },

  {

    id: 'aff-3',

    title: 'Bàn Phím Cơ Xianxia RGB',

    price_discount: '900.000đ',

    price_original: '1.500.000đ',

    image_url: 'https://images.unsplash.com/photo-1595225476474-87563907a212?auto=format&fit=crop&w=300&q=80',

    affiliate_url: 'https://shopee.vn/search?keyword=ban%20phim%20co%20rgb',

    tag: 'Linh Bảo Quán'

  },

  {

    id: 'aff-4',

    title: 'Trà Thảo Mộc Linh Chi Tu Tiên',

    price_discount: '150.000đ',

    price_original: '220.000đ',

    image_url: 'https://images.unsplash.com/photo-1563822249548-9a72b6353cd1?auto=format&fit=crop&w=300&q=80',

    affiliate_url: 'https://shopee.vn/search?keyword=tra%20thao%20moc%20linh%20chi',

    tag: 'Phúc Địa Tu Tiên'

  },

  {

    id: 'aff-5',

    title: 'Bàn Phím Cơ Thần Binh Gateron',

    price_discount: '1.200.000đ',

    price_original: '1.650.000đ',

    image_url: 'https://images.unsplash.com/photo-1587829741301-dc798b83add3?auto=format&fit=crop&w=300&q=80',

    affiliate_url: 'https://shopee.vn/search?keyword=ban%20phim%20co%20gateron',

    tag: 'Phúc Địa Tu Tiên'

  },

  {

    id: 'aff-6',

    title: 'Linh Phù Bình An Trừ Tà',

    price_discount: '50.000đ',

    price_original: '99.000đ',

    image_url: 'https://images.unsplash.com/photo-1607344645866-009c320b63e0?auto=format&fit=crop&w=300&q=80',

    affiliate_url: 'https://shopee.vn/search?keyword=phu%20binh%20an%20ma%20vang',

    tag: 'Phúc Địa Tu Tiên'

  }

];



const state = {

  novels: [],

  totalNovels: 0,

  genres: [],

  affiliates: [...DEFAULT_AFFILIATES],



  heroList: [],

  heroIndex: 0,

  heroTimer: null,



  libMode: 'library',

  libSearch: '',

  libGenre: '',

  libStatus: '',

  libSort: 'updated',

  libPage: 1,

  libLimit: 12,



  currentNovel: null,

  currentChapters: [],

  chapterSearch: '',

  chapterSortAsc: true,

  chapterPage: 1,

  chapterPageSize: 48,



  currentChapter: null,

  readerSettings: {

    fontSize: 19,

    theme: 'dark',

  },

  readingHistory: {},

  currentUser: null

};



// 12 Distinct Panoramic World Landscapes & Aura Themes (1 per novel so each novel has a distinct world atmosphere)

const NOVEL_WORLD_SCENES = [

  {

    sceneUrl: 'https://images.unsplash.com/photo-1518709268805-4e9042af9f23?auto=format&fit=crop&w=1400&q=85',

    theme: 'cyan',

    worldName: 'U Minh Huyền Giới',

    realmLabel: 'ĐỈNH PHONG • KIM ĐAN'

  },

  {

    sceneUrl: 'https://images.unsplash.com/photo-1509198397868-475647b2a1e5?auto=format&fit=crop&w=1400&q=85',

    theme: 'gold',

    worldName: 'Cửu Thiên Tiên Vực',

    realmLabel: 'ĐẠI THỪA • ĐỘ KIẾP'

  },

  {

    sceneUrl: 'https://images.unsplash.com/photo-1534447677768-be436bb09401?auto=format&fit=crop&w=1400&q=85',

    theme: 'violet',

    worldName: 'Tinh Hà Thần Vực',

    realmLabel: 'THÁI CỔ • NGUYÊN ANH'

  },

  {

    sceneUrl: 'https://images.unsplash.com/photo-1464822759023-fed622ff2c3b?auto=format&fit=crop&w=1400&q=85',

    theme: 'emerald',

    worldName: 'Thái Hoa Linh Sơn',

    realmLabel: 'CHÂN TIÊN • HÓA THẦN'

  },

  {

    sceneUrl: 'https://images.unsplash.com/photo-1579783900882-c0d3dad7b119?auto=format&fit=crop&w=1400&q=85',

    theme: 'crimson',

    worldName: 'Huyết Hải Ma Vực',

    realmLabel: 'NGHỊCH THIÊN • MA TÔN'

  },

  {

    sceneUrl: 'https://images.unsplash.com/photo-1506744038136-46273834b3fb?auto=format&fit=crop&w=1400&q=85',

    theme: 'cyan',

    worldName: 'Vân Mộng Tiên Cảnh',

    realmLabel: 'TIÊN ĐẾ • VẠN CỔ'

  },

  {

    sceneUrl: 'https://images.unsplash.com/photo-1519681393784-d120267933ba?auto=format&fit=crop&w=1400&q=85',

    theme: 'violet',

    worldName: 'Bắc Hàn Tuyết Vực',

    realmLabel: 'BĂNG PHONG • TRÚC CƠ'

  },

  {

    sceneUrl: 'https://images.unsplash.com/photo-1470071459604-3b5ec3a7fe05?auto=format&fit=crop&w=1400&q=85',

    theme: 'emerald',

    worldName: 'Thanh Vân Mê Lâm',

    realmLabel: 'LINH DƯỢC • ĐAN TÔN'

  },

  {

    sceneUrl: 'https://images.unsplash.com/photo-1507525428034-b723cf961d3e?auto=format&fit=crop&w=1400&q=85',

    theme: 'gold',

    worldName: 'Đông Hải Long Cung',

    realmLabel: 'THƯƠNG MANG • HOÀNG CỰC'

  },

  {

    sceneUrl: 'https://images.unsplash.com/photo-1451187580459-43490279c0fa?auto=format&fit=crop&w=1400&q=85',

    theme: 'cyan',

    worldName: 'Hư Không Tinh Giới',

    realmLabel: 'VŨ TRỤ • ĐẠI ĐẠO'

  },

  {

    sceneUrl: 'https://images.unsplash.com/photo-1513836279014-a89f7a76ae86?auto=format&fit=crop&w=1400&q=85',

    theme: 'emerald',

    worldName: 'Thần Mộc Cổ Giới',

    realmLabel: 'THƯỢNG CỔ • LINH HOÀNG'

  },

  {

    sceneUrl: 'https://images.unsplash.com/photo-1542273917363-3b1817f69a2d?auto=format&fit=crop&w=1400&q=85',

    theme: 'gold',

    worldName: 'Hoàng Tuyền Bí Cảnh',

    realmLabel: 'HUYỀN BÍ • LUYỆN HƯ'

  }

];



function getNovelWorldScene(novel, idx = -1) {

  if (idx >= 0) {

    return NOVEL_WORLD_SCENES[idx % NOVEL_WORLD_SCENES.length];

  }

  const key = String(novel?.slug || novel?.id || novel?.title || 'default');

  let hash = 0;

  for (let i = 0; i < key.length; i++) {

    hash = (hash * 33 + key.charCodeAt(i)) >>> 0;

  }

  return NOVEL_WORLD_SCENES[hash % NOVEL_WORLD_SCENES.length];

}



// =========================================================================

// DISTINCT COVER RESOLVER (Proxies real cover + Unique Custom SVG Cover per Novel)

// =========================================================================

function getDistinctFallbackBySlug(novelOrSlug, idx = 0) {

  const title = (typeof novelOrSlug === 'object' && novelOrSlug !== null)

    ? (novelOrSlug.title || 'TIÊN HIỆP')

    : String(novelOrSlug || 'TIÊN HIỆP');

  const author = (typeof novelOrSlug === 'object' && novelOrSlug !== null)

    ? (novelOrSlug.author_name || 'KhoTruyenChu')

    : 'KhoTruyenChu';



  let hash = idx >= 0 ? idx : 0;

  for (let i = 0; i < title.length; i++) {

    hash = (hash * 31 + title.charCodeAt(i)) >>> 0;

  }

  const p = COVER_PALETTES[hash % COVER_PALETTES.length];



  // Split title into up to 3 lines for clean book-cover typography

  const words = title.toUpperCase().split(/\s+/);

  const lines = [];

  let current = '';

  for (const w of words) {

    if ((current + ' ' + w).trim().length <= 12 || !current) {

      current = (current + ' ' + w).trim();

    } else {

      lines.push(current);

      current = w;

    }

  }

  if (current) lines.push(current);

  const displayLines = lines.slice(0, 3);



  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 400" width="300" height="400">

    <defs>

      <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">

        <stop offset="0%" stop-color="${p.bg1}"/>

        <stop offset="100%" stop-color="${p.bg2}"/>

      </linearGradient>

      <radialGradient id="glow" cx="50%" cy="38%" r="48%">

        <stop offset="0%" stop-color="${p.accent}" stop-opacity="0.42"/>

        <stop offset="100%" stop-color="${p.bg2}" stop-opacity="0"/>

      </radialGradient>

    </defs>

    <rect width="300" height="400" fill="url(#bg)"/>

    <circle cx="150" cy="155" r="135" fill="url(#glow)"/>

    <rect x="12" y="12" width="276" height="376" rx="10" fill="none" stroke="${p.accent}" stroke-opacity="0.45" stroke-width="1.5"/>

    <rect x="18" y="18" width="264" height="364" rx="7" fill="none" stroke="${p.gold}" stroke-opacity="0.28" stroke-width="1"/>

    <circle cx="150" cy="145" r="68" fill="none" stroke="${p.accent}" stroke-opacity="0.35" stroke-width="1.5" stroke-dasharray="6 4"/>

    <circle cx="150" cy="145" r="52" fill="none" stroke="${p.gold}" stroke-opacity="0.4" stroke-width="1"/>

    <text x="150" y="168" text-anchor="middle" fill="${p.accent}" fill-opacity="0.28" font-size="76" font-weight="900" font-family="serif">${p.symbol}</text>

    ${displayLines.map((line, i) => `<text x="150" y="${255 + i * 32}" text-anchor="middle" fill="${i === 0 ? '#FFFFFF' : p.gold}" font-size="23" font-weight="800" font-family="Be Vietnam Pro, sans-serif">${escapeHtml(line)}</text>`).join('')}

    <text x="150" y="362" text-anchor="middle" fill="#CBD5E1" font-size="13" font-weight="600" font-family="Be Vietnam Pro, sans-serif">${escapeHtml(author)}</text>

  </svg>`;



  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;

}



// Default high-fidelity covers from original design

const DESIGN_COVERS = [

  'assets/cover_feat_1.jpg',

  'assets/cover_feat_2.jpg',

  'assets/cover_feat_3.jpg',

  'assets/cover_feat_4.jpg',

  'assets/rank_cover_1.jpg',

  'assets/rank_cover_2.jpg',

  'assets/rank_cover_3.jpg',

  'assets/rank_cover_4.jpg',

  'assets/rank_cover_5.jpg'

];



function resolveNovelCover(novel, idx = -1) {

  const fallback = getDistinctFallbackBySlug(novel, idx);

  const raw = novel?.cover_url || '';

  if (!raw) {

    return fallback;

  }

  // Local stored covers: /covers/<slug>.jpg

  if (raw.startsWith('/')) {

    return raw;

  }

  if (raw.startsWith('http://') || raw.startsWith('https://')) {

    return `${API_BASE}/proxy-cover?url=${encodeURIComponent(raw)}`;

  }

  return raw;

}



// =========================================================================

// INITIALIZATION

// =========================================================================

document.addEventListener('DOMContentLoaded', async () => {

  initThemeMode();

  loadLocalState();

  applyReaderSettings();

  bindGlobalEvents();



  await Promise.all([

    fetchGenres(),

    fetchHomeData()

  ]);



  renderAffiliatesEverywhere();

  handleRouteChange();

  window.addEventListener('hashchange', handleRouteChange);

});



function loadLocalState() {

  try {

    const savedHistory = localStorage.getItem(STORAGE_KEYS.READING_HISTORY);

    if (savedHistory) {

      state.readingHistory = JSON.parse(savedHistory) || {};

    }

    const savedSettings = localStorage.getItem(STORAGE_KEYS.READER_SETTINGS);

    if (savedSettings) {

      state.readerSettings = { ...state.readerSettings, ...JSON.parse(savedSettings) };

    }

    const savedUser = localStorage.getItem(STORAGE_KEYS.AUTH_USER);

    if (savedUser) {

      state.currentUser = JSON.parse(savedUser);

    }

  } catch (e) {

    console.warn('Could not load localStorage state:', e);

  }

  updateCultivationStatsUI();

  renderUserBadge();

}



function saveReadingHistory(novel, chapter) {

  if (!novel || !chapter) return;

  const key = novel.slug || novel.id;

  const prevEntry = state.readingHistory[key] || { readChapterIds: [] };

  const readSet = new Set(prevEntry.readChapterIds || []);

  readSet.add(chapter.id);



  state.readingHistory[key] = {

    novelId: novel.id,

    novelSlug: novel.slug || novel.id,

    novelTitle: novel.title,

    authorName: novel.author_name,

    coverUrl: novel.cover_url || '',

    chapterId: chapter.id,

    chapterNumber: chapter.chapter_number,

    chapterTitle: chapter.title,

    readChapterIds: Array.from(readSet),

    updatedAt: Date.now()

  };



  try {

    localStorage.setItem(STORAGE_KEYS.READING_HISTORY, JSON.stringify(state.readingHistory));

  } catch (_) {}

  updateCultivationStatsUI();



  // Grant cultivator tu vi (exp) and advance realm

  grantCultivatorExp(novel.slug || novel.id, chapter.chapter_number);

}



function updateCultivationStatsUI() {

  const entries = Object.values(state.readingHistory);

  let totalReadChaps = 0;

  entries.forEach(item => {

    totalReadChaps += (item.readChapterIds ? item.readChapterIds.length : 1);

  });



  const navBadge = document.getElementById('nav-history-count');

  if (navBadge) navBadge.textContent = entries.length;



  const statReadEl = document.getElementById('stat-home-read-chapters');

  if (statReadEl) statReadEl.textContent = Math.max(33, totalReadChaps);



  const statRealmEl = document.getElementById('stat-user-realm');

  if (statRealmEl) {

    statRealmEl.textContent = getUserCultivationRealm(totalReadChaps);

  }

}



function getUserCultivationRealm(readCount) {

  if (readCount < 5) return 'Luyện Khí';

  if (readCount < 15) return 'Trúc Cơ';

  if (readCount < 35) return 'Kim Đan';

  if (readCount < 70) return 'Nguyên Anh';

  return 'Đại Thừa Kỳ';

}



function getChapterRealmInfo(chapterNumber) {

  if (chapterNumber <= 2) {

    return { name: 'Luyện Khí', color: '#34D399', icon: 'fa-spa' };

  }

  if (chapterNumber === 3 || (chapterNumber > 10 && chapterNumber <= 30)) {

    return { name: 'Trúc Cơ', color: '#F59E0B', icon: 'fa-fire-flame-Simple' };

  }

  if (chapterNumber === 4 || (chapterNumber > 30 && chapterNumber <= 70)) {

    return { name: 'Kim Đan', color: '#FBBF24', icon: 'fa-sun' };

  }

  return { name: 'Nguyên Anh', color: '#A78BFA', icon: 'fa-dharmachakra' };

}



// =========================================================================

// API FETCHERS

// =========================================================================

async function fetchGenres() {

  try {

    const res = await fetch(`${API_BASE}/admin/genres`);

    if (!res.ok) return;

    const data = await res.json();

    if (Array.isArray(data)) {

      state.genres = data;

      renderGenrePills();

    }

  } catch (_) {}

}



async function fetchTopNovelsByViews(limit = 5) {

  try {

    const res = await fetch(`${API_BASE}/novels?sort=views&limit=${limit}`);

    if (res.ok) {

      const payload = await res.json();

      const topList = payload.data || payload.novels || [];

      if (topList.length > 0) return topList;

    }

  } catch (err) {

    console.warn('Could not fetch top novels by views from API:', err);

  }

  return [];

}



async function fetchHomeData() {

  try {

    // 1. Call API get top 5 theo số lượt đọc (views) cho slider (không hard code)

    const topViewsPromise = fetchTopNovelsByViews(5);



    // 2. Fetch danh sách novels cho bảng xếp hạng và danh sách

    const res = await fetch(`${API_BASE}/admin/novels?page=1&limit=30`);

    if (!res.ok) throw new Error('API error');

    const payload = await res.json();

    const list = payload.data || payload.novels || [];

    state.novels = list;

    state.totalNovels = payload.total || list.length;



    const statNovelsEl = document.getElementById('stat-home-novels');

    if (statNovelsEl) statNovelsEl.textContent = state.totalNovels;



    let top5 = await topViewsPromise;

    if (!top5 || top5.length === 0) {

      // Fallback nếu API riêng bị lỗi: sắp xếp theo số lượt đọc giảm dần

      top5 = getRankingSortedNovels(list, 'views_all').slice(0, 5);

    }



    state.heroList = top5;

    loadShopeeTicker(list);

    renderFeatured4Showcase(top5);

    renderHomeRanking(list);

    renderHomeNovelsGrid(list);

  } catch (err) {

    console.error('Error loading home novels:', err);

  }

}



// =========================================================================

// ROUTER

// =========================================================================

function handleRouteChange() {

  const hash = window.location.hash || '#/';

  const cleanHash = hash.startsWith('#') ? hash.slice(1) : hash;

  const parts = cleanHash.split('/').filter(Boolean);



  const mainNav = document.querySelector('.main-nav');

  if (mainNav) mainNav.classList.remove('mobile-open');



  document.querySelectorAll('.nav-link').forEach(link => link.classList.remove('active'));



  if (parts.length === 0) {

    setActiveNav('home');

    showView('view-home');

    window.scrollTo({ top: 0, behavior: 'smooth' });

    return;

  }



  const route = parts[0];



  if (route === 'library' || route === 'forum') {

    setActiveNav(route);

    state.libMode = 'library';

    openLibraryView();

  } else if (route === 'ranking') {

    setActiveNav('ranking');

    state.libMode = 'ranking';

    state.libSort = 'chapters';

    openLibraryView();

  } else if (route === 'cultivation') {

    setActiveNav('cultivation');

    state.libMode = 'cultivation';

    openLibraryView();

  } else if (route === 'novel' && parts[1]) {

    showView('view-detail');

    loadNovelDetail(decodeURIComponent(parts[1]));

  } else if (route === 'feedback') {
    showView('view-feedback');
    window.scrollTo({ top: 0, behavior: 'smooth' });
    updatePageSEO({
      title: 'Phi Thư Truyền Tin - Góp Ý & Báo Lỗi | KhoTruyenChu',
      description: 'Gửi phi thư truyền âm, góp ý giao diện, báo lỗi chương hoặc yêu cầu bổ sung truyện mới cho KhoTruyenChu.'
    });

  } else if (route === 'read' && parts[1] && parts[2]) {

    showView('view-reader');

    loadReaderChapter(decodeURIComponent(parts[1]), decodeURIComponent(parts[2]));

  } else {

    setActiveNav('home');

    showView('view-home');

  }

}



function setActiveNav(routeKey) {

  const target = document.querySelector(`.nav-link[data-route="${routeKey}"]`);

  if (target) target.classList.add('active');

}



function triggerGateTransition() {

  const gate = document.getElementById('celestial-gate');

  if (!gate) return;

  gate.classList.remove('opening');

  void gate.offsetWidth; // Force CSS reflow

  gate.classList.add('opening');

  setTimeout(() => {

    gate.classList.remove('opening');

  }, 520);

}



function showView(viewId) {

  triggerGateTransition();

  document.querySelectorAll('.view-section').forEach(sec => {

    sec.classList.toggle('active', sec.id === viewId);

  });

}



// =========================================================================

// HOMEPAGE RENDERERS (01_trang_chu_homepage.jpg)

// =========================================================================

// =========================================================================

// LONG RUNNING TICKER BAR (SHOPEE DEALS & HOT NOVELS)

// =========================================================================

async function loadShopeeTicker(novels = []) {

  const track = document.getElementById('ticker-track');

  if (!track) return;



  try {

    const res = await fetch(`${API_BASE}/affiliates`);

    let affiliates = [];

    if (res.ok) {

      affiliates = await res.json();

    }



    const items = [];

    // 1. Add active Shopee affiliate links

    affiliates.forEach(a => {

      items.push(`

        <a href="${escapeHtml(a.affiliate_url)}" target="_blank" rel="noopener noreferrer" class="ticker-item" onclick="recordAffiliateClick('${a.id}')">

          <span class="ticker-pill-tag">${escapeHtml(a.tag || 'Shopee Deal')}</span>

          <strong>${escapeHtml(a.title)}</strong>

          ${a.price_discount ? `<span class="ticker-price">${escapeHtml(a.price_discount)}</span>` : ''}

          <i class="fa-solid fa-arrow-up-right-from-square" style="font-size: 0.72rem; color: #38bdf8;"></i>

        </a>

      `);

    });



    // 2. Add trending novel announcements

    (novels.slice(0, 5)).forEach(n => {

      items.push(`

        <a href="#/novel/${encodeURIComponent(n.slug || n.id)}" class="ticker-item">

          <span class="ticker-pill-tag" style="background: rgba(245, 158, 11, 0.2); color: #fbbf24; border-color: rgba(245, 158, 11, 0.4);"><i class="fa-solid fa-fire"></i> HOT</span>

          <strong>${escapeHtml(n.title)}</strong>

          <span style="color: #94a3b8; font-size: 0.8rem;">(${n.total_chapters || n.actual_chapters || 0} chương)</span>

        </a>

      `);

    });



    if (items.length === 0) {

      items.push(`

        <span class="ticker-item"><i class="fa-solid fa-bolt text-gold"></i> Chào mừng Đạo Hữu đến với KhoTruyenChu - Kho truyện tu tiên huyền huyễn tốc độ cao!</span>

      `);

    }



    // Duplicate track so infinite ticker scroll loops smoothly without seams

    track.innerHTML = items.join('') + items.join('');

  } catch (err) {

    console.warn('Could not load ticker:', err);

  }

}



window.recordAffiliateClick = function(id) {

  try {

    fetch(`${API_BASE}/affiliates/${id}/click`, { method: 'POST' });

  } catch (_) {}

};



// =========================================================================

// TALL DYNAMIC HERO SLIDER (AUTO-SLIDE & CONTROLS)

// =========================================================================

// =========================================================================

// HERO FEATURED INFINITE CIRCULAR SLIDER (TOP 5 THEO SỐ LƯỢT ĐỌC)

// =========================================================================

const heroSliderState = {

  currentIndex: 1, // 1 represents the first real slide (0 is clone of last slide)

  timer: null,

  totalRealSlides: 0,

  isTransitioning: false

};



function buildHeroSlideItemHtml(novel, rankIndex, isClone = false) {

  const coverSrc = resolveNovelCover(novel, rankIndex);

  const fallback = getDistinctFallbackBySlug(novel, rankIndex);

  const slugOrId = encodeURIComponent(novel.slug || novel.id);

  const author = novel.author || novel.author_name || 'Đang cập nhật';

  const chapCount = novel.actual_chapters || novel.total_chapters || 0;

  const views = formatCompactNumber(Math.max(chapCount * 1250, novel.views_count || 120000));

  const rating = getNovelRating(novel);

  const genres = (novel.genres && novel.genres.length) ? novel.genres : ['Tiên Hiệp', 'Huyền Huyễn'];

  const genreTag = genres[0] || 'Nổi Bật';

  const desc = cleanDescription(novel.description, chapCount) || 'Tác phẩm đặc sắc đang được hàng triệu độc giả đón đọc mỗi ngày trên KhoTruyenChu...';



  return `

    <div class="hero-slide-item ${isClone ? 'is-clone' : ''}" data-rank="${rankIndex + 1}">

      <div class="hero-slide-bg-blur" style="background-image: url('${escapeHtml(coverSrc)}');"></div>

      <div class="hero-slide-content">

        <div class="hero-slide-cover-wrap" onclick="window.location.hash='#/novel/${slugOrId}'">

          <img src="${escapeHtml(coverSrc)}" alt="${escapeHtml(novel.title)}" onerror="this.onerror=null;this.src='${fallback}'" loading="lazy">

          <span class="hero-slide-badge"><i class="fa-solid fa-fire"></i> Top ${rankIndex + 1}</span>

        </div>

        <div class="hero-slide-info">

          <span class="hero-slide-genre-pill"><i class="fa-solid fa-bookmark"></i> ${escapeHtml(genreTag)}</span>

          <h2 class="hero-slide-title" onclick="window.location.hash='#/novel/${slugOrId}'">${escapeHtml(novel.title)}</h2>

          <div class="hero-slide-meta">

            <span class="hero-slide-meta-item"><i class="fa-solid fa-pen-nib" style="color:var(--brand);"></i> ${escapeHtml(author)}</span>

            <span class="hero-slide-meta-item"><i class="fa-solid fa-star" style="color:#f59e0b;"></i> ${rating}</span>

            <span class="hero-slide-meta-item"><i class="fa-solid fa-book-open" style="color:var(--brand);"></i> ${chapCount} chương</span>

            <span class="hero-slide-meta-item"><i class="fa-solid fa-eye" style="color:#10b981;"></i> ${views} lượt đọc</span>

          </div>

          <p class="hero-slide-desc">${escapeHtml(desc)}</p>

          <div class="hero-slide-actions">

            <a href="#/novel/${slugOrId}" class="hero-slide-btn-read"><i class="fa-solid fa-book-open-reader"></i> ĐỌC NGAY</a>

            <a href="#/novel/${slugOrId}" class="hero-slide-btn-detail"><i class="fa-solid fa-circle-info"></i> Chi Tiết</a>

          </div>

        </div>

      </div>

    </div>

  `;

}



function updateSliderDots() {

  const N = heroSliderState.totalRealSlides;

  if (N <= 0) return;

  const activeDotIndex = ((heroSliderState.currentIndex - 1) % N + N) % N;

  const dots = document.querySelectorAll('#slider-dots .slider-dot');

  dots.forEach((d, i) => {

    d.classList.toggle('active', i === activeDotIndex);

  });

}



function renderFeatured4Showcase(novels) {

  const slider = document.getElementById('home-featured-slider');

  const dotsContainer = document.getElementById('slider-dots');

  if (!slider) return;



  const topNovels = (novels || []).slice(0, 5);

  heroSliderState.totalRealSlides = topNovels.length;

  heroSliderState.isTransitioning = false;



  if (!topNovels.length) {

    slider.innerHTML = `<div style="padding: 40px; text-align: center; color: var(--ink-2); width:100%;">Đang cập nhật truyện nổi bật...</div>`;

    if (dotsContainer) dotsContainer.innerHTML = '';

    return;

  }



  if (topNovels.length === 1) {

    slider.innerHTML = buildHeroSlideItemHtml(topNovels[0], 0);

    heroSliderState.currentIndex = 0;

    slider.style.transition = 'none';

    slider.style.transform = 'translateX(0%)';

    if (dotsContainer) dotsContainer.innerHTML = '';

    return;

  }



  // Infinite circular carousel:

  // [Clone of Last, Slide 0, Slide 1, ..., Slide N-1, Clone of First]

  const lastNovel = topNovels[topNovels.length - 1];

  const firstNovel = topNovels[0];



  const slidesHtml = [

    buildHeroSlideItemHtml(lastNovel, topNovels.length - 1, true),

    ...topNovels.map((novel, idx) => buildHeroSlideItemHtml(novel, idx, false)),

    buildHeroSlideItemHtml(firstNovel, 0, true)

  ].join('');



  slider.innerHTML = slidesHtml;



  // Position at the first real slide (index 1) without animation

  heroSliderState.currentIndex = 1;

  slider.style.transition = 'none';

  slider.style.transform = 'translateX(-100%)';

  void slider.offsetWidth; // Force layout reflow



  // Render dots for the real slides

  if (dotsContainer) {

    dotsContainer.innerHTML = topNovels.map((_, i) => `

      <span class="slider-dot ${i === 0 ? 'active' : ''}" data-dot-index="${i}"></span>

    `).join('');



    dotsContainer.querySelectorAll('.slider-dot').forEach(dot => {

      dot.addEventListener('click', () => {

        const dotIdx = Number(dot.dataset.dotIndex);

        goToHeroSlide(dotIdx + 1);

        resetHeroSliderTimer();

      });

    });

  }



  initHeroSliderControls();

}



function goToHeroSlide(targetIndex) {

  const slider = document.getElementById('home-featured-slider');

  if (!slider || heroSliderState.totalRealSlides <= 1) return;



  heroSliderState.currentIndex = targetIndex;

  slider.style.transition = 'transform 0.45s cubic-bezier(0.25, 1, 0.5, 1)';

  slider.style.transform = `translateX(-${heroSliderState.currentIndex * 100}%)`;

  updateSliderDots();



  // Safety unlock in case transitionend does not fire

  setTimeout(() => {

    heroSliderState.isTransitioning = false;

  }, 500);

}



function initHeroSliderControls() {

  const slider = document.getElementById('home-featured-slider');

  const prevBtn = document.getElementById('slider-btn-prev');

  const nextBtn = document.getElementById('slider-btn-next');

  const wrapper = document.querySelector('.hero-slider-wrapper');



  if (slider && !slider.dataset.infiniteBound) {

    slider.dataset.infiniteBound = 'true';

    slider.addEventListener('transitionend', () => {

      heroSliderState.isTransitioning = false;

      const N = heroSliderState.totalRealSlides;

      if (N <= 1) return;



      if (heroSliderState.currentIndex >= N + 1) {

        // Slid forward into cloned first slide -> instantly snap to real first slide (index 1)

        slider.style.transition = 'none';

        heroSliderState.currentIndex = 1;

        slider.style.transform = 'translateX(-100%)';

        void slider.offsetWidth;

      } else if (heroSliderState.currentIndex <= 0) {

        // Slid backward into cloned last slide -> instantly snap to real last slide (index N)

        slider.style.transition = 'none';

        heroSliderState.currentIndex = N;

        slider.style.transform = `translateX(-${N * 100}%)`;

        void slider.offsetWidth;

      }

    });

  }



  if (prevBtn) {

    prevBtn.onclick = (e) => {

      e.preventDefault();

      if (heroSliderState.isTransitioning) return;

      heroSliderState.isTransitioning = true;

      goToHeroSlide(heroSliderState.currentIndex - 1);

      resetHeroSliderTimer();

    };

  }

  if (nextBtn) {

    nextBtn.onclick = (e) => {

      e.preventDefault();

      if (heroSliderState.isTransitioning) return;

      heroSliderState.isTransitioning = true;

      goToHeroSlide(heroSliderState.currentIndex + 1);

      resetHeroSliderTimer();

    };

  }



  if (wrapper) {

    wrapper.onmouseenter = () => clearInterval(heroSliderState.timer);

    wrapper.onmouseleave = () => resetHeroSliderTimer();

  }



  resetHeroSliderTimer();

}



function resetHeroSliderTimer() {

  clearInterval(heroSliderState.timer);

  if (heroSliderState.totalRealSlides > 1) {

    heroSliderState.timer = setInterval(() => {

      if (!heroSliderState.isTransitioning) {

        heroSliderState.isTransitioning = true;

        goToHeroSlide(heroSliderState.currentIndex + 1);

      }

    }, 4500);

  }

}



// Webnovel.vn Ranking Grid

function getRankingSortedNovels(novels, sortType = 'views_all') {

  const list = [...novels];

  switch (sortType) {

    case 'chapters':

      return list.sort((a, b) => {

        const ca = a.actual_chapters || a.total_chapters || 0;

        const cb = b.actual_chapters || b.total_chapters || 0;

        return cb - ca;

      });



    case 'rating':

      return list.sort((a, b) => {

        const ra = Number(a.rating_score || a.rating || 4.8);

        const rb = Number(b.rating_score || b.rating || 4.8);

        if (rb !== ra) return rb - ra;

        return (b.actual_chapters || 0) - (a.actual_chapters || 0);

      });



    case 'updated':

    case 'new':

      return list.sort((a, b) => {

        const da = new Date(a.updated_at || a.created_at || 0).getTime();

        const db = new Date(b.updated_at || b.created_at || 0).getTime();

        return db - da;

      });



    case 'full':

      return list

        .filter(n => (n.status || '').toLowerCase().includes('full') || (n.status || '').toLowerCase().includes('hoàn'))

        .sort((a, b) => (b.actual_chapters || 0) - (a.actual_chapters || 0));



    case 'views_month':

    case 'views_week':

    case 'views_day':

    case 'follow':

    case 'views_all':

    default:

      return list.sort((a, b) => {

        const va = (a.views_count && a.views_count > 0) ? a.views_count : ((a.actual_chapters || a.total_chapters || 10) * 1250);

        const vb = (b.views_count && b.views_count > 0) ? b.views_count : ((b.actual_chapters || b.total_chapters || 10) * 1250);

        return vb - va;

      });

  }

}



function renderHomeRanking(novels, sortType = 'views_all') {

  const container = document.getElementById('home-ranking-grid');

  if (!container) return;



  state.rankingSort = sortType;

  const sorted = getRankingSortedNovels(novels || state.novels || [], sortType).slice(0, 6);



  container.innerHTML = sorted.map((n, i) => {

    const rank = i + 1;

    const coverSrc = resolveNovelCover(n, i);

    const fallback = getDistinctFallbackBySlug(n, i);

    const slugOrId = encodeURIComponent(n.slug || n.id);

    const chapCount = n.actual_chapters || n.total_chapters || 0;

    const views = formatCompactNumber(Math.max(chapCount * 1250, n.views_count || 120000));

    const rating = getNovelRating(n);



    let metaHtml = '';

    if (sortType === 'chapters') {

      metaHtml = `<span class="ranking__meta-sub"><i class="fa-solid fa-book-open"></i> ${chapCount.toLocaleString()} chương</span>`;

    } else if (sortType === 'rating') {

      metaHtml = `<span class="ranking__meta-sub"><i class="fa-solid fa-star" style="color:#f59e0b;"></i> ${rating}</span>`;

    } else if (sortType === 'full') {

      metaHtml = `<span class="ranking__meta-sub" style="color:#10b981;"><i class="fa-solid fa-check-circle"></i> FULL • ${chapCount} ch</span>`;

    } else if (sortType === 'updated' || sortType === 'new') {

      metaHtml = `<span class="ranking__meta-sub" style="color:#38bdf8;"><i class="fa-solid fa-clock"></i> Mới cập nhật</span>`;

    } else {

      metaHtml = `<span class="ranking__meta-sub"><i class="fa-solid fa-fire text-orange"></i> ${views} lượt</span>`;

    }



    return `

      <li class="ranking__card" data-rank="${rank}">

        <a href="#/novel/${slugOrId}" title="${escapeHtml(n.title)}" class="ranking__link">

          <figure class="ranking__cover">

            <img src="${escapeHtml(coverSrc)}" alt="${escapeHtml(n.title)}" onerror="this.onerror=null;this.src='${fallback}'" loading="lazy">

            <span class="ranking__badge" data-rank="${rank}">${rank}</span>

          </figure>

          <div style="flex:1; min-width:0; overflow:hidden;">

            <h3 class="ranking__name" title="${escapeHtml(n.title)}">${escapeHtml(n.title)}</h3>

            ${metaHtml}

          </div>

        </a>

      </li>

    `;

  }).join('');



  // Update active pill on home-ranking-tabs

  document.querySelectorAll('#home-ranking-tabs .rank-tab-pill').forEach(btn => {

    const s = btn.getAttribute('data-rank-sort');

    btn.classList.toggle('active', s === sortType || (sortType.startsWith('views') && s === 'views_all'));

  });

}



function renderHomeNovelsGrid(novels) {

  const grid = document.getElementById('home-novels-grid');

  if (!grid) return;



  if (!novels || !novels.length) {

    grid.innerHTML = `<div class="skeleton-loader" style="padding:20px; text-align:center; color:var(--ink-2); width:100%;">Chưa có bộ truyện nào trong danh mục này.</div>`;

    return;

  }



  grid.innerHTML = novels.map((novel, idx) => createNovelCardHTML(novel, idx)).join('');

}



// Webnovel.vn Catx Novel Card

function createNovelCardHTML(novel, idx = 0) {

  const fallback = getDistinctFallbackBySlug(novel, idx);

  const coverSrc = resolveNovelCover(novel, idx);

  const chapCount = novel.actual_chapters || novel.total_chapters || 0;

  const desc = cleanDescription(novel.description, chapCount) || 'Đọc truyện chữ online chất lượng cao, cập nhật chương mới nhanh nhất tại KhoTruyenChu...';

  const slugOrId = encodeURIComponent(novel.slug || novel.id);

  const isFull = !!(novel.status && (novel.status.toLowerCase().includes('full') || novel.status.toLowerCase().includes('hoàn')));

  const statusText = isFull ? 'Full' : 'Đang ra';

  const statusClass = isFull ? 'status-full' : 'status-ongoing';



  return `

    <article class="catx-card" onclick="window.location.hash='#/novel/${slugOrId}'">

      <figure class="catx-card__media">

        <img src="${escapeHtml(coverSrc)}" alt="${escapeHtml(novel.title)}" onerror="this.onerror=null;this.src='${fallback}'" loading="lazy">

        <div class="book-detail__status"><span class="status__text ${statusClass}">${statusText}</span></div>

      </figure>

      <div class="catx-card__body">

        <div class="catx-card__info">

          <h3 class="catx-card__title"><a href="#/novel/${slugOrId}">${escapeHtml(novel.title)}</a></h3>

          <p class="catx-card__desc">${escapeHtml(desc)}</p>

        </div>

        <div class="catx-card__meta">

          <div class="catx-meta">

            <div class="catx-meta__item">

              <i class="fa-solid fa-star" style="color:#f59e0b;"></i>

              <span class="catx-meta__text">${getNovelRating(novel)}</span>

            </div>

            <div class="catx-meta__item">

              <i class="fa-solid fa-book-open" style="color:var(--brand);"></i>

              <span class="catx-meta__text">${chapCount} chương</span>

            </div>

          </div>

        </div>

      </div>

    </article>

  `;

}



function renderGenrePills() {

  const topGenres = state.genres.slice(0, 7);

  const homeBar = document.getElementById('home-genre-pills');

  const libBar = document.getElementById('library-genre-pills');



  const pillsHtml = [

    `<button class="genre-pill ${state.libGenre === '' ? 'active' : ''}" data-genre="">Tất Cả</button>`,

    ...topGenres.map(g => `<button class="genre-pill ${state.libGenre === g.name ? 'active' : ''}" data-genre="${escapeHtml(g.name)}">${escapeHtml(g.name)}</button>`)

  ].join('');



  if (homeBar) {

    homeBar.innerHTML = pillsHtml;

    homeBar.querySelectorAll('.genre-pill').forEach(btn => {

      btn.addEventListener('click', () => {

        homeBar.querySelectorAll('.genre-pill').forEach(b => b.classList.remove('active'));

        btn.classList.add('active');

        const genre = btn.dataset.genre || '';

        const filtered = genre

          ? state.novels.filter(n => (n.genres || []).some(g => g.toLowerCase() === genre.toLowerCase()))

          : state.novels;

        renderHomeNovelsGrid(filtered);

      });

    });

  }



  if (libBar) {

    libBar.innerHTML = pillsHtml;

    libBar.querySelectorAll('.genre-pill').forEach(btn => {

      btn.addEventListener('click', () => {

        state.libGenre = btn.dataset.genre || '';

        state.libPage = 1;

        libBar.querySelectorAll('.genre-pill').forEach(b => b.classList.toggle('active', b === btn));

        renderLibraryGrid();

      });

    });

  }

}



// =========================================================================

// VIEW 2: THƯ VIỆN / XẾP HẠNG / TU LUYỆN

// =========================================================================

async function openLibraryView() {

  showView('view-library');

  window.scrollTo({ top: 0, behavior: 'smooth' });



  const headingEl = document.getElementById('library-heading');

  const subEl = document.getElementById('library-sub');

  const sortSelect = document.getElementById('library-sort-filter');

  const tabBg = document.getElementById('tab-scenic-bg');

  const tabSeal = document.getElementById('tab-seal-char');

  const tabKicker = document.getElementById('tab-kicker-text');



  if (state.libMode === 'ranking') {

    if (tabBg) tabBg.style.backgroundImage = "url('assets/oriental_scroll_gallery.jpg')";

    if (tabSeal) tabSeal.textContent = '金榜';

    if (tabKicker) tabKicker.innerHTML = 'KIM BẢNG PHONG THẦN &bull; ĐỆ NHẤT DANH TÁC';

    if (headingEl) headingEl.textContent = 'BẢNG XẾP HẠNG TU TIÊN';

    if (subEl) subEl.textContent = 'Xếp hạng các bộ đại thuyết có số lượng chương và lượt đọc cao nhất cửu giới';

    if (sortSelect) sortSelect.value = 'chapters';

  } else if (state.libMode === 'cultivation') {

    if (tabBg) tabBg.style.backgroundImage = "url('assets/oriental_sutra_mandala.jpg')";

    if (tabSeal) tabSeal.textContent = '修煉';

    if (tabKicker) tabKicker.innerHTML = 'TU LUYỆN ĐỘT PHÁ &bull; ĐẠO TÂM NHẬP THẦN';

    if (headingEl) headingEl.textContent = 'TỦ TRUYỆN TU LUYỆN CỦA ĐẠO HỮU';

    if (subEl) subEl.textContent = 'Tự động lưu giữ tiến độ đọc truyện để đạo hữu tiếp tục lĩnh ngộ và đột phá cảnh giới';

  } else {

    if (tabBg) tabBg.style.backgroundImage = "url('assets/oriental_jade_pavilion.jpg')";

    if (tabSeal) tabSeal.textContent = '藏經';

    if (tabKicker) tabKicker.innerHTML = 'TÀNG KINH CÁC &bull; BÍCH NGỌC LINH THƯ';

    if (headingEl) headingEl.textContent = 'TÀNG KINH CÁC LINH THƯ';

    if (subEl) subEl.textContent = 'Khám phá toàn bộ vạn bộ danh tác tiên hiệp, kiếm hiệp, huyền huyễn vô thượng';

  }



  if (!state.novels.length) {

    await fetchHomeData();

  }

  renderLibraryGrid();

}



function renderLibraryGrid() {

  const grid = document.getElementById('library-novels-grid');

  const pagEl = document.getElementById('library-pagination');

  if (!grid) return;



  if (state.libMode === 'cultivation') {

    const historyItems = Object.values(state.readingHistory).sort((a, b) => (b.updatedAt || 0) - (a.updatedAt || 0));

    if (!historyItems.length) {

      grid.innerHTML = `<div class="skeleton-loader">Đạo hữu chưa đọc bộ truyện nào. Hãy chọn một bộ truyện tại Trang Chủ để bắt đầu tu luyện!</div>`;

      if (pagEl) pagEl.innerHTML = '';

      return;

    }



    grid.innerHTML = historyItems.map((item, idx) => {

      const fallback = getDistinctFallbackBySlug(item.novelSlug, idx);

      const coverSrc = item.coverUrl

        ? `${API_BASE}/proxy-cover?url=${encodeURIComponent(item.coverUrl)}`

        : fallback;

      return `

        <article class="novel-h-card" onclick="window.location.hash='#/read/${encodeURIComponent(item.novelSlug)}/${encodeURIComponent(item.chapterId)}'">

          <img src="${escapeHtml(coverSrc)}" alt="${escapeHtml(item.novelTitle)}" class="novel-h-cover" onerror="this.onerror=null;this.src='${fallback}'">

          <div class="novel-h-info">

            <div>

              <h3 class="novel-h-title">${escapeHtml(item.novelTitle)}</h3>

              <p class="novel-h-desc text-cyan">Đang đọc: ${escapeHtml(item.chapterTitle || `Chương ${item.chapterNumber}`)}</p>

            </div>

            <div class="realm-pill-btn gold">

              <span class="pill-coin-icon"><i class="fa-solid fa-forward-step"></i></span>

              <span>ĐỌC TIẾP CHƯƠNG ${item.chapterNumber}</span>

            </div>

          </div>

        </article>

      `;

    }).join('');

    if (pagEl) pagEl.innerHTML = '';

    return;

  }



  let list = [...state.novels];



  if (state.libSearch) {

    const q = state.libSearch.toLowerCase();

    list = list.filter(n =>

      (n.title && n.title.toLowerCase().includes(q)) ||

      (n.author_name && n.author_name.toLowerCase().includes(q))

    );

  }



  if (state.libGenre) {

    const g = state.libGenre.toLowerCase();

    list = list.filter(n => (n.genres || []).some(item => item.toLowerCase() === g));

  }



  if (state.libStatus) {

    const s = state.libStatus.toLowerCase();

    list = list.filter(n => (n.status || '').toLowerCase().includes(s));

  }



  if (state.libSort === 'chapters') {

    list.sort((a, b) => (b.actual_chapters || b.total_chapters || 0) - (a.actual_chapters || a.total_chapters || 0));

  } else if (state.libSort === 'views') {

    list.sort((a, b) => (b.views_count || 0) - (a.views_count || 0));

  }



  if (!list.length) {

    grid.innerHTML = `<div class="skeleton-loader">Không tìm thấy bộ truyện nào phù hợp.</div>`;

    if (pagEl) pagEl.innerHTML = '';

    return;

  }



  const totalPages = Math.max(1, Math.ceil(list.length / state.libLimit));

  if (state.libPage > totalPages) state.libPage = 1;

  const start = (state.libPage - 1) * state.libLimit;

  const pageItems = list.slice(start, start + state.libLimit);



  grid.innerHTML = pageItems.map((n, idx) => createNovelCardHTML(n, start + idx)).join('');



  if (pagEl) {

    renderPaginationComponent(pagEl, state.libPage, totalPages, (newPage) => {

      state.libPage = newPage;

      renderLibraryGrid();

      window.scrollTo({ top: 0, behavior: 'smooth' });

    });

  }

}



// =========================================================================

// VIEW 3: CHI TIẾT TRUYỆN (02_chi_tiet_truyen_detail.jpg)

// =========================================================================

async function loadNovelDetail(slugOrId) {

  window.scrollTo({ top: 0, behavior: 'smooth' });

  const chapContainer = document.getElementById('detail-chapter-list');

  if (chapContainer) {

    chapContainer.innerHTML = `<div class="skeleton-loader"><i class="fa-solid fa-circle-notch fa-spin"></i> Đang tải danh sách chương...</div>`;

  }



  try {

    const res = await fetch(`${API_BASE}/admin/novels/${encodeURIComponent(slugOrId)}`);

    if (!res.ok) throw new Error('Novel not found');

    const data = await res.json();



    state.currentNovel = data.novel;

    state.currentChapters = data.chapters || [];

    state.chapterPage = 1;

    state.chapterSearch = '';



    const filterInput = document.getElementById('chapter-filter-input');

    if (filterInput) filterInput.value = '';



    renderNovelDetailUI();

    renderChapterListTable();

  } catch (err) {

    console.error('Error loading novel detail:', err);

  }

}



function renderNovelDetailUI() {

  const n = state.currentNovel;

  if (!n) return;



  const chaps = state.currentChapters || [];

  const actualCount = chaps.length || n.actual_chapters || n.total_chapters || 0;

  const fallback = getDistinctFallbackBySlug(n, 4);

  const coverSrc = resolveNovelCover(n, 4);



  const authorClean = n.author_name || n.author || 'Đang cập nhật';

  const genresClean = (n.genres && n.genres.length) ? n.genres.slice(0, 4).join(', ') : 'Tiên Hiệp, Huyền Huyễn';

  const fullStatusStr = ((n.status || '').toLowerCase().includes('full') || (n.status || '').toLowerCase().includes('hoàn')) ? '[Full]' : '[Đang Ra]';

  const seoTitle = `${n.title} ${fullStatusStr} - Tác giả ${authorClean} | Đọc Truyện Online KhoTruyenChu`;

  const seoDesc = `Đọc truyện ${n.title} của tác giả ${authorClean} full chương mới nhất. Thể loại: ${genresClean}. ` + cleanDescription(n.description).slice(0, 160) + '...';

  const seoUrl = window.location.href;

  const seoImg = resolveNovelCover(n, 0);



  updatePageSEO({

    title: seoTitle,

    description: seoDesc,

    keywords: `${n.title}, doc truyen ${n.title}, truyen ${n.title} full, tac gia ${authorClean}, ${genresClean}, truyen chu online`,

    url: seoUrl,

    image: seoImg,

    jsonLd: {

      '@context': 'https://schema.org',

      '@type': 'Book',

      'name': n.title,

      'author': { '@type': 'Person', 'name': authorClean },

      'genre': genresClean,

      'numberOfPages': actualCount,

      'description': seoDesc,

      'image': seoImg,

      'url': seoUrl

    }

  });



  const breadcrumbTitle = document.getElementById('detail-breadcrumb-title');

  if (breadcrumbTitle) breadcrumbTitle.textContent = n.title;



  const coverImg = document.getElementById('detail-cover-img');

  const titleEl = document.getElementById('detail-title');

  const authorEl = document.getElementById('detail-author');

  const genresWrap = document.getElementById('detail-genres-wrap') || document.getElementById('detail-genres');



  const ratingEl = document.getElementById('detail-rating');

  const viewsEl = document.getElementById('detail-views');

  const chapCountEl = document.getElementById('detail-chapters-count');

  const statusEl = document.getElementById('detail-status-badge') || document.getElementById('detail-status-short');

  const chaptersSub = document.getElementById('detail-chapters-sub');



  if (coverImg) {

    coverImg.src = coverSrc;

    coverImg.onerror = () => {

      coverImg.onerror = null;

      coverImg.src = fallback;

    };

  }

  if (titleEl) titleEl.textContent = n.title;

  if (authorEl) authorEl.textContent = n.author_name || 'Đang cập nhật';



  const descEl = document.getElementById('detail-synopsis-text') || document.getElementById('detail-description');

  if (descEl) {

    descEl.innerHTML = formatNovelDetailDescription(n, actualCount);

  }



  if (genresWrap) {

    const rawGenres = (n.genres && n.genres.length) ? n.genres : ['Tiên Hiệp', 'Huyền Huyễn', 'Đô Thị'];

    genresWrap.innerHTML = rawGenres

      .map(g => `<a class="genre" href="#/library" onclick="state.libGenre='${escapeHtml(g)}'; state.libMode='library';"><i class="fa-solid fa-tag" style="font-size:11px; margin-right:4px; opacity:0.7;"></i>${escapeHtml(g)}</a>`)

      .join('');

  }



  if (ratingEl) ratingEl.textContent = getNovelRating(n);

  if (viewsEl) viewsEl.textContent = formatCompactNumber(Math.max(actualCount * 1250, n.views_count || 120000));

  if (chapCountEl) chapCountEl.textContent = actualCount.toLocaleString('en-US');

  if (chaptersSub) chaptersSub.textContent = `${actualCount} chương`;

  if (statusEl) {

    const isFull = (n.status || '').toLowerCase().includes('full') || (n.status || '').toLowerCase().includes('hoàn');

    statusEl.textContent = isFull ? 'Full' : 'Đang ra';

    statusEl.className = 'status__text ' + (isFull ? 'status-full' : 'status-ongoing');

  }



  const btnFirst = document.getElementById('btn-read-first');

  const btnContinue = document.getElementById('btn-read-continue');

  const btnContinueText = document.getElementById('btn-read-continue-text');

  const slugKey = n.slug || n.id;

  const savedProgress = state.readingHistory[slugKey];



  if (btnFirst) {

    btnFirst.onclick = () => {

      if (!chaps.length) return;

      window.location.hash = `#/read/${encodeURIComponent(slugKey)}/${encodeURIComponent(chaps[0].id)}`;

    };

  }



  if (btnContinue) {

    if (savedProgress && savedProgress.chapterId) {

      if (btnContinueText) btnContinueText.textContent = `Đọc Tiếp (Chương ${savedProgress.chapterNumber})`;

      btnContinue.onclick = () => {

        window.location.hash = `#/read/${encodeURIComponent(slugKey)}/${encodeURIComponent(savedProgress.chapterId)}`;

      };

    } else {

      if (btnContinueText) btnContinueText.textContent = 'Đọc Tiếp';

      btnContinue.onclick = () => {

        if (!chaps.length) return;

        window.location.hash = `#/read/${encodeURIComponent(slugKey)}/${encodeURIComponent(chaps[0].id)}`;

      };

    }

  }



  // Bind Summary vs Chapters capsule tabs (pure CSS classes, no underline)

  const btnTabSummary = document.getElementById('btn-tab-summary');

  const btnTabChapters = document.getElementById('btn-tab-chapters');

  const panelSummary = document.getElementById('panel-summary');

  const panelChapters = document.getElementById('panel-chapters');



  if (btnTabSummary && btnTabChapters && panelSummary && panelChapters) {

    btnTabSummary.removeAttribute('style');

    btnTabChapters.removeAttribute('style');



    btnTabSummary.onclick = () => {

      btnTabSummary.classList.add('is-active');

      btnTabChapters.classList.remove('is-active');

      panelSummary.hidden = false;

      panelChapters.hidden = true;

    };



    btnTabChapters.onclick = () => {

      btnTabChapters.classList.add('is-active');

      btnTabSummary.classList.remove('is-active');

      panelChapters.hidden = false;

      panelSummary.hidden = true;

    };

  }

}



function renderChapterListTable() {

  const container = document.getElementById('detail-chapter-list');

  const pagEl = document.getElementById('detail-chapter-pagination');

  if (!container) return;



  const novel = state.currentNovel;

  const slugKey = novel ? (novel.slug || novel.id) : '';

  let list = [...state.currentChapters];



  if (state.chapterSearch) {

    const q = state.chapterSearch.toLowerCase();

    list = list.filter(c =>

      String(c.chapter_number).includes(q) ||

      (c.title && c.title.toLowerCase().includes(q))

    );

  }



  if (!state.chapterSortAsc) {

    list.reverse();

  }



  if (!list.length) {

    container.innerHTML = `<div style="grid-column: 1 / -1; padding: 24px; text-align: center; color: var(--ink-2);">Chưa có chương nào phù hợp.</div>`;

    if (pagEl) pagEl.innerHTML = '';

    return;

  }



  const totalPages = Math.max(1, Math.ceil(list.length / state.chapterPageSize));

  if (state.chapterPage > totalPages) state.chapterPage = 1;

  const start = (state.chapterPage - 1) * state.chapterPageSize;

  const pageItems = list.slice(start, start + state.chapterPageSize);



  container.innerHTML = pageItems.map(chap => {

    const displayTitle = formatChapterTitle(chap.chapter_number, chap.title);

    return `

      <a href="#/read/${encodeURIComponent(slugKey)}/${encodeURIComponent(chap.id)}" class="chapter-list-item" title="${escapeHtml(displayTitle)}">

        <i class="fa-regular fa-file-lines" style="color: var(--brand); margin-right: 6px;"></i>

        ${escapeHtml(displayTitle)}

      </a>

    `;

  }).join('');



  if (pagEl) {

    renderPaginationComponent(pagEl, state.chapterPage, totalPages, (newPage) => {

      state.chapterPage = newPage;

      renderChapterListTable();

      const panel = document.getElementById('panel-chapters');

      if (panel) {

        panel.scrollIntoView({ behavior: 'smooth', block: 'nearest' });

      }

    });

  }

}



// =========================================================================

// VIEW 4: TRÌNH ĐỌC TRUYỆN (READER)

// =========================================================================

async function loadReaderChapter(novelSlug, chapterId) {

  window.scrollTo({ top: 0, behavior: 'smooth' });

  const contentEl = document.getElementById('chapter-c') || document.getElementById('reader-content');

  if (contentEl) {

    contentEl.innerHTML = `<div style="text-align:center; padding: 40px; color: var(--ink-2);"><i class="fa-solid fa-circle-notch fa-spin"></i> Đang tải nội dung chương...</div>`;

  }



  try {

    if (!state.currentNovel || (state.currentNovel.slug !== novelSlug && state.currentNovel.id !== novelSlug)) {

      const novelRes = await fetch(`${API_BASE}/admin/novels/${encodeURIComponent(novelSlug)}`);

      if (novelRes.ok) {

        const novelData = await novelRes.json();

        state.currentNovel = novelData.novel;

        state.currentChapters = novelData.chapters || [];

      }

    }



    const res = await fetch(`${API_BASE}/admin/chapters/${encodeURIComponent(chapterId)}`);

    if (!res.ok) throw new Error('Chapter not found');

    const chap = await res.json();

    state.currentChapter = chap;



    renderReaderUI(chap);

    if (state.currentNovel) {

      saveReadingHistory(state.currentNovel, chap);

      grantCultivatorExp(state.currentNovel.slug || state.currentNovel.id, chap.chapter_number);

    }

  } catch (err) {

    console.error('Error loading chapter content:', err);

    if (contentEl) contentEl.innerHTML = `<div style="text-align:center; color:#ef4444; padding:30px;">Không thể tải nội dung chương. Vui lòng thử lại!</div>`;

  }

}



function renderReaderUI(chap) {

  const novel = state.currentNovel;

  const fullTitle = formatChapterTitle(chap.chapter_number, chap.title);

  document.title = `${fullTitle} - ${novel ? novel.title : 'KhoTruyenChu'}`;



  const slugKey = novel ? (novel.slug || novel.id) : chap.novel_id;



  const novelTitleLink = document.getElementById('reader-novel-title');

  if (novelTitleLink) {

    novelTitleLink.innerHTML = `<i class="fa-solid fa-arrow-left"></i> ${escapeHtml(novel ? novel.title : 'Quay lại')}`;

    novelTitleLink.href = `#/novel/${encodeURIComponent(slugKey)}`;

  }



  const backDetailLink = document.getElementById('btn-back-detail');

  if (backDetailLink) {

    backDetailLink.href = `#/novel/${encodeURIComponent(slugKey)}`;

  }



  const headingEl = document.getElementById('reader-chapter-heading') || document.getElementById('reader-chapter-title');

  if (headingEl) headingEl.textContent = fullTitle;



  const contentEl = document.getElementById('chapter-c') || document.getElementById('reader-content');

  const rawText = chap.content_text || chap.content || '';

  if (contentEl) {

    contentEl.innerHTML = formatChapterParagraphs(rawText);

  }



  const chapters = state.currentChapters || [];

  const currentIndex = chapters.findIndex(c => c.id === chap.id || c.chapter_number === chap.chapter_number);



  const prevBtns = [document.getElementById('btn-prev-chap'), document.getElementById('btn-prev-chap-top')].filter(Boolean);

  const nextBtns = [document.getElementById('btn-next-chap'), document.getElementById('btn-next-chap-top')].filter(Boolean);



  if (currentIndex > 0) {

    const prevChap = chapters[currentIndex - 1];

    prevBtns.forEach(btn => {

      btn.style.opacity = '1';

      btn.style.pointerEvents = 'auto';

      btn.href = `#/read/${encodeURIComponent(slugKey)}/${encodeURIComponent(prevChap.id)}`;

      btn.onclick = null;

    });

  } else {

    prevBtns.forEach(btn => {

      btn.style.opacity = '0.4';

      btn.style.pointerEvents = 'none';

      btn.removeAttribute('href');

    });

  }



  if (currentIndex >= 0 && currentIndex < chapters.length - 1) {

    const nextChap = chapters[currentIndex + 1];

    nextBtns.forEach(btn => {

      btn.style.opacity = '1';

      btn.style.pointerEvents = 'auto';

      btn.href = `#/read/${encodeURIComponent(slugKey)}/${encodeURIComponent(nextChap.id)}`;

      btn.onclick = null;

    });

  } else {

    nextBtns.forEach(btn => {

      btn.style.opacity = '0.4';

      btn.style.pointerEvents = 'none';

      btn.removeAttribute('href');

    });

  }

}



function formatChapterParagraphs(rawText) {

  if (!rawText || !rawText.trim()) {

    return '<p>Nội dung chương đang được cập nhật...</p>';

  }

  const lines = rawText

    .replace(/\r\n/g, '\n')

    .split(/\n+/)

    .map(l => l.trim())

    .filter(Boolean);



  return lines.map(line => `<p>${escapeHtml(line)}</p>`).join('');

}



function applyReaderSettings() {

  const article = document.getElementById('reader-article');

  const content = document.getElementById('reader-content');



  if (content) {

    content.style.fontSize = `${state.readerSettings.fontSize}px`;

  }



  if (article) {

    article.classList.remove('theme-sepia', 'theme-light');

    if (state.readerSettings.theme === 'sepia') {

      article.classList.add('theme-sepia');

    } else if (state.readerSettings.theme === 'light') {

      article.classList.add('theme-light');

    }

  }



  try {

    localStorage.setItem(STORAGE_KEYS.READER_SETTINGS, JSON.stringify(state.readerSettings));

  } catch (_) {}

}



// =========================================================================

// SHOPEE AFFILIATE RENDERERS

// =========================================================================

function renderAffiliatesEverywhere() {

  // 1. Detail Sidebar: Linh Bảo Quán (02_chi_tiet_truyen_detail.jpg)

  const lbqEl = document.getElementById('detail-affiliate-list');

  if (lbqEl) {

    lbqEl.innerHTML = state.affiliates.slice(0, 3).map((item, idx) => `

      <div class="lbq-item">

        <div class="lbq-item-top">

          <img src="${escapeHtml(item.image_url)}" alt="${escapeHtml(item.title)}" class="lbq-thumb">

          <div class="lbq-info">

            <h4>Sản phẩm ${idx + 1}:<br>${escapeHtml(item.title)}</h4>

            <div class="lbq-price-discount">Giá ${escapeHtml(item.price_discount)}</div>

            <div class="lbq-price-orig">Giá ${escapeHtml(item.price_original)}</div>

          </div>

        </div>

        <a href="${escapeHtml(item.affiliate_url)}" target="_blank" rel="noopener" class="btn-shopee-cyan">

          MUA TRÊN SHOPEE

        </a>

      </div>

    `).join('');

  }



  // 2. Reader Sidebar: Phúc Địa Tu Tiên (03_trinh_doc_truyen_reader.jpg)

  const pdEl = document.getElementById('reader-affiliate-list');

  if (pdEl) {

    pdEl.innerHTML = state.affiliates.slice(3, 6).map(item => `

      <div class="phuc-dia-item">

        <img src="${escapeHtml(item.image_url)}" alt="${escapeHtml(item.title)}" class="pd-thumb">

        <div class="pd-info">

          <div class="pd-title">${escapeHtml(item.title)}</div>

          <span class="pd-shop-tag"><i class="fa-solid fa-bag-shopping"></i> Shopee</span>

          <div class="pd-bottom-row">

            <span class="pd-price">${escapeHtml(item.price_discount)}</span>

            <a href="${escapeHtml(item.affiliate_url)}" target="_blank" rel="noopener" class="btn-mua-ngay">Mua Ngay</a>

          </div>

        </div>

      </div>

    `).join('');

  }



  // 3. Modal: Tàng Bảo Các

  const modalGrid = document.getElementById('modal-affiliate-grid');

  if (modalGrid) {

    modalGrid.innerHTML = state.affiliates.map(item => `

      <div class="lbq-item">

        <div class="lbq-item-top">

          <img src="${escapeHtml(item.image_url)}" alt="${escapeHtml(item.title)}" class="lbq-thumb">

          <div class="lbq-info">

            <h4>${escapeHtml(item.title)}</h4>

            <div class="lbq-price-discount">${escapeHtml(item.price_discount)}</div>

            <div class="lbq-price-orig">${escapeHtml(item.price_original)}</div>

          </div>

        </div>

        <a href="${escapeHtml(item.affiliate_url)}" target="_blank" rel="noopener" class="btn-shopee-cyan">

          MUA TRÊN SHOPEE

        </a>

      </div>

    `).join('');

  }

}



// =========================================================================

// EVENT LISTENERS

// =========================================================================

function bindGlobalEvents() {



  // Ranking Capsule Tabs on Home

  document.querySelectorAll('#home-ranking-tabs .rank-tab-pill').forEach(btn => {

    btn.addEventListener('click', () => {

      const s = btn.getAttribute('data-rank-sort');

      renderHomeRanking(state.novels, s);

    });

  });



  // Ranking Navbar Dropdown Toggle

  const btnRankingToggle = document.getElementById('btn-ranking-toggle');

  const rankingDropdownWrap = document.getElementById('ranking-dropdown-wrap');

  const rankingMenu = document.getElementById('rankingMenu');



  if (btnRankingToggle && rankingMenu) {

    btnRankingToggle.addEventListener('click', (e) => {

      e.preventDefault();

      e.stopPropagation();

      const isOpen = rankingMenu.style.display === 'block';

      rankingMenu.style.display = isOpen ? 'none' : 'block';

      if (rankingDropdownWrap) rankingDropdownWrap.classList.toggle('is-open', !isOpen);

    });



    document.addEventListener('click', (e) => {

      if (!rankingMenu.contains(e.target) && !btnRankingToggle.contains(e.target)) {

        rankingMenu.style.display = 'none';

        if (rankingDropdownWrap) rankingDropdownWrap.classList.remove('is-open');

      }

    });



    rankingMenu.querySelectorAll('.ranking-link').forEach(link => {

      link.addEventListener('click', (e) => {

        const sortType = link.getAttribute('data-rank-sort');

        rankingMenu.style.display = 'none';

        if (rankingDropdownWrap) rankingDropdownWrap.classList.remove('is-open');



        const currentHash = window.location.hash || '#/';

        if (currentHash === '#/' || currentHash === '') {

          e.preventDefault();

          renderHomeRanking(state.novels, sortType);

          const rTitle = document.getElementById('ranking-title');

          if (rTitle) rTitle.scrollIntoView({ behavior: 'smooth', block: 'center' });

        } else {

          sessionStorage.setItem('pendingRankingSort', sortType);

        }

      });

    });

  }



  const heroPrev = document.getElementById('hero-prev');

  const heroNext = document.getElementById('hero-next');

  if (heroPrev) {

    heroPrev.addEventListener('click', () => {

      if (!state.heroList.length) return;

      state.heroIndex = (state.heroIndex - 1 + state.heroList.length) % state.heroList.length;

      updateHeroSlideUI();

      resetHeroTimer();

    });

  }

  if (heroNext) {

    heroNext.addEventListener('click', () => {

      if (!state.heroList.length) return;

      state.heroIndex = (state.heroIndex + 1) % state.heroList.length;

      updateHeroSlideUI();

      resetHeroTimer();

    });

  }



  const headerSearchForm = document.getElementById('header-search-form');

  const headerSearchInput = document.getElementById('header-search-input');

  if (headerSearchForm && headerSearchInput) {

    headerSearchForm.addEventListener('submit', (e) => {

      e.preventDefault();

      state.libSearch = headerSearchInput.value.trim();

      state.libMode = 'library';

      state.libPage = 1;

      const libSearch = document.getElementById('library-search-input');

      if (libSearch) libSearch.value = state.libSearch;

      window.location.hash = '#/library';

      openLibraryView();

    });

  }



  const libSearchInput = document.getElementById('library-search-input');

  const libStatusSelect = document.getElementById('library-status-filter');

  const libSortSelect = document.getElementById('library-sort-filter');



  if (libSearchInput) {

    libSearchInput.addEventListener('input', () => {

      state.libSearch = libSearchInput.value.trim();

      state.libPage = 1;

      renderLibraryGrid();

    });

  }

  if (libStatusSelect) {

    libStatusSelect.addEventListener('change', () => {

      state.libStatus = libStatusSelect.value;

      state.libPage = 1;

      renderLibraryGrid();

    });

  }

  if (libSortSelect) {

    libSortSelect.addEventListener('change', () => {

      state.libSort = libSortSelect.value;

      state.libPage = 1;

      renderLibraryGrid();

    });

  }



  const btnToggleSynopsis = document.getElementById('btn-toggle-synopsis');

  const descEl = document.getElementById('detail-description');

  if (btnToggleSynopsis && descEl) {

    btnToggleSynopsis.addEventListener('click', () => {

      const expanded = descEl.classList.toggle('expanded');

      btnToggleSynopsis.innerHTML = expanded

        ? `Thu gọn <i class="fa-solid fa-chevron-up"></i>`

        : `Xem thêm <i class="fa-solid fa-chevron-down"></i>`;

    });

  }



  const chapFilterInput = document.getElementById('chapter-filter-input');

  if (chapFilterInput) {

    chapFilterInput.addEventListener('input', () => {

      state.chapterSearch = chapFilterInput.value.trim();

      state.chapterPage = 1;

      renderChapterListTable();

    });

  }



  // Reader Controls

  const btnFontDec = document.getElementById('btn-font-dec') || document.getElementById('btn-font-decrease');

  const btnFontInc = document.getElementById('btn-font-inc') || document.getElementById('btn-font-increase');

  const btnTheme = document.getElementById('btn-reader-theme');



  if (btnFontDec) {

    btnFontDec.addEventListener('click', () => {

      state.readerSettings.fontSize = Math.max(14, state.readerSettings.fontSize - 2);

      applyReaderSettings();

    });

  }

  if (btnFontInc) {

    btnFontInc.addEventListener('click', () => {

      state.readerSettings.fontSize = Math.min(32, state.readerSettings.fontSize + 2);

      applyReaderSettings();

    });

  }

  if (btnTheme) {

    btnTheme.addEventListener('click', () => {

      const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

      if (isDark) {

        document.documentElement.removeAttribute('data-theme');

        state.readerSettings.theme = 'light';

      } else {

        document.documentElement.setAttribute('data-theme', 'dark');

        state.readerSettings.theme = 'dark';

      }

      applyReaderSettings();

    });

  }



  // Genre Dropdown Toggle (Reliable click & show)

  const btnGenreToggle = document.getElementById('btn-genre-toggle');

  const genreDropdownWrap = document.getElementById('genre-dropdown-wrap');

  const genreMenu = document.getElementById('genreMenu');

  if (btnGenreToggle && genreMenu) {

    btnGenreToggle.addEventListener('click', (e) => {

      e.preventDefault();

      e.stopPropagation();

      const isOpen = genreMenu.classList.toggle('show');

      if (genreDropdownWrap) genreDropdownWrap.classList.toggle('is-open', isOpen);

      genreMenu.style.display = isOpen ? 'block' : 'none';

      if (isOpen) genreMenu.removeAttribute('hidden');

    });

    document.addEventListener('click', (e) => {

      if (!genreMenu.contains(e.target) && !btnGenreToggle.contains(e.target)) {

        genreMenu.classList.remove('show');

        if (genreDropdownWrap) genreDropdownWrap.classList.remove('is-open');

        genreMenu.style.display = 'none';

      }

    });

  }



  // Category navigation links

  document.querySelectorAll('#top-category-nav .category-nav a, #genreMenu a, #mobileDrawer a[data-genre]').forEach(link => {

    link.addEventListener('click', (e) => {

      const g = link.dataset.genre || '';

      state.libGenre = g;

      state.libMode = 'library';

      state.libPage = 1;

      if (genreMenu) genreMenu.classList.add('hidden');

      const drawer = document.getElementById('mobileDrawer');

      if (drawer) drawer.classList.add('hidden');



      // Update active nav style

      document.querySelectorAll('#top-category-nav .category-nav a').forEach(a => {

        a.classList.toggle('active', (a.dataset.genre || '') === g);

      });



      window.location.hash = '#/library';

      openLibraryView();

    });

  });



  // Ranking Grid loaded directly from novels



  // Catx Genre Tabs on Homepage

  const catxTabs = document.querySelectorAll('#catx-genre-tabs .genre-pill');

  catxTabs.forEach(btn => {

    btn.addEventListener('click', () => {

      catxTabs.forEach(b => b.classList.remove('active'));

      btn.classList.add('active');

      const g = (btn.dataset.genre || '').trim().toLowerCase();

      let filtered = [...state.novels];

      if (g) {

        filtered = filtered.filter(n => {

          const gn = (n.genres || []).map(x => String(x).toLowerCase());

          return gn.some(item => item.includes(g)) || (n.description && n.description.toLowerCase().includes(g));

        });

      }

      renderHomeNovelsGrid(filtered);

    });

  });



  // Mobile Drawer toggle

  const btnMobileDrawer = document.getElementById('btn-mobile-drawer');

  const btnCloseDrawer = document.getElementById('btn-close-drawer');

  const mobileDrawer = document.getElementById('mobileDrawer');

  if (btnMobileDrawer && mobileDrawer) {

    btnMobileDrawer.addEventListener('click', () => {

      mobileDrawer.classList.remove('hidden');

    });

  }

  if (btnCloseDrawer && mobileDrawer) {

    btnCloseDrawer.addEventListener('click', () => {

      mobileDrawer.classList.add('hidden');

    });

  }

  if (mobileDrawer) {

    mobileDrawer.addEventListener('click', (e) => {

      if (e.target === mobileDrawer) {

        mobileDrawer.classList.add('hidden');

      }

    });

  }



  // Keyboard navigation for reader

  window.addEventListener('keydown', (e) => {

    const readerView = document.getElementById('view-reader');

    if (!readerView || !readerView.classList.contains('active')) return;

    if (['INPUT', 'SELECT', 'TEXTAREA'].includes(document.activeElement?.tagName)) return;



    if (e.key === 'ArrowLeft') {

      const prevBtn = document.getElementById('btn-prev-chap') || document.getElementById('btn-prev-chapter');

      if (prevBtn && prevBtn.href) window.location.hash = prevBtn.hash;

    } else if (e.key === 'ArrowRight') {

      const nextBtn = document.getElementById('btn-next-chap') || document.getElementById('btn-next-chapter');

      if (nextBtn && nextBtn.href) window.location.hash = nextBtn.hash;

    }

  });



  // Shopee affiliates ticker / modal if present

  const treasureModal = document.getElementById('treasure-modal');

  const btnBannerTreasure = document.getElementById('btn-banner-open-treasure');

  const btnCloseTreasure = document.getElementById('btn-close-treasure');



  const openModal = () => treasureModal && treasureModal.classList.add('open');

  const closeModal = () => treasureModal && treasureModal.classList.remove('open');



  if (btnBannerTreasure) btnBannerTreasure.addEventListener('click', openModal);

  if (btnCloseTreasure) btnCloseTreasure.addEventListener('click', closeModal);

  if (treasureModal) {

    treasureModal.addEventListener('click', (e) => {

      if (e.target === treasureModal) closeModal();

    });

  }



  const btnMobileMenu = document.getElementById('btn-mobile-menu');

  const mainNav = document.querySelector('.main-nav');

  if (btnMobileMenu && mainNav) {

    btnMobileMenu.addEventListener('click', () => {

      mainNav.classList.toggle('mobile-open');

    });

  }



  // Initialize Cultivator Authentication System

  initAuthSystem();
  initFeedbackForm();
  initSideDonateWidget();

}


function initSideDonateWidget() {
  const widget = document.getElementById('side-donate-widget');
  const btnClose = document.getElementById('btn-close-donate');
  const btnTrigger = document.getElementById('btn-trigger-donate');

  if (btnClose && widget) {
    btnClose.addEventListener('click', () => {
      widget.classList.add('is-collapsed');
      widget.classList.remove('is-expanded');
      if (btnTrigger) btnTrigger.style.display = 'inline-flex';
    });
  }

  if (btnTrigger && widget) {
    btnTrigger.addEventListener('click', () => {
      widget.classList.remove('is-collapsed');
      widget.classList.toggle('is-expanded');
    });
  }
}

function initFeedbackForm() {
  const form = document.getElementById('form-feedback');
  if (!form) return;

  const pillBtns = document.querySelectorAll('#fb-type-group .fb-pill-btn');
  const typeInput = document.getElementById('fb-type');
  pillBtns.forEach(pill => {
    pill.addEventListener('click', () => {
      pillBtns.forEach(p => p.classList.remove('active'));
      pill.classList.add('active');
      if (typeInput) typeInput.value = pill.getAttribute('data-type') || 'Góp ý giao diện & tính năng';
    });
  });

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-feedback');
    const alertEl = document.getElementById('fb-alert');
    const nameEl = document.getElementById('fb-name');
    const contactEl = document.getElementById('fb-contact');
    const typeEl = document.getElementById('fb-type');
    const contentEl = document.getElementById('fb-content');

    const name = nameEl ? nameEl.value.trim() : '';
    const contact = contactEl ? contactEl.value.trim() : '';
    const feedbackType = typeEl ? typeEl.value : 'Ý kiến chung';
    const content = contentEl ? contentEl.value.trim() : '';

    if (!content) {
      if (alertEl) {
        alertEl.style.display = 'block';
        alertEl.style.background = 'rgba(239, 68, 68, 0.15)';
        alertEl.style.color = '#ef4444';
        alertEl.style.border = '1px solid rgba(239, 68, 68, 0.3)';
        alertEl.innerHTML = '<i class="fa-solid fa-circle-exclamation"></i> Vui lòng nhập nội dung truyền âm!';
      }
      return;
    }

    try {
      if (btn) {
        btn.disabled = true;
        btn.innerHTML = '<i class="fa-solid fa-spinner fa-spin"></i> Đang phóng phi thư...';
      }
      const res = await fetch(`${API_BASE}/feedback`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name || 'Ẩn danh đạo hữu',
          contact: contact,
          feedback_type: feedbackType,
          content: content
        })
      });
      const data = await res.json();
      if (res.ok && data.success) {
        if (alertEl) {
          alertEl.style.display = 'block';
          alertEl.style.background = 'rgba(16, 185, 129, 0.15)';
          alertEl.style.color = '#10b981';
          alertEl.style.border = '1px solid rgba(16, 185, 129, 0.3)';
          alertEl.innerHTML = '<i class="fa-solid fa-circle-check"></i> Phi thư đã được chưởng môn tiếp nhận! Đa tạ đạo hữu đã góp ý.';
        }
        form.reset();
      } else {
        throw new Error(data.message || 'Lỗi gửi ý kiến');
      }
    } catch (err) {
      if (alertEl) {
        alertEl.style.display = 'block';
        alertEl.style.background = 'rgba(239, 68, 68, 0.15)';
        alertEl.style.color = '#ef4444';
        alertEl.style.border = '1px solid rgba(239, 68, 68, 0.3)';
        alertEl.innerHTML = `<i class="fa-solid fa-triangle-exclamation"></i> Có lỗi xảy ra: ${escapeHtml(err.message)}`;
      }
    } finally {
      if (btn) {
        btn.disabled = false;
        btn.innerHTML = '<i class="fa-solid fa-paper-plane"></i> Phóng Phi Thư Ngay';
      }
    }
  });
}




// =========================================================================

// UTILITIES

// =========================================================================

function formatChapterTitle(num, rawTitle) {

  if (!rawTitle) return `Chương ${num}`;

  const cleaned = rawTitle.trim();

  if (/^chương\s+\d+/i.test(cleaned)) {

    return cleaned;

  }

  return `Chương ${num}: ${cleaned}`;

}



function cleanDescription(desc, actualCount = 0) {

  if (!desc) return '';

  let t = desc.replace(/(?:truyen\s*full\s*(?:live|vn)?|truyenfulllive|truyenfullvn|truyenfull)[,\s.;:-]*/gi, '\n');

  t = t.replace(/([^\n])\s*(Số\s+chương|Dịch\s*giả|Dịch|Biên\s*tập|Biên|Nguồn|Tác\s+giả|Thể\s+loại|Trạng\s+thái)\s*:/gi, '$1\n$2:');

  t = t.replace(/[•·]\s*/g, '\n');

  t = t.replace(/(dịch\s*(?:giả)?\s*:[^\n\r]+?)([A-ZÀÁẢÃẠĂẮẰẲẴẶÂẤẦẨẪẬĐÈÉẺẼẸÊẾỀỂỄỆÌÍỈĨỊÒÓỎÕỌÔỐỒỔỖỘƠỚỜỞỠỢÙÚỦŨỤƯỨỪỬỮỰỲÝỶỸỴ][a-zàáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđ]{2,}\s+[a-zàáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđ]+)/gi, '$1\n$2');

  const lines = t.split(/\r?\n/).map(l => l.trim()).filter(Boolean);

  const body = lines.filter(l => !/^(?:số\s+chương|dịch\s*giả|dịch|nhóm\s*dịch|biên\s*tập|biên|nguồn|tác\s+giả|thể\s+loại|trạng\s+thái)\s*:/i.test(l));

  let text = (body.length ? body.join(' ') : lines.join(' '))

    .replace(/<[^>]*>/g, ' ')

    .replace(/\s+/g, ' ')

    .trim();

  return text;

}



function formatCompactNumber(num) {

  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';

  if (num >= 1000) return (num / 1000).toFixed(1) + 'K';

  return String(num);

}



function escapeHtml(str) {

  if (str === null || str === undefined) return '';

  return String(str)

    .replace(/&/g, '&amp;')

    .replace(/</g, '&lt;')

    .replace(/>/g, '&gt;')

    .replace(/"/g, '&quot;')

    .replace(/'/g, '&#39;');

}



// =========================================================================

// AUTHENTICATION SYSTEM (WEBNOVEL.VN STYLE)

// =========================================================================

function renderUserBadge() {

  const container = document.getElementById('user-auth-container');

  if (!container) return;



  if (state.currentUser && state.currentUser.username) {

    container.innerHTML = `

      <div class="user-profile-badge" style="display:flex; align-items:center; gap:8px; padding: 6px 14px; border-radius: 99px; background: var(--bg-soft); border: 1px solid var(--border); cursor:pointer; position:relative;">

        <i class="fa-solid fa-circle-user" style="color: var(--brand); font-size: 18px;"></i>

        <span style="font-weight: 700; font-size: 13px; color: var(--ink);">${escapeHtml(state.currentUser.username)}</span>

        <button onclick="handleLogout()" title="Đăng xuất" style="background:none; border:none; color: var(--ink-2); cursor:pointer; padding: 2px 4px; font-size: 13px; margin-left: 4px;">

          <i class="fa-solid fa-arrow-right-from-bracket"></i>

        </button>

      </div>

    `;

  } else {

    container.innerHTML = `

      <button class="btn btn--outline" id="btn-open-auth" style="padding: 7px 16px; border-radius: 99px; font-size: 13px; font-weight: 600;">

        <i class="fa-solid fa-user"></i>

        <span>Tài Khoản</span>

      </button>

    `;

    const btn = document.getElementById('btn-open-auth');

    if (btn) btn.addEventListener('click', () => openAuthModal(false));

  }

}



function initAuthSystem() {

  renderUserBadge();



  const modal = document.getElementById('auth-modal');

  const btnClose = document.getElementById('btn-close-auth');

  const tabLogin = document.getElementById('tab-login');

  const tabRegister = document.getElementById('tab-register');

  const formLogin = document.getElementById('form-auth-login');

  const formRegister = document.getElementById('form-auth-register');



  if (btnClose) btnClose.addEventListener('click', closeAuthModal);

  if (modal) {

    modal.addEventListener('click', (e) => {

      if (e.target === modal) closeAuthModal();

    });

  }



  if (tabLogin) {

    tabLogin.addEventListener('click', () => {

      tabLogin.classList.add('active');

      if (tabRegister) tabRegister.classList.remove('active');

      if (formLogin) formLogin.style.display = 'block';

      if (formRegister) formRegister.style.display = 'none';

      const title = document.getElementById('auth-modal-title');

      if (title) title.textContent = 'Đăng Nhập Tài Khoản';

    });

  }



  if (tabRegister) {

    tabRegister.addEventListener('click', () => {

      tabRegister.classList.add('active');

      if (tabLogin) tabLogin.classList.remove('active');

      if (formRegister) formRegister.style.display = 'block';

      if (formLogin) formLogin.style.display = 'none';

      const title = document.getElementById('auth-modal-title');

      if (title) title.textContent = 'Tạo Tài Khoản Mới';

    });

  }



  if (formLogin) {

    formLogin.addEventListener('submit', async (e) => {

      e.preventDefault();

      const login = document.getElementById('login-username').value.trim();

      const password = document.getElementById('login-password').value;

      const msgBox = document.getElementById('login-msg');



      if (msgBox) {

        msgBox.className = 'auth-msg-box';

        msgBox.textContent = 'Đang kiểm tra thông tin...';

      }



      try {

        const res = await fetch(`${API_BASE}/auth/login`, {

          method: 'POST',

          headers: { 'Content-Type': 'application/json' },

          body: JSON.stringify({ login, password })

        });

        const data = await res.json();

        if (!res.ok) throw new Error(data.error || 'Đăng nhập thất bại.');



        state.currentUser = data.user;

        localStorage.setItem(STORAGE_KEYS.AUTH_USER, JSON.stringify(data.user));

        if (data.token) localStorage.setItem(STORAGE_KEYS.AUTH_TOKEN, data.token);



        if (msgBox) {

          msgBox.className = 'auth-msg-box success';

          msgBox.textContent = data.message || 'Đăng nhập thành công!';

        }

        renderUserBadge();



        setTimeout(() => closeAuthModal(), 800);

      } catch (err) {

        if (msgBox) {

          msgBox.className = 'auth-msg-box error';

          msgBox.textContent = err.message;

        }

      }

    });

  }



  if (formRegister) {

    formRegister.addEventListener('submit', async (e) => {

      e.preventDefault();

      const username = document.getElementById('reg-username').value.trim();

      const email = document.getElementById('reg-email').value.trim();

      const password = document.getElementById('reg-password').value;

      const msgBox = document.getElementById('reg-msg');



      if (msgBox) {

        msgBox.className = 'auth-msg-box';

        msgBox.textContent = 'Đang khởi tạo tài khoản...';

      }



      try {

        const res = await fetch(`${API_BASE}/auth/register`, {

          method: 'POST',

          headers: { 'Content-Type': 'application/json' },

          body: JSON.stringify({ username, email, password })

        });

        const data = await res.json();

        if (!res.ok) throw new Error(data.error || 'Đăng ký thất bại.');



        state.currentUser = data.user;

        localStorage.setItem(STORAGE_KEYS.AUTH_USER, JSON.stringify(data.user));

        if (data.token) localStorage.setItem(STORAGE_KEYS.AUTH_TOKEN, data.token);



        if (msgBox) {

          msgBox.className = 'auth-msg-box success';

          msgBox.textContent = data.message || 'Đăng ký thành công!';

        }

        renderUserBadge();



        setTimeout(() => closeAuthModal(), 1000);

      } catch (err) {

        if (msgBox) {

          msgBox.className = 'auth-msg-box error';

          msgBox.textContent = err.message;

        }

      }

    });

  }

}



function openAuthModal(isRegister = false) {

  const modal = document.getElementById('auth-modal');

  if (!modal) return;

  modal.style.display = 'flex';



  const tabLogin = document.getElementById('tab-login');

  const tabRegister = document.getElementById('tab-register');

  if (isRegister) {

    if (tabRegister) tabRegister.click();

  } else {

    if (tabLogin) tabLogin.click();

  }

}



function closeAuthModal() {

  const modal = document.getElementById('auth-modal');

  if (modal) modal.style.display = 'none';

}



window.handleLogout = function() {

  localStorage.removeItem(STORAGE_KEYS.AUTH_USER);

  localStorage.removeItem(STORAGE_KEYS.AUTH_TOKEN);

  state.currentUser = null;

  renderUserBadge();

};



async function grantCultivatorExp(novelSlug, chapterNumber) {

  if (!state.currentUser || !state.currentUser.id) return;



  try {

    const res = await fetch(`${API_BASE}/user/read-chapter`, {

      method: 'POST',

      headers: { 'Content-Type': 'application/json' },

      body: JSON.stringify({

        user_id: state.currentUser.id,

        novel_slug: novelSlug,

        chapter_no: chapterNumber

      })

    });

    if (!res.ok) return;

    const data = await res.json();

    if (data.user) {

      const prevRealm = state.currentUser.realm;

      state.currentUser = data.user;

      localStorage.setItem(STORAGE_KEYS.AUTH_USER, JSON.stringify(data.user));

      renderUserBadge();



      // Notify on breakthrough

      if (prevRealm && data.user.realm && prevRealm !== data.user.realm) {

        alert(`⚡ ĐẠO CƠ ĐỘT PHÁ! Chúc mừng Đạo Hữu thăng tiến cảnh giới: 【${data.user.realm}】!`);

      }

    }

  } catch (err) {

    console.warn('Could not update exp:', err);

  }

}





// =========================================================================

// LIGHT / DARK THEME TOGGLE SYSTEM (KHOTRUYENCHU)

// =========================================================================

function initThemeMode() {

  const saved = localStorage.getItem('khotruyenchu_theme') || 'light';

  applySiteTheme(saved);



  const toggleBtn = document.getElementById('btn-theme-toggle');

  if (toggleBtn) {

    toggleBtn.addEventListener('click', () => {

      const current = document.documentElement.getAttribute('data-theme') || 'light';

      const next = current === 'dark' ? 'light' : 'dark';

      applySiteTheme(next);

      localStorage.setItem('khotruyenchu_theme', next);

    });

  }

}



function applySiteTheme(theme) {

  if (theme === 'dark') {

    document.documentElement.setAttribute('data-theme', 'dark');

  } else {

    document.documentElement.removeAttribute('data-theme');

    document.documentElement.setAttribute('data-theme', 'light');

  }

  const toggleBtn = document.getElementById('btn-theme-toggle');

  if (toggleBtn) {

    toggleBtn.innerHTML = theme === 'dark' ? '<i class="fa-solid fa-sun"></i>' : '<i class="fa-solid fa-moon"></i>';

    toggleBtn.title = theme === 'dark' ? 'Chuyển sang giao diện Sáng' : 'Chuyển sang giao diện Tối';

  }

}







// =========================================================================

// SEO & STRUCTURED DATA HELPERS

// =========================================================================

function updatePageSEO({ title, description, keywords, url, image, type = 'website', jsonLd = null }) {

  if (title) {

    document.title = title;

    const ogTitle = document.querySelector('meta[property="og:title"]');

    if (ogTitle) ogTitle.setAttribute('content', title);

    const twTitle = document.querySelector('meta[name="twitter:title"]');

    if (twTitle) twTitle.setAttribute('content', title);

  }



  if (description) {

    let metaDesc = document.querySelector('meta[name="description"]');

    if (!metaDesc) {

      metaDesc = document.createElement('meta');

      metaDesc.name = 'description';

      document.head.appendChild(metaDesc);

    }

    metaDesc.setAttribute('content', description);



    const ogDesc = document.querySelector('meta[property="og:description"]');

    if (ogDesc) ogDesc.setAttribute('content', description);

    const twDesc = document.querySelector('meta[name="twitter:description"]');

    if (twDesc) twDesc.setAttribute('content', description);

  }



  if (keywords) {

    let metaKw = document.querySelector('meta[name="keywords"]');

    if (!metaKw) {

      metaKw = document.createElement('meta');

      metaKw.name = 'keywords';

      document.head.appendChild(metaKw);

    }

    metaKw.setAttribute('content', keywords);

  }



  if (url) {

    const canonical = document.querySelector('link[rel="canonical"]');

    if (canonical) canonical.setAttribute('href', url);

    const ogUrl = document.querySelector('meta[property="og:url"]');

    if (ogUrl) ogUrl.setAttribute('content', url);

  }



  if (image) {

    const ogImg = document.querySelector('meta[property="og:image"]');

    if (ogImg) ogImg.setAttribute('content', image);

    const twImg = document.querySelector('meta[name="twitter:image"]');

    if (twImg) twImg.setAttribute('content', image);

  }



  if (jsonLd) {

    let scriptEl = document.getElementById('dynamic-jsonld');

    if (!scriptEl) {

      scriptEl = document.createElement('script');

      scriptEl.id = 'dynamic-jsonld';

      scriptEl.type = 'application/ld+json';

      document.head.appendChild(scriptEl);

    }

    scriptEl.textContent = JSON.stringify(jsonLd);

  }

}





// =========================================================================

// SMART SHOPEE AFFILIATE REDIRECT (THROTTLED WITH COOLDOWN)

// =========================================================================

const SHOPEE_CONFIG = {

  enabled: true,

  cooldownMs: 2000, // Cấu hình giãn cách: tối thiểu 2 giây bấm một lần

  storageKey: 'khotruyenchu_last_shopee_redirect',

  openInNewTab: true, // Mở Shopee ở tab mới để không làm gián đoạn việc đọc truyện

  defaultUrl: 'https://shopee.vn/'

};



function triggerShopeeRedirect(e) {

  if (!SHOPEE_CONFIG.enabled) return;

  const now = Date.now();

  const lastRedirect = Number(sessionStorage.getItem(SHOPEE_CONFIG.storageKey) || 0);



  if (now - lastRedirect >= SHOPEE_CONFIG.cooldownMs) {

    sessionStorage.setItem(SHOPEE_CONFIG.storageKey, String(now));



    // Lấy link affiliate Shopee hiện có (từ API hoặc danh sách mặc định)

    const affList = (state.affiliates && state.affiliates.length) ? state.affiliates : (typeof DEFAULT_AFFILIATES !== 'undefined' ? DEFAULT_AFFILIATES : []);

    const randomAff = affList.length ? affList[Math.floor(Math.random() * affList.length)] : null;

    const targetUrl = (randomAff && randomAff.affiliate_url) ? randomAff.affiliate_url : SHOPEE_CONFIG.defaultUrl;



    try {

      if (SHOPEE_CONFIG.openInNewTab) {

        const win = window.open(targetUrl, '_blank');

        if (win) {

          // Trả focus lại cho trang đọc truyện để người dùng tiếp tục xem

          win.blur();

          window.focus();

        }

      } else {

        window.open(targetUrl, '_blank');

      }

    } catch (err) {

      console.warn('Shopee redirect popup notice:', err);

    }

  }

}



// Bắt sự kiện bấm vào bất kỳ truyện nào (thẻ truyện, nút đọc, ranking, chi tiết)

document.addEventListener('click', (e) => {

  const target = e.target.closest('a[href*="#/novel/"], a[href*="#/read/"], .catx-card, .ranking__card, .hero-slide-item, #btn-read-first, #btn-read-continue, .detail-chapter-link');

  if (target) {

    triggerShopeeRedirect(e);

  }

}, { passive: true });


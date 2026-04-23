(() => {
    // ── Config ──────────────────────────────────────────────────────────
    const CHANGE_TYPES = ['all', 'breaking', 'deprecated', 'new', 'changed', 'fixed'];

    const TYPE_LABELS = {
        breaking:   'Breaking',
        deprecated: 'Deprecated',
        new:        'New',
        changed:    'Changed',
        fixed:      'Fixed',
    };

    // ── State ────────────────────────────────────────────────────────────
    let activeFilter = 'all';

    // ── DOM refs ────────────────────────────────────────────────────────
    const filtersEl  = document.getElementById('changelogFilters');
    const feedEl     = document.getElementById('changelogFeed');
    const emptyEl    = document.getElementById('changelogEmpty');

    // ── Rendering helpers ────────────────────────────────────────────────

    function renderTypeBadge(type) {
        return `<span class="badge badge-${type}">${TYPE_LABELS[type] ?? type}</span>`;
    }

    function renderServiceTag(service) {
        return `<span class="service-tag">${service}</span>`;
    }

    function renderMigration(migration) {
        if (!migration || !migration.length) return '';
        const rows = migration.map(m => `
            <div class="migration-row">
                <span class="migration-before">${escHtml(m.before)}</span>
                <span class="migration-arrow">→</span>
                <span class="migration-after">${escHtml(m.after)}</span>
            </div>`).join('');
        return `<div class="migration-table">${rows}</div>`;
    }

    function renderNewEndpoints(endpoints) {
        if (!endpoints || !endpoints.length) return '';
        const items = endpoints.map(e => `<span class="new-endpoint">${escHtml(e)}</span>`).join('');
        return `<div class="new-endpoints">${items}</div>`;
    }

    function renderChange(change) {
        const div = document.createElement('div');
        div.className = 'change-entry';
        div.dataset.type = change.type;
        div.innerHTML = `
            <div class="change-left">
                ${renderTypeBadge(change.type)}
                ${renderServiceTag(change.service)}
            </div>
            <div class="change-body">
                <div class="change-summary">${escHtml(change.summary)}</div>
                <div class="change-description">${escHtml(change.description)}</div>
                ${renderMigration(change.migration)}
                ${renderNewEndpoints(change.endpoints)}
            </div>`;
        return div;
    }

    function renderReleaseGroup(release) {
        const group = document.createElement('div');
        group.className = 'release-group';
        group.dataset.date = release.date;

        group.innerHTML = `
            <div class="release-header">
                <span class="release-date">${escHtml(release.date)}</span>
                <div class="release-divider"></div>
            </div>`;

        release.changes.forEach(change => {
            group.appendChild(renderChange(change));
        });

        return group;
    }

    function render(data) {
        feedEl.innerHTML = '';
        data.forEach(release => {
            feedEl.appendChild(renderReleaseGroup(release));
        });
        applyFilter(activeFilter);
    }

    // ── Filtering ────────────────────────────────────────────────────────

    function applyFilter(type) {
        activeFilter = type;

        // Update button states
        document.querySelectorAll('.filter-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.type === type);
        });

        // Show/hide entries
        let anyVisible = false;
        document.querySelectorAll('.change-entry').forEach(entry => {
            const match = type === 'all' || entry.dataset.type === type;
            entry.classList.toggle('hidden', !match);
            if (match) anyVisible = true;
        });

        // Show/hide release-group headings when all their children are hidden
        document.querySelectorAll('.release-group').forEach(group => {
            const visible = group.querySelectorAll('.change-entry:not(.hidden)').length > 0;
            group.style.display = visible ? '' : 'none';
        });

        // Empty state
        emptyEl.classList.toggle('visible', !anyVisible);

        // Sync URL hash for deep-linking
        if (type === 'all') {
            history.replaceState(null, '', window.location.pathname);
        } else {
            history.replaceState(null, '', `#${type}`);
        }
    }

    // ── Filter bar ───────────────────────────────────────────────────────

    function renderFilters() {
        filtersEl.innerHTML = CHANGE_TYPES.map(type => `
            <button class="filter-btn${type === activeFilter ? ' active' : ''}"
                    data-type="${type}">
                ${type === 'all' ? 'All changes' : TYPE_LABELS[type]}
            </button>`).join('');

        filtersEl.addEventListener('click', e => {
            const btn = e.target.closest('.filter-btn');
            if (btn) applyFilter(btn.dataset.type);
        });
    }

    // ── Utility ──────────────────────────────────────────────────────────

    function escHtml(str) {
        return String(str ?? '')
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;');
    }

    // ── Bootstrap ────────────────────────────────────────────────────────

    async function init() {
        // Read active filter from URL hash (deep-link support)
        const hashType = window.location.hash.replace('#', '');
        if (CHANGE_TYPES.includes(hashType)) activeFilter = hashType;

        renderFilters();

        try {
            const res = await fetch('../assets/data/changelog.json');
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            const data = await res.json();
            render(data);
        } catch (err) {
            feedEl.innerHTML = `<div class="changelog-empty visible">
                Failed to load changelog data. Please try again later.
            </div>`;
            console.error('Changelog load error:', err);
        }
    }

    init();
})();

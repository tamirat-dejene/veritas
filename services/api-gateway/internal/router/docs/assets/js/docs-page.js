const searchBox = document.getElementById('searchBox');
const serviceSections = document.getElementById('serviceSections');
const serviceList = document.getElementById('serviceList');
const activeServiceName = document.getElementById('activeServiceName');
const headerActions = document.getElementById('headerActions');
const dashboardView = document.getElementById('dashboardView');
const serviceCards = document.getElementById('serviceCards');
const backToHome = document.getElementById('backToHome');

const serviceGroups = {
    go: ['auth', 'enterprise', 'payment', 'exam', 'candidate', 'notification'],
    python: ['proctoring', 'face', 'grading', 'reporting'],
    monitoring: ['monitoring'],
};

const serviceOrder = [...serviceGroups.go, ...serviceGroups.python, ...serviceGroups.monitoring];

const serviceDetails = {
    auth: { description: 'Identity and Access Management service handling authentication and authorization.', swagger: '/swagger/auth/index.html' },
    enterprise: { description: 'Enterprise management service for organizations and departments.', swagger: '/swagger/enterprise/index.html' },
    payment: { description: 'Payment processing service for transactions and subscriptions.', swagger: '/swagger/payment/index.html' },
    exam: { description: 'Core exam engine for creating and managing assessments.', swagger: '/swagger/exam/index.html' },
    candidate: { description: 'Service for managing candidates and their exam schedules.', swagger: '/swagger/candidate/index.html' },
    proctoring: { description: 'Remote proctoring service for monitoring exam integrity.', swagger: '/swagger/proctoring/index.html' },
    face: { description: 'Biometric face recognition service for candidate verification.', swagger: '/swagger/face/index.html' },
    grading: { description: 'Automated grading service for exam responses.', swagger: '/swagger/grading/index.html' },
    reporting: { description: 'Analytics and reporting service for exam results.', swagger: '/swagger/reporting/index.html' },
    monitoring: { description: 'Real-time system monitoring and alerting service.', swagger: '/swagger/monitoring/index.html' },
    notification: { description: 'Event-driven notification and mailing service for system alerts and user communications.', swagger: '' },
};

const loadedServices = new Set();

function escapeHtml(value) {
    return String(value)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
}

function registerEndpointItemComponent() {
    if (customElements.get('endpoint-item')) return;

    class EndpointItem extends HTMLElement {
        connectedCallback() {
            if (this.dataset.rendered === 'true') return;
            const method = (this.getAttribute('method') || 'GET').toUpperCase();
            const path = this.getAttribute('path') || '';
            const methodClass = method.toLowerCase();
            const isDeprecated = this.hasAttribute('deprecated');
            const badgesMarkup = this.innerHTML.trim();
            const deprecatedPill = isDeprecated
                ? `<span style="font-family:inherit;font-size:0.65rem;font-weight:700;text-transform:uppercase;letter-spacing:0.5px;color:#92400e;background:#fffbeb;border:1px solid #fde68a;padding:2px 7px;border-radius:4px;margin-left:8px;">Deprecated</span>`
                : '';
            this.innerHTML = `
        <div class="endpoint-item" style="${isDeprecated ? 'opacity:0.7;' : ''}">
          <span class="method ${escapeHtml(methodClass)}">${escapeHtml(method)}</span>
          <span class="endpoint-path" style="${isDeprecated ? 'text-decoration:line-through;text-decoration-color:#d1d5db;' : ''}">${escapeHtml(path)}${deprecatedPill}</span>
          <div class="access-badges">
            ${badgesMarkup}
          </div>
          <button class="copy-btn" data-path="${escapeHtml(path)}">
            <svg class="copy-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z">
              </path>
            </svg>
            <span class="copy-text">Copy</span>
          </button>
        </div>
      `;
            this.dataset.rendered = 'true';
        }
    }
    customElements.define('endpoint-item', EndpointItem);

}

async function loadService(serviceName) {
    if (loadedServices.has(serviceName)) return;
    const basePath = serviceSections.dataset.sectionBase || 'assets/sections';
    const response = await fetch(`${basePath}/${serviceName}.html`);
    if (!response.ok) throw new Error(`Failed to load service: ${serviceName}`);
    const html = await response.text();
    const wrapper = document.createElement('div');
    wrapper.id = `section-${serviceName}`;
    wrapper.className = 'service-content-wrapper';
    wrapper.innerHTML = html;
    wrapper.style.display = 'none';
    serviceSections.appendChild(wrapper);
    loadedServices.add(serviceName);
}

function showDashboard() {
    dashboardView.style.display = 'block';
    document.querySelectorAll('.service-content-wrapper').forEach(w => w.style.display = 'none');
    document.querySelectorAll('.service-item').forEach(item => item.classList.remove('active'));
    activeServiceName.textContent = 'Dashboard';
    headerActions.innerHTML = '';
    window.location.hash = '';
    backToHome.style.visibility = 'hidden';
}

function switchService(serviceName) {
    dashboardView.style.display = 'none';
    document.querySelectorAll('.service-content-wrapper').forEach(w => w.style.display = 'none');
    const activeWrapper = document.getElementById(`section-${serviceName}`);
    if (activeWrapper) activeWrapper.style.display = 'block';

    document.querySelectorAll('.service-item').forEach(item => {
        item.classList.toggle('active', item.dataset.service === serviceName);
    });

    activeServiceName.textContent = serviceName.charAt(0).toUpperCase() + serviceName.slice(1) + ' Service';
    backToHome.style.visibility = 'visible';

    // Update Header Actions with Swagger Link
    const details = serviceDetails[serviceName];
    if (details && details.swagger) {
        headerActions.innerHTML = `
            <a href="${details.swagger}" target="_blank" rel="noopener noreferrer">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                    <polyline points="15 3 21 3 21 9"></polyline>
                    <line x1="10" y1="14" x2="21" y2="3"></line>
                </svg>
                Open Swagger UI
            </a>`;
    } else {
        headerActions.innerHTML = '';
    }

    window.location.hash = serviceName;
    if (searchBox) searchBox.value = '';
    applySearch('');
}

function renderSidebar() {
    if (!serviceList) return;
    const renderGroup = (label, services) => {
        const items = services.map(service => `
            <li class="service-item" data-service="${service}">
                <span class="service-name">${service.charAt(0).toUpperCase() + service.slice(1)}</span>
            </li>
        `).join('');

        return `
            <li class="service-group-label">${label}</li>
            ${items}
        `;
    };

    serviceList.innerHTML = [
        renderGroup('Go Services', serviceGroups.go),
        renderGroup('Python Services', serviceGroups.python),
        renderGroup('Monitoring', serviceGroups.monitoring),
    ].join('');

    serviceList.addEventListener('click', async (e) => {
        const item = e.target.closest('.service-item');
        if (!item) return;
        const serviceName = item.dataset.service;
        try {
            await loadService(serviceName);
            switchService(serviceName);
        } catch (error) {
            console.error(error);
        }
    });
}

function renderDashboard() {
    if (!serviceCards) return;
    serviceCards.innerHTML = serviceOrder.map(service => {
        const details = serviceDetails[service] || { description: 'Microservice for the Veritas platform.' };
        return `
            <div class="service-card" data-service="${service}">
                <h3>${service.charAt(0).toUpperCase() + service.slice(1)}</h3>
                <p>${details.description}</p>
            </div>
        `;
    }).join('');

    serviceCards.addEventListener('click', async (e) => {
        const card = e.target.closest('.service-card');
        if (!card) return;
        const serviceName = card.dataset.service;
        try {
            await loadService(serviceName);
            switchService(serviceName);
        } catch (error) {
            console.error(error);
        }
    });
}

function applySearch(searchTerm) {
    const wrappers = document.querySelectorAll('.service-content-wrapper');
    wrappers.forEach(wrapper => {
        const sections = wrapper.querySelectorAll('.section');
        sections.forEach(section => {
            const sectionTitle = section.querySelector('h2');
            const sectionName = sectionTitle ? sectionTitle.textContent.toLowerCase() : '';
            const endpoints = Array.from(section.querySelectorAll('.endpoint-path'))
                .map((endpoint) => endpoint.textContent.toLowerCase())
                .join(' ');
            const matches = sectionName.includes(searchTerm) || endpoints.includes(searchTerm);
            section.style.display = matches ? 'block' : 'none';
        });
    });
}

function setupSearch() {
    if (!searchBox) return;
    searchBox.addEventListener('input', (e) => {
        const searchTerm = e.target.value.toLowerCase();
        applySearch(searchTerm);
    });
}

function setupCopyHandler() {
    document.addEventListener('click', async (event) => {
        const copyButton = event.target.closest('.copy-btn');
        if (!copyButton) return;
        const path = copyButton.dataset.path;
        const copyText = copyButton.querySelector('.copy-text');
        try {
            await navigator.clipboard.writeText(path);
            copyButton.classList.add('copied');
            if (copyText) copyText.textContent = 'Copied!';
            setTimeout(() => {
                copyButton.classList.remove('copied');
                if (copyText) copyText.textContent = 'Copy';
            }, 2000);
        } catch (error) {
            console.error('Failed to copy:', error);
        }
    });
}

if (backToHome) {
    backToHome.addEventListener('click', showDashboard);
}

(async function initDocsPage() {
    registerEndpointItemComponent();
    renderSidebar();
    renderDashboard();
    setupSearch();
    setupCopyHandler();

    const hash = window.location.hash.replace('#', '');
    if (serviceOrder.includes(hash)) {
        await loadService(hash);
        switchService(hash);
    } else {
        showDashboard();
    }
})();

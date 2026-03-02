const searchBox = document.getElementById('searchBox');
const serviceSections = document.getElementById('serviceSections');

const serviceOrder = [
    'auth',
    'enterprise',
    'payment',
    'exam',
    'candidate',
    'proctoring',
    'face',
    'grading',
    'reporting',
    'monitoring',
];

function escapeHtml(value) {
    return String(value)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
}

function registerEndpointItemComponent() {
    if (customElements.get('endpoint-item')) {
        return;
    }

    class EndpointItem extends HTMLElement {
        connectedCallback() {
            if (this.dataset.rendered === 'true') {
                return;
            }

            const method = (this.getAttribute('method') || 'GET').toUpperCase();
            const path = this.getAttribute('path') || '';
            const methodClass = method.toLowerCase();
            const badgesMarkup = this.innerHTML.trim();

            this.innerHTML = `
        <div class="endpoint-item">
          <span class="method ${escapeHtml(methodClass)}">${escapeHtml(method)}</span>
          <span class="endpoint-path">${escapeHtml(path)}</span>
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

async function loadServiceSections() {
    if (!serviceSections) {
        return;
    }

    const basePath = serviceSections.dataset.sectionBase || 'assets/sections';
    const sectionContents = await Promise.all(
        serviceOrder.map(async (serviceName) => {
            const response = await fetch(`${basePath}/${serviceName}.html`);
            if (!response.ok) {
                throw new Error(`Failed to load section: ${serviceName}`);
            }
            return response.text();
        })
    );

    serviceSections.innerHTML = sectionContents.join('\n');
}

function setupSearch() {
    if (!searchBox) {
        return;
    }

    searchBox.addEventListener('input', (e) => {
        const searchTerm = e.target.value.toLowerCase();
        const sections = document.querySelectorAll('.section');

        sections.forEach(section => {
            const sectionTitle = section.querySelector('h2');
            if (!sectionTitle) {
                section.style.display = 'none';
                return;
            }

            const sectionName = sectionTitle.textContent.toLowerCase();
            const endpoints = Array.from(section.querySelectorAll('.endpoint-path'))
                .map((endpoint) => endpoint.textContent.toLowerCase())
                .join(' ');

            const matches = sectionName.includes(searchTerm) || endpoints.includes(searchTerm);
            section.style.display = matches ? 'block' : 'none';
        });
    });
}

function setupCopyHandler() {
    document.addEventListener('click', async (event) => {
        const copyButton = event.target.closest('.copy-btn');
        if (!copyButton) {
            return;
        }

        const path = copyButton.dataset.path;
        const copyText = copyButton.querySelector('.copy-text');

        try {
            await navigator.clipboard.writeText(path);
            copyButton.classList.add('copied');
            if (copyText) {
                copyText.textContent = 'Copied!';
            }

            setTimeout(() => {
                copyButton.classList.remove('copied');
                if (copyText) {
                    copyText.textContent = 'Copy';
                }
            }, 2000);
        } catch (error) {
            console.error('Failed to copy:', error);
        }
    });
}

(async function initDocsPage() {
    registerEndpointItemComponent();

    try {
        await loadServiceSections();
    } catch (error) {
        console.error(error);
    }

    setupSearch();
    setupCopyHandler();
})();

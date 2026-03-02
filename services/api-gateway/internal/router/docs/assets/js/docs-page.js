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
  try {
    await loadServiceSections();
  } catch (error) {
    console.error(error);
  }

  setupSearch();
  setupCopyHandler();
})();

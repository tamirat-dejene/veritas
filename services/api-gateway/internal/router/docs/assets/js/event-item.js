/**
 * EventItem Component
 * Documents Kafka/Event-driven messages
 */
class EventItem extends HTMLElement {
    connectedCallback() {
        const topic = this.getAttribute('topic') || 'unknown.topic';
        const description = this.innerHTML.trim();
        
        this.innerHTML = `
            <div class="event-item">
                <div class="event-topic-wrapper">
                    <svg class="event-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M4 11a9 9 0 0 1 9 9"></path>
                        <path d="M4 4a16 16 0 0 1 16 16"></path>
                        <circle cx="5" cy="19" r="1"></circle>
                    </svg>
                    <code class="event-topic">${this.escapeHtml(topic)}</code>
                </div>
                <p class="event-description">${description}</p>
            </div>
        `;
    }

    escapeHtml(value) {
        return String(value)
            .replaceAll('&', '&amp;')
            .replaceAll('<', '&lt;')
            .replaceAll('>', '&gt;')
            .replaceAll('"', '&quot;')
            .replaceAll("'", '&#39;');
    }
}

if (!customElements.get('event-item')) {
    customElements.define('event-item', EventItem);
}

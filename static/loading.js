// Loading States Manager - Premium Edition
// Handles skeleton screens, progress indicators, and smooth transitions

(function() {
    'use strict';

    // Configuration
    const LOADING_MESSAGES = [
        "Fetching your leagues...",
        "Analyzing rosters...",
        "Comparing tiers...",
        "Calculating win probabilities...",
        "Finding free agents...",
        "Loading dynasty values...",
        "Almost ready..."
    ];

    const LOADING_STEPS = [
        { id: 1, text: "Connecting to Sleeper API", icon: "🔗" },
        { id: 2, text: "Loading your leagues", icon: "🏈" },
        { id: 3, text: "Fetching player data", icon: "👥" },
        { id: 4, text: "Analyzing tiers", icon: "📊" },
        { id: 5, text: "Finalizing results", icon: "✨" }
    ];

    class LoadingManager {
        constructor() {
            this.currentMessage = 0;
            this.currentStep = 0;
            this.messageInterval = null;
            this.stepInterval = null;
        }

        // Show skeleton loading for league cards
        showLeagueSkeletons(container, count = 3) {
            container.innerHTML = '';

            for (let i = 0; i < count; i++) {
                const skeleton = this.createLeagueSkeleton();
                container.appendChild(skeleton);
            }
        }

        createLeagueSkeleton() {
            const card = document.createElement('div');
            card.className = 'league-card-skeleton';

            card.innerHTML = `
                <div class="skeleton-header">
                    <div class="skeleton skeleton-title"></div>
                    <div class="skeleton skeleton-badge"></div>
                </div>
                <div class="skeleton-table">
                    ${this.createSkeletonRows(8)}
                </div>
            `;

            return card;
        }

        createSkeletonRows(count) {
            let rows = '';
            for (let i = 0; i < count; i++) {
                rows += `
                    <div class="skeleton-table-row">
                        <div class="skeleton skeleton-cell"></div>
                        <div class="skeleton skeleton-cell"></div>
                        <div class="skeleton skeleton-cell"></div>
                        <div class="skeleton skeleton-cell"></div>
                    </div>
                `;
            }
            return rows;
        }

        // Show spinner with rotating messages
        showSpinnerLoading(container) {
            container.innerHTML = `
                <div class="loading-spinner-container">
                    <div class="loading-spinner"></div>
                    <div class="loading-text" id="loading-message">${LOADING_MESSAGES[0]}</div>
                    <div class="loading-subtext">This usually takes a few seconds</div>
                </div>
            `;

            this.currentMessage = 0;
            this.messageInterval = setInterval(() => {
                this.currentMessage = (this.currentMessage + 1) % LOADING_MESSAGES.length;
                const messageEl = document.getElementById('loading-message');
                if (messageEl) {
                    messageEl.style.opacity = '0';
                    setTimeout(() => {
                        messageEl.textContent = LOADING_MESSAGES[this.currentMessage];
                        messageEl.style.opacity = '1';
                    }, 200);
                }
            }, 2500);
        }

        // Show progress bar with steps
        showProgressLoading(container) {
            container.innerHTML = `
                <div class="progress-loading-container">
                    <div class="loading-text">Loading Your Fantasy Data</div>
                    <div class="progress-bar-bg">
                        <div class="progress-bar-fill" id="progress-bar" style="width: 0%"></div>
                    </div>
                    <div class="progress-steps" id="progress-steps">
                        ${LOADING_STEPS.map((step, index) => `
                            <div class="progress-step ${index === 0 ? 'active' : ''}" id="step-${step.id}">
                                <div class="progress-step-icon">${step.icon}</div>
                                <div class="progress-step-text">${step.text}</div>
                            </div>
                        `).join('')}
                    </div>
                </div>
            `;

            this.currentStep = 0;
            this.animateProgress();
        }

        animateProgress() {
            const progressBar = document.getElementById('progress-bar');
            const totalSteps = LOADING_STEPS.length;
            const stepDuration = 1200; // ms per step

            const advanceStep = () => {
                if (this.currentStep < totalSteps) {
                    // Update progress bar
                    const progress = ((this.currentStep + 1) / totalSteps) * 100;
                    if (progressBar) {
                        progressBar.style.width = `${progress}%`;
                    }

                    // Update step states
                    const prevStepEl = document.getElementById(`step-${LOADING_STEPS[this.currentStep].id}`);
                    if (prevStepEl) {
                        prevStepEl.classList.remove('active');
                        prevStepEl.classList.add('complete');
                    }

                    this.currentStep++;

                    if (this.currentStep < totalSteps) {
                        const nextStepEl = document.getElementById(`step-${LOADING_STEPS[this.currentStep].id}`);
                        if (nextStepEl) {
                            nextStepEl.classList.add('active');
                        }
                    }
                }
            };

            // Initial delay then advance steps
            setTimeout(() => {
                this.stepInterval = setInterval(advanceStep, stepDuration);
            }, 500);
        }

        // Show dot loader (simple/minimal)
        showDotLoader(container) {
            container.innerHTML = `
                <div class="dot-loader">
                    <div class="dot-loader-dot"></div>
                    <div class="dot-loader-dot"></div>
                    <div class="dot-loader-dot"></div>
                </div>
            `;
        }

        // Show success state
        showSuccess(container, message = "Loaded successfully!") {
            this.cleanup();

            container.innerHTML = `
                <div class="loading-spinner-container content-reveal">
                    <div class="success-checkmark"></div>
                    <div class="loading-text" style="color: #22c55e;">${message}</div>
                </div>
            `;

            // Auto-hide after animation
            setTimeout(() => {
                const successEl = container.querySelector('.loading-spinner-container');
                if (successEl) {
                    successEl.style.opacity = '0';
                    successEl.style.transform = 'scale(0.9)';
                    setTimeout(() => successEl.remove(), 300);
                }
            }, 1500);
        }

        // Show error state
        showError(container, message = "Something went wrong") {
            this.cleanup();

            container.innerHTML = `
                <div class="loading-spinner-container content-reveal">
                    <div class="error-icon"></div>
                    <div class="loading-text" style="color: #ef4444;">${message}</div>
                    <div class="loading-subtext">Please try again or check your connection</div>
                </div>
            `;
        }

        // Clean up intervals and timers
        cleanup() {
            if (this.messageInterval) {
                clearInterval(this.messageInterval);
                this.messageInterval = null;
            }
            if (this.stepInterval) {
                clearInterval(this.stepInterval);
                this.stepInterval = null;
            }
        }

        // Reveal content with staggered animation
        revealContent(elements, delay = 100) {
            elements.forEach((el, index) => {
                el.classList.add('content-reveal');
                el.style.animationDelay = `${index * delay}ms`;
            });
        }

        // Add glow effect to elements
        addGlowEffect(elements) {
            elements.forEach(el => {
                el.classList.add('glow-on-hover');
            });
        }
    }

    // Initialize and expose globally
    window.LoadingManager = new LoadingManager();

    // Enhance form submissions with loading states
    function enhanceFormSubmit() {
        const form = document.getElementById('userform');
        const outputDiv = document.getElementById('output');

        if (form && outputDiv) {
            form.addEventListener('submit', (e) => {
                // Show loading state
                window.LoadingManager.showProgressLoading(outputDiv);

                // Scroll to output
                outputDiv.scrollIntoView({ behavior: 'smooth', block: 'center' });
            });
        }
    }

    // Enhance league tabs with smoother transitions
    function enhanceLeagueTabs() {
        const tabs = document.querySelectorAll('.league-tab');
        const contents = document.querySelectorAll('.league-content');

        tabs.forEach((tab, index) => {
            tab.addEventListener('click', () => {
                // Fade out current content
                const activeContent = document.querySelector('.league-content[style*="display: block"]');
                if (activeContent) {
                    activeContent.style.opacity = '0';
                    setTimeout(() => {
                        activeContent.style.display = 'none';

                        // Fade in new content
                        const newContent = contents[index];
                        if (newContent) {
                            newContent.style.display = 'block';
                            newContent.style.opacity = '0';
                            setTimeout(() => {
                                newContent.style.opacity = '1';
                            }, 50);
                        }
                    }, 200);
                }
            });
        });
    }

    // Add interactive effects to buttons
    function enhanceButtons() {
        const buttons = document.querySelectorAll('button:not(.theme-option), .cta-button');
        buttons.forEach(btn => {
            if (!btn.classList.contains('interactive-btn')) {
                btn.classList.add('interactive-btn');
            }
        });
    }

    // Add glass card effects
    function enhanceCards() {
        const cards = document.querySelectorAll('.pretty-table, .winprob-row');
        cards.forEach(card => {
            // Add subtle glow on hover
            window.LoadingManager.addGlowEffect([card]);
        });
    }

    // Initialize enhancements on page load
    function init() {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => {
                enhanceFormSubmit();
                enhanceButtons();
                enhanceCards();

                // Reveal page content with animation
                const mainContent = document.querySelector('.container');
                if (mainContent) {
                    mainContent.style.opacity = '0';
                    setTimeout(() => {
                        mainContent.style.transition = 'opacity 0.6s ease-out';
                        mainContent.style.opacity = '1';
                    }, 100);
                }
            });
        } else {
            enhanceFormSubmit();
            enhanceButtons();
            enhanceCards();
        }
    }

    init();
})();

// Utility: Show loading on any container
function showLoading(containerId, type = 'progress') {
    const container = document.getElementById(containerId);
    if (!container) return;

    switch(type) {
        case 'skeleton':
            window.LoadingManager.showLeagueSkeletons(container, 3);
            break;
        case 'spinner':
            window.LoadingManager.showSpinnerLoading(container);
            break;
        case 'progress':
            window.LoadingManager.showProgressLoading(container);
            break;
        case 'dots':
            window.LoadingManager.showDotLoader(container);
            break;
    }
}

// Utility: Hide loading and show content
function hideLoading(containerId, showSuccess = false) {
    const container = document.getElementById(containerId);
    if (!container) return;

    if (showSuccess) {
        window.LoadingManager.showSuccess(container);
    } else {
        window.LoadingManager.cleanup();
    }
}

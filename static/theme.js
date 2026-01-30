// Theme switcher for SleeperPy
// Handles theme persistence and switching between default (dark blue), pure dark, and light modes

(function() {
    const THEME_KEY = 'sleeperpy-theme';
    const DEFAULT_THEME = ''; // Empty string = default CSS (dark blue)

    // Get saved theme from localStorage
    function getSavedTheme() {
        try {
            return localStorage.getItem(THEME_KEY) || DEFAULT_THEME;
        } catch (e) {
            return DEFAULT_THEME;
        }
    }

    // Save theme to localStorage
    function saveTheme(theme) {
        try {
            localStorage.setItem(THEME_KEY, theme);
        } catch (e) {
            console.warn('Could not save theme preference:', e);
        }
    }

    // Apply theme to document
    function applyTheme(theme) {
        if (theme === '') {
            document.documentElement.removeAttribute('data-theme');
        } else {
            document.documentElement.setAttribute('data-theme', theme);
        }
    }

    // Update active state of theme buttons
    function updateActiveButtons() {
        const currentTheme = getSavedTheme();
        document.querySelectorAll('.theme-option').forEach(btn => {
            const btnTheme = btn.dataset.theme;
            if (btnTheme === currentTheme) {
                btn.classList.add('active');
            } else {
                btn.classList.remove('active');
            }
        });
    }

    // Switch theme
    function switchTheme(theme) {
        applyTheme(theme);
        saveTheme(theme);
        updateActiveButtons();
    }

    // Initialize theme on page load
    function initTheme() {
        const savedTheme = getSavedTheme();
        applyTheme(savedTheme);

        // Wait for DOM to be ready before updating buttons
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', updateActiveButtons);
        } else {
            updateActiveButtons();
        }
    }

    // Expose switchTheme globally for button onclick handlers
    window.switchTheme = switchTheme;

    // Initialize immediately
    initTheme();
})();

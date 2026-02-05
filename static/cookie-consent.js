// ABOUTME: Cookie consent banner functionality
// ABOUTME: Handles cookie consent display, acceptance, and storage

(function() {
    'use strict';

    const CONSENT_COOKIE_NAME = 'sleeperpy_cookie_consent';
    const CONSENT_EXPIRY_DAYS = 365;

    function getCookie(name) {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) return parts.pop().split(';').shift();
        return null;
    }

    function setCookie(name, value, days) {
        const expires = new Date();
        expires.setTime(expires.getTime() + (days * 24 * 60 * 60 * 1000));
        document.cookie = `${name}=${value};expires=${expires.toUTCString()};path=/;SameSite=Lax`;
    }

    function hideConsentBanner() {
        const banner = document.getElementById('cookie-consent-banner');
        if (banner) {
            banner.style.opacity = '0';
            setTimeout(() => {
                banner.style.display = 'none';
            }, 300);
        }
    }

    function acceptCookies() {
        setCookie(CONSENT_COOKIE_NAME, 'accepted', CONSENT_EXPIRY_DAYS);
        hideConsentBanner();
    }

    function declineCookies() {
        setCookie(CONSENT_COOKIE_NAME, 'declined', CONSENT_EXPIRY_DAYS);
        hideConsentBanner();

        // Clear any existing cookies except the consent cookie
        document.cookie.split(";").forEach(function(c) {
            const cookieName = c.split("=")[0].trim();
            if (cookieName !== CONSENT_COOKIE_NAME) {
                document.cookie = cookieName + '=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/';
            }
        });
    }

    function showConsentBanner() {
        const consent = getCookie(CONSENT_COOKIE_NAME);

        // Don't show banner if user has already made a choice
        if (consent) {
            return;
        }

        // Create and show the banner
        const banner = document.createElement('div');
        banner.id = 'cookie-consent-banner';
        banner.className = 'cookie-consent-banner';
        banner.innerHTML = `
            <div class="cookie-consent-content">
                <div class="cookie-consent-text">
                    <strong>🍪 We use cookies</strong>
                    <p>We use cookies to remember your Sleeper username and preferences. We don't track you across other sites.
                    <a href="/privacy" target="_blank" rel="noopener">Learn more</a></p>
                </div>
                <div class="cookie-consent-actions">
                    <button id="cookie-accept" class="cookie-btn cookie-btn-accept">Accept</button>
                    <button id="cookie-decline" class="cookie-btn cookie-btn-decline">Decline</button>
                </div>
            </div>
        `;

        document.body.appendChild(banner);

        // Add event listeners
        document.getElementById('cookie-accept').addEventListener('click', acceptCookies);
        document.getElementById('cookie-decline').addEventListener('click', declineCookies);

        // Show banner with fade-in animation
        setTimeout(() => {
            banner.style.opacity = '1';
        }, 100);
    }

    // Initialize on page load
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', showConsentBanner);
    } else {
        showConsentBanner();
    }

    // Expose functions globally for inline onclick handlers if needed
    window.cookieConsent = {
        accept: acceptCookies,
        decline: declineCookies,
        hasConsent: function() {
            return getCookie(CONSENT_COOKIE_NAME) === 'accepted';
        }
    };
})();

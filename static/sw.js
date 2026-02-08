// SleeperPy Service Worker for PWA offline support
// Version 1.0.0

const CACHE_NAME = 'sleeperpy-v1';
const RUNTIME_CACHE = 'sleeperpy-runtime';

// Assets to cache immediately on install
const PRECACHE_ASSETS = [
  '/',
  '/static/main.css',
  '/static/dynasty.css',
  '/static/theme.css',
  '/static/loading.css',
  '/static/cookie-consent.css',
  '/static/manifest.json',
  '/static/favicon.svg'
];

// Install event - cache core assets
self.addEventListener('install', (event) => {
  console.log('[SW] Installing service worker...');
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        console.log('[SW] Precaching core assets');
        return cache.addAll(PRECACHE_ASSETS);
      })
      .then(() => self.skipWaiting())
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  console.log('[SW] Activating service worker...');
  event.waitUntil(
    caches.keys()
      .then((cacheNames) => {
        return Promise.all(
          cacheNames
            .filter((cacheName) => cacheName !== CACHE_NAME && cacheName !== RUNTIME_CACHE)
            .map((cacheName) => {
              console.log('[SW] Deleting old cache:', cacheName);
              return caches.delete(cacheName);
            })
        );
      })
      .then(() => self.clients.claim())
  );
});

// Fetch event - serve from cache when offline, cache API responses
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Skip caching for:
  // - Non-GET requests
  // - External API calls (we want fresh data)
  // - Admin routes
  // - Metrics endpoint
  if (
    request.method !== 'GET' ||
    url.pathname.startsWith('/admin') ||
    url.pathname === '/metrics' ||
    url.hostname.includes('sleeper.app') ||
    url.hostname.includes('borischen.co') ||
    url.hostname.includes('keeptradecut.com')
  ) {
    return event.respondWith(fetch(request));
  }

  // For static assets and pages, use cache-first strategy
  if (url.pathname.startsWith('/static/') || url.pathname === '/') {
    event.respondWith(
      caches.match(request)
        .then((cachedResponse) => {
          if (cachedResponse) {
            console.log('[SW] Serving from cache:', request.url);
            return cachedResponse;
          }

          // Not in cache, fetch and cache
          return fetch(request)
            .then((response) => {
              // Only cache successful responses
              if (response && response.status === 200) {
                const responseClone = response.clone();
                caches.open(CACHE_NAME).then((cache) => {
                  cache.put(request, responseClone);
                });
              }
              return response;
            })
            .catch(() => {
              // Return offline page if available
              return caches.match('/');
            });
        })
    );
    return;
  }

  // For other requests (lookup, etc.), use network-first strategy
  event.respondWith(
    fetch(request)
      .then((response) => {
        // Cache successful responses for offline fallback
        if (response && response.status === 200) {
          const responseClone = response.clone();
          caches.open(RUNTIME_CACHE).then((cache) => {
            cache.put(request, responseClone);
          });
        }
        return response;
      })
      .catch(() => {
        // Try to serve from cache if network fails
        return caches.match(request)
          .then((cachedResponse) => {
            if (cachedResponse) {
              console.log('[SW] Network failed, serving from cache:', request.url);
              return cachedResponse;
            }

            // Return a fallback offline page
            return caches.match('/');
          });
      })
  );
});

// Background sync for offline actions (future enhancement)
self.addEventListener('sync', (event) => {
  console.log('[SW] Background sync:', event.tag);
  // Future: handle offline actions here
});

// Push notifications (future enhancement)
self.addEventListener('push', (event) => {
  console.log('[SW] Push notification received');
  // Future: handle push notifications here
});

// Message handler for communication with main app
self.addEventListener('message', (event) => {
  console.log('[SW] Message received:', event.data);

  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }

  if (event.data && event.data.type === 'CLEAR_CACHE') {
    event.waitUntil(
      caches.keys().then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => caches.delete(cacheName))
        );
      }).then(() => {
        return self.clients.claim();
      })
    );
  }
});

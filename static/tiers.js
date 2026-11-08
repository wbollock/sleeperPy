// ABOUTME: JavaScript for tiers page - handles league navigation, favorites, and UI interactions
// ABOUTME: Manages league tabs/dropdown, mode switching, section toggling, and transaction filtering

function showLeagueTab(evt, tabId) {
    document.querySelectorAll('.league-content').forEach(div => div.style.display = 'none');
    document.getElementById(tabId).style.display = 'block';
    document.querySelectorAll('.league-tab').forEach(btn => btn.classList.remove('active'));
    evt.currentTarget.classList.add('active');

    // Remember last viewed league
    localStorage.setItem('sleeperpy_last_league', tabId);
}

function switchMode(evt, leagueIndex, mode) {
    const league = document.getElementById('league' + leagueIndex);

    // Update button states
    const btns = league.querySelectorAll('.mode-btn');
    btns.forEach(btn => btn.classList.remove('active'));
    evt.currentTarget.classList.add('active');

    // Update description
    const inseasonDesc = league.querySelector('.inseason-desc');
    const dynastyDesc = league.querySelector('.dynasty-desc');

    // Toggle content visibility
    const inseasonContent = league.querySelectorAll('.inseason-only');
    const dynastyContent = league.querySelectorAll('.dynasty-only');

    if (mode === 'inseason') {
        inseasonDesc.style.display = 'inline';
        dynastyDesc.style.display = 'none';
        inseasonContent.forEach(el => el.style.display = '');
        dynastyContent.forEach(el => el.style.display = 'none');
    } else {
        inseasonDesc.style.display = 'none';
        dynastyDesc.style.display = 'inline';
        inseasonContent.forEach(el => el.style.display = 'none');
        dynastyContent.forEach(el => el.style.display = '');
    }
}

// Cards worth seeing without an extra click on a brand-new visit. Matched
// against the toggleSection id prefix (before the trailing "-<leagueIndex>").
// Only applies the first time a browser loads this page -- see
// initCollapsibleCards -- so a user who deliberately collapses one of these
// later has that choice remembered like any other card.
const DEFAULT_EXPANDED_CARD_PREFIXES = ['season-plan', 'power-rankings', 'league-age', 'trade-targets', 'league-history'];

function toggleSection(sectionId) {
    const content = document.getElementById(sectionId + '-content');
    const icon = document.getElementById(sectionId + '-icon');

    const expanded = getExpandedSections() || [];
    if (content.style.display === 'none') {
        content.style.display = '';
        icon.textContent = '▼';
        if (!expanded.includes(sectionId)) expanded.push(sectionId);
    } else {
        content.style.display = 'none';
        icon.textContent = '▶';
        const idx = expanded.indexOf(sectionId);
        if (idx !== -1) expanded.splice(idx, 1);
    }
    localStorage.setItem('sleeperpy_expanded_sections', JSON.stringify(expanded));
}

// Returns null (not []) when the user has never toggled a card, so
// initCollapsibleCards can tell "brand-new visit" apart from "user collapsed
// everything" and only apply the default-expanded set in the former case.
function getExpandedSections() {
    const raw = localStorage.getItem('sleeperpy_expanded_sections');
    if (raw === null) return null;
    try {
        return JSON.parse(raw) || [];
    } catch (e) {
        return [];
    }
}

// Dynasty toolkit cards (draft picks, trade targets, league age, etc.) are
// collapsed by default -- there are ~20 of them and showing all expanded at
// once reads as a wall of unrelated panels. A handful of the most useful
// ones (see DEFAULT_EXPANDED_CARD_PREFIXES) start open on a user's first
// visit so the toolkit isn't a wall of collapsed headers with nothing to
// look at. After that first visit, a card's state is whatever the user last
// left it as (see toggleSection), including if they collapse a default one.
function initCollapsibleCards() {
    const stored = getExpandedSections();
    const isFirstVisit = stored === null;
    const expanded = stored || [];

    document.querySelectorAll('.toolkit-card.collapseable').forEach(card => {
        const header = card.querySelector('.card-header');
        const content = card.querySelector('.card-content');
        const icon = card.querySelector('.collapse-icon');
        if (!header || !content || !icon) return;

        const match = (header.getAttribute('onclick') || '').match(/toggleSection\('([^']+)'\)/);
        const sectionId = match && match[1];
        const isDefaultCard = sectionId && DEFAULT_EXPANDED_CARD_PREFIXES.some(prefix => sectionId.startsWith(prefix + '-'));

        let shouldExpand;
        if (isFirstVisit) {
            shouldExpand = !!isDefaultCard;
            if (shouldExpand) expanded.push(sectionId);
        } else {
            shouldExpand = sectionId && expanded.includes(sectionId);
        }

        content.style.display = shouldExpand ? '' : 'none';
        icon.textContent = shouldExpand ? '▼' : '▶';
    });

    if (isFirstVisit) {
        localStorage.setItem('sleeperpy_expanded_sections', JSON.stringify(expanded));
    }
}

function toggleLeagueDropdown() {
    const dropdown = document.getElementById('leagueDropdown');
    const searchInput = document.getElementById('leagueSearch');
    if (dropdown.style.display === 'none') {
        dropdown.style.display = 'block';
        searchInput.focus();
    } else {
        dropdown.style.display = 'none';
        searchInput.value = '';
        filterLeagues(); // Reset filter
    }
}

function selectLeague(evt, leagueId, leagueName) {
    // Hide all league contents
    document.querySelectorAll('.league-content').forEach(div => div.style.display = 'none');

    // Show selected league
    document.getElementById(leagueId).style.display = 'block';

    // Update active state for ALL instances of this league (favorites + main groups)
    document.querySelectorAll('.league-option').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll(`.league-option[data-league-id="${leagueId}"]`).forEach(btn => {
        btn.classList.add('active');
    });

    // Update dropdown button text
    document.getElementById('currentLeagueName').textContent = leagueName;

    // Close dropdown
    document.getElementById('leagueDropdown').style.display = 'none';
    document.getElementById('leagueSearch').value = '';
    filterLeagues(); // Reset filter

    // Remember last viewed league
    localStorage.setItem('sleeperpy_last_league', leagueId);
}

function filterLeagues() {
    const searchTerm = document.getElementById('leagueSearch').value.toLowerCase();
    const options = document.querySelectorAll('.league-option');
    const groups = document.querySelectorAll('.league-group');

    options.forEach(option => {
        const leagueName = option.dataset.leagueName.toLowerCase();
        if (leagueName.includes(searchTerm)) {
            option.style.display = 'flex';
        } else {
            option.style.display = 'none';
        }
    });

    // Hide groups if all options are hidden
    groups.forEach(group => {
        const groupOptions = group.querySelectorAll('.league-option');
        const visibleCount = Array.from(groupOptions).filter(opt => opt.style.display !== 'none').length;
        if (visibleCount === 0) {
            group.style.display = 'none';
        } else {
            group.style.display = 'block';
        }
    });
}

// Close dropdown when clicking outside
document.addEventListener('click', function(event) {
    const dropdown = document.getElementById('leagueDropdown');
    const btn = document.querySelector('.league-dropdown-btn');
    if (dropdown && btn && !dropdown.contains(event.target) && !btn.contains(event.target)) {
        dropdown.style.display = 'none';
    }
});

// League favorites functionality
function getFavorites() {
    const stored = localStorage.getItem('sleeperpy_favorites');
    return stored ? JSON.parse(stored) : [];
}

function saveFavorites(favorites) {
    localStorage.setItem('sleeperpy_favorites', JSON.stringify(favorites));
}

function toggleFavorite(event, leagueId, leagueName) {
    event.stopPropagation(); // Prevent league selection when clicking star

    const favorites = getFavorites();
    const index = favorites.findIndex(f => f.id === leagueId);
    const starIcon = event.target;

    if (index > -1) {
        // Remove from favorites
        favorites.splice(index, 1);
        starIcon.textContent = '☆';
        starIcon.title = 'Add to favorites';
    } else {
        // Add to favorites
        favorites.push({ id: leagueId, name: leagueName });
        starIcon.textContent = '★';
        starIcon.title = 'Remove from favorites';
    }

    saveFavorites(favorites);
    updateFavoritesGroup();
}

function updateFavoritesGroup() {
    const favorites = getFavorites();
    const favoritesGroup = document.getElementById('favorites-group');

    if (favorites.length === 0) {
        favoritesGroup.style.display = 'none';
        return;
    }

    favoritesGroup.style.display = 'block';

    // Clear existing favorites (keep header)
    const existingOptions = favoritesGroup.querySelectorAll('.league-option');
    existingOptions.forEach(opt => opt.remove());

    // Add each favorite
    favorites.forEach(fav => {
        const originalOption = document.querySelector(`.league-option[data-league-id="${fav.id}"]`);
        if (originalOption) {
            const clone = originalOption.cloneNode(true);
            // Update star to filled
            const star = clone.querySelector('.favorite-star');
            if (star) {
                star.textContent = '★';
                star.title = 'Remove from favorites';
            }
            favoritesGroup.appendChild(clone);
        }
    });
}

function initFavorites() {
    const favorites = getFavorites();

    // Update all star icons based on saved favorites
    favorites.forEach(fav => {
        const starIcon = document.querySelector(`.league-option[data-league-id="${fav.id}"] .favorite-star`);
        if (starIcon) {
            starIcon.textContent = '★';
            starIcon.title = 'Remove from favorites';
        }
    });

    // Populate favorites group
    updateFavoritesGroup();
}

// Initialize favorites on page load
if (document.getElementById('leagueDropdown')) {
    initFavorites();
}

// Transaction filtering
function filterTransactions(event, leagueIndex, filterType) {
    const transactionsList = document.getElementById('transactions-list-' + leagueIndex);
    const transactions = transactionsList.querySelectorAll('.transaction-item');
    const filterBtns = event.currentTarget.parentElement.querySelectorAll('.filter-btn');

    // Update active button state
    filterBtns.forEach(btn => btn.classList.remove('active'));
    event.currentTarget.classList.add('active');

    // Show/hide transactions based on filter
    transactions.forEach(transaction => {
        if (filterType === 'all') {
            transaction.style.display = '';
        } else {
            if (transaction.classList.contains('transaction-' + filterType)) {
                transaction.style.display = '';
            } else {
                transaction.style.display = 'none';
            }
        }
    });
}

// Restore last viewed league on page load
function restoreLastLeague() {
    const lastLeagueId = localStorage.getItem('sleeperpy_last_league');
    if (!lastLeagueId) return;

    // Check if the league exists on this page
    const leagueElement = document.getElementById(lastLeagueId);
    if (!leagueElement) return;

    // For dropdown view
    const dropdown = document.getElementById('leagueDropdown');
    if (dropdown) {
        const leagueOption = document.querySelector(`.league-option[data-league-id="${lastLeagueId}"]`);
        if (leagueOption) {
            const leagueName = leagueOption.dataset.leagueName;

            // Hide all leagues
            document.querySelectorAll('.league-content').forEach(div => div.style.display = 'none');

            // Show the last viewed league
            leagueElement.style.display = 'block';

            // Update dropdown button text
            document.getElementById('currentLeagueName').textContent = leagueName;

            // Update active state for ALL instances (favorites + main groups)
            document.querySelectorAll('.league-option').forEach(btn => btn.classList.remove('active'));
            document.querySelectorAll(`.league-option[data-league-id="${lastLeagueId}"]`).forEach(btn => {
                btn.classList.add('active');
            });
        }
    } else {
        // For tab view (<5 leagues)
        const tabButton = document.querySelector(`[onclick*="${lastLeagueId}"]`);
        if (tabButton) {
            tabButton.click();
        }
    }
}

// Player search within a league
function searchPlayers(leagueIndex) {
    const searchInput = document.getElementById('playerSearch' + leagueIndex);
    const term = searchInput.value.toLowerCase().trim();
    const league = document.getElementById('league' + leagueIndex);
    const rows = league.querySelectorAll('.pretty-table tbody tr');

    rows.forEach(function(row) {
        // Find the player name cell (second column typically)
        const cells = row.querySelectorAll('td');
        if (cells.length < 2) return;

        const playerName = cells[1].textContent.toLowerCase();
        if (term === '' || playerName.includes(term)) {
            row.style.display = '';
            row.classList.remove('search-dimmed');
        } else {
            row.classList.add('search-dimmed');
        }
    });
}

// Jump to a specific league via #league-<leagueID> (used by dashboard action
// links, which don't know the on-page tab index -- only the Sleeper league ID).
function selectLeagueFromHash() {
    if (!location.hash.startsWith('#league-')) return false;
    const anchor = document.querySelector(location.hash);
    const targetId = anchor && anchor.dataset.target;
    const targetEl = targetId && document.getElementById(targetId);
    if (!targetEl) return false;

    document.querySelectorAll('.league-content').forEach(div => div.style.display = 'none');
    targetEl.style.display = 'block';

    const leagueOption = document.querySelector(`.league-option[data-league-id="${targetId}"]`);
    if (leagueOption) {
        document.querySelectorAll('.league-option').forEach(btn => btn.classList.remove('active'));
        document.querySelectorAll(`.league-option[data-league-id="${targetId}"]`).forEach(btn => btn.classList.add('active'));
        const nameEl = document.getElementById('currentLeagueName');
        if (nameEl) nameEl.textContent = leagueOption.dataset.leagueName;
    }

    localStorage.setItem('sleeperpy_last_league', targetId);
    return true;
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    if (!selectLeagueFromHash()) {
        restoreLastLeague();
    }
    initCollapsibleCards();
});

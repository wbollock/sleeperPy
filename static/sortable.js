// Simple sort table JS for static tables (no dependencies)
function makeTableSortable(table) {
  const thead = table.querySelector('thead');
  if (!thead) return;

  const tbody = table.querySelector('tbody');
  if (!tbody) return;

  // Mark table as sortable to prevent re-initialization
  if (table.dataset.sortableInitialized) return;
  table.dataset.sortableInitialized = 'true';

  const ths = thead.querySelectorAll('th');

  ths.forEach((th, idx) => {
    th.style.cursor = 'pointer';

    th.addEventListener('click', () => {
      const rows = Array.from(tbody.querySelectorAll('tr'));
      const asc = !th.classList.contains('asc');

      // Clear all header sort states
      thead.querySelectorAll('th').forEach(oth => {
        oth.classList.remove('asc');
        oth.classList.remove('desc');
      });

      // Set current header sort state
      if (asc) {
        th.classList.add('asc');
        th.classList.remove('desc');
      } else {
        th.classList.add('desc');
        th.classList.remove('asc');
      }

      rows.sort((a, b) => {
        let t1 = a.children[idx]?.innerText.trim() || '';
        let t2 = b.children[idx]?.innerText.trim() || '';

        // Remove non-numeric characters for numeric comparison
        const n1 = parseFloat(t1.replace(/[^0-9.-]/g, ''));
        const n2 = parseFloat(t2.replace(/[^0-9.-]/g, ''));

        // If both are valid numbers, compare numerically
        if (!isNaN(n1) && !isNaN(n2)) {
          return asc ? (n1 - n2) : (n2 - n1);
        }

        // Otherwise compare as strings
        return asc ? t1.localeCompare(t2) : t2.localeCompare(t1);
      });

      // Re-append rows in sorted order
      rows.forEach(r => tbody.appendChild(r));
    });
  });
}

document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('.sortable-table').forEach(makeTableSortable);
});

document.body.addEventListener('htmx:afterSwap', function(evt) {
  document.querySelectorAll('.sortable-table').forEach(makeTableSortable);
});

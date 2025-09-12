// Simple sort table JS for static tables (no dependencies)
function makeTableSortable(table) {
  const ths = table.querySelectorAll('th');
  ths.forEach((th, idx) => {
    th.style.cursor = 'pointer';
    th.addEventListener('click', () => {
      const rows = Array.from(table.querySelectorAll('tr')).slice(1);
      const asc = th.classList.toggle('asc');
      ths.forEach((oth, i) => { if (i !== idx) oth.classList.remove('asc'); });
      rows.sort((a, b) => {
        let t1 = a.children[idx].innerText.trim();
        let t2 = b.children[idx].innerText.trim();
        if (!isNaN(t1) && !isNaN(t2)) { t1 = +t1; t2 = +t2; }
        return asc ? (t1 > t2 ? 1 : -1) : (t1 < t2 ? 1 : -1);
      });
      rows.forEach(r => table.appendChild(r));
    });
  });
}

document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('.sortable-table').forEach(makeTableSortable);
});

document.body.addEventListener('htmx:afterSwap', function(evt) {
  document.querySelectorAll('.sortable-table').forEach(makeTableSortable);
});

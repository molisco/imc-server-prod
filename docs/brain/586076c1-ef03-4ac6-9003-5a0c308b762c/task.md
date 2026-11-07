# Checklist: Target Store Selection Search & Premium Dropdown Filters

- [x] 1. Add Search capability to store selection interfaces in `commercials.html`
  - [x] Add search input to inline store checklist
  - [x] Implement inline checklist search JS filtering function
  - [x] Add search input to distribution modal
  - [x] Implement modal checklist search JS filtering function
- [x] 2. Convert Store listing filter in `commercials.html` to dropdown checklist with search
  - [x] Replace select filter with dropdown checkbox combobox markup
  - [x] Update JS to filter store checkboxes based on Matrix & Regional selections
  - [x] Update JS `applyFilters()` and `updateCategoryCounts()` to query store checkboxes
- [x] 3. Implement premium Matrix, Regional, and Filial dropdown filters with search on `stores.html`
  - [x] Rebuild filters panel HTML structure with 3 checkbox dropdown comboboxes & resources select
  - [x] Add `data-id`, `data-tenant-id`, and `data-regional-id` to store row element
  - [x] Implement search filtering logic for dropdown checklist elements
  - [x] Implement dependency cascading filter (matrix -> regional -> stores)
  - [x] Rewrite JS `filterTable()` to apply multi-selections
- [x] 4. Verify compilation, run automated tests, and manually check UI

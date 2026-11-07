# Add Search inside Distribution Engine and Advanced Filters

We will implement search capabilities inside the target store selection interfaces, convert the store filter in the commercials list to a premium checkbox dropdown with search, and create a premium filter system (Matriz, Regional, and Filial checkbox dropdowns with search options) in the Filiais (Lojas) table view.

## User Review Required

> [!NOTE]
> All new filter selectors will use the same visual look & feel as the existing premium company filters: dark themed dropdown comboboxes with checkable list elements, selection counts, and real-time search inputs inside the menus.

## Open Questions

No open questions. The requirements are fully detailed.

## Proposed Changes

### Commercials & Events Dashboard

We will update the distribution engine inputs and the store filter in the commercials listing.

---

#### [MODIFY] [commercials.html](file:///P:/imc-server/web/templates/commercials.html)

- **Inline Store Checklist Search**:
  - Add `<input type="text" id="inline-store-search"...>` right before the inline checklist container.
  - Implement `filterInlineStoresList(query)` to show/hide `.inline-store-checkbox-wrapper` and collapse or hide empty `.inline-tenant-group` blocks.
- **Distribution Modal Search**:
  - Add `<input type="text" id="modal-store-search"...>` inside the `distributionModal` body.
  - Implement `filterModalStoresList(query)` to show/hide `.store-checkbox-wrapper` and hide empty `.tenant-group` blocks.
- **Convert Store Filter to Combobox Dropdown**:
  - Replaced the select `#filter-store-select` with a dropdown menu `#filter-stores-dropdown-wrapper`.
  - Add checkbox list for stores (`.filter-store-cb`) and a search input (`#filter-stores-search`).
  - Update `updateFilterDropdowns()` to dynamically hide/reveal stores matching the active tenant/regional selections, and uncheck hidden ones.
  - Update `applyFilters()` and `updateCategoryCounts()` to query `.filter-store-cb:checked` array instead of the select element.

---

### Filiais (Lojas) Dashboard

We will rebuild the filters section and add search functions.

---

#### [MODIFY] [stores.html](file:///P:/imc-server/web/templates/stores.html)

- **Filter Panel Rebuild**:
  - Insert a custom advanced filters card containing:
    - **Matriz (Cliente)** dropdown checklist with search.
    - **Regional** dropdown checklist with search.
    - **Filial (Loja)** dropdown checklist with search.
    - **Recursos / Status** select dropdown.
  - Add data attributes `data-id`, `data-tenant-id`, and `data-regional-id` to the table's `store-row` rows.
- **Javascript Filter Engine**:
  - Implement dropdown search input filtering via `filterDropdownList()`.
  - Implement dropdown selection text sync: `updateFilterDropdownText()`, `updateFilterRegionalsDropdownText()`, and `updateFilterStoresDropdownText()`.
  - Implement dropdown dependencies cascade inside `updateFilterDropdowns()` (hide regionals/stores not in checked tenants).
  - Rewrite `filterTable()` to apply multi-matrix, multi-regional, multi-store, and text queries in combination, updating pagination at the end.

## Verification Plan

### Automated Tests
- Run `DATABASE_URL='postgres://neoroute:@@Mo1isc00@@@localhost:5432/imc_server?sslmode=disable' go test ./...` on the remote server `dreams`.

### Manual Verification
- Open `/dashboard/commercials` in a browser.
  - Check "Distribuir para Todas as Lojas automaticamente" off to reveal the inline checklist. Type in the search box; verify stores/matrices hide or show dynamically.
  - Click the `🎯` button on any commercial row to open the distribution modal. Type in the modal's search box and verify store lists filter properly.
  - Open the "Filiais (Lojas)" filter dropdown in the main filter panel. Verify that typing in its search box filters stores. Check multiple stores, and check that rows filter in real time.
- Open `/dashboard/stores` in a browser.
  - Verify the new filter card is displayed.
  - Filter by Matriz, Regional, and Filial. Verify that checking options filters the table.
  - Verify that typing in the search boxes of the dropdowns filters options in real time.
  - Verify pagination resets to page 1 and updates counts correctly when filters are changed.

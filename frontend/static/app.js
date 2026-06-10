(function () {
  const storageKey = "soc5_user";
  const themeKey = "soc5_theme";
  const user = readUser();
  const apiBase = String((window.SOC5_CONFIG && window.SOC5_CONFIG.API_BASE) || "").replace(/\/$/, "");
  const routeMap = {
    "/": "./index.html",
    "/dashboard": "./dashboard.html",
    "/outbound/lh-request": "./lh-request.html",
    "/outbound/lh-requests": "./lh-request.html",
    "/midmile/truck-request": "./truck-request.html",
    "/dock/officer": "./dock-officer.html",
    "/settings": "./settings.html",
  };
  let notificationAudioContext = null;
  let notificationAudioReady = false;
  let pendingNotificationChimes = 0;
  let audioUnlockToastShown = false;
  let workflowRefreshTimer = null;
  const state = {
    rows: [],
    clusters: [],
    page: 1,
    perPage: 12,
    sortKey: "request_date",
    sortDirection: "desc",
    inlineRequestOpen: false,
    inlineRequestSaving: false,
    lastOps: Number(sessionStorage.getItem("last_ops_count") || 0),
    lastMM: Number(sessionStorage.getItem("last_mm_count") || 0),
    lastDock: Number(sessionStorage.getItem("last_dock_count") || 0),
  };

  applyStoredTheme();

  document.addEventListener("DOMContentLoaded", () => {
    applyDensity();
    applyRoleVisibility();
    bindNavigation();
    bindTopbar();
    bindUserMenu();
    bindLogin();
    bindRequestsPage();
    bindSettingsPage();
    bindDashboard();
    bindNotifications();
    bindRequestDetailModal();
    bindWorkflowEvents();
    bindNotificationAudioUnlock();
    bindTruckLabelModal();
    updateStats();
    window.setInterval(updateStats, 5000);
  });

  function readUser() {
    try {
      return JSON.parse(localStorage.getItem(storageKey) || "null");
    } catch (_) {
      return null;
    }
  }

  function saveUser(nextUser) {
    localStorage.setItem(storageKey, JSON.stringify(nextUser));
  }

  function apiPath(path) {
    return apiBase + path;
  }

  function appRoute(path) {
    const value = String(path || "/dashboard");
    const match = value.match(/^([^?#]*)(.*)$/);
    const base = match ? match[1] : value;
    const suffix = match ? match[2] : "";
    return (routeMap[base] || value) + (routeMap[base] ? suffix : "");
  }

  function applyRoleVisibility() {
    const currentUser = readUser();
    const role = currentUser && currentUser.role ? currentUser.role : "";
    document.body.classList.add("role-ready");

    document.querySelectorAll("[data-role-link]").forEach((node) => {
      const roles = (node.getAttribute("data-role-link") || "").split(/\s+/);
      node.classList.toggle("is-role-hidden", role === "" || !roles.includes(role));
    });

    document.querySelectorAll("[data-user-name]").forEach((node) => {
      node.textContent = currentUser ? currentUser.name : "Guest";
    });
    document.querySelectorAll("[data-user-role]").forEach((node) => {
      node.textContent = currentUser ? roleLabel(currentUser.role) : "Not signed in";
    });
    document.querySelectorAll("[data-user-initials]").forEach((node) => {
      node.textContent = initials(currentUser ? currentUser.name : "Guest");
    });
    document.querySelectorAll("[data-current-user-name]").forEach((node) => {
      node.value = currentUser ? currentUser.name : "";
    });

    const loginButtons = document.querySelectorAll("[data-open-login]");
    loginButtons.forEach((button) => {
      button.classList.toggle("is-hidden", Boolean(currentUser));
    });

    document.querySelectorAll("[data-logout]").forEach((button) => {
      button.classList.toggle("is-hidden", !currentUser);
    });

    document.querySelectorAll(".topbar-user").forEach((node) => {
      node.classList.toggle("is-hidden", !currentUser);
    });
  }

  function bindNavigation() {
    document.querySelectorAll(".nav-group-button").forEach((button) => {
      button.addEventListener("click", () => {
        const group = button.closest(".nav-group");
        const items = group ? group.querySelector(".nav-group-items") : null;
        const expanded = button.getAttribute("aria-expanded") !== "false";
        button.setAttribute("aria-expanded", String(!expanded));
        if (items) {
          items.classList.toggle("is-hidden", expanded);
        }
      });
    });

    document.querySelectorAll("[data-logout]").forEach((button) => {
      button.addEventListener("click", () => {
        localStorage.removeItem(storageKey);
        location.href = appRoute("/");
      });
    });
  }

  function bindTopbar() {
    const sidebarButton = document.querySelector("[data-toggle-sidebar]");
    if (sidebarButton) {
      sidebarButton.addEventListener("click", () => {
        const collapsed = document.body.classList.toggle("sidebar-collapsed");
        localStorage.setItem("soc5_sidebar_collapsed", collapsed ? "1" : "0");
      });
    }

    if (localStorage.getItem("soc5_sidebar_collapsed") === "1") {
      document.body.classList.add("sidebar-collapsed");
    }

    document.querySelectorAll("[data-toggle-theme]").forEach((button) => {
      button.addEventListener("click", () => {
        const nextTheme = document.body.classList.contains("theme-dark") ? "light" : "dark";
        localStorage.setItem(themeKey, nextTheme);
        applyStoredTheme();
        toast(nextTheme === "light" ? "Light mode enabled" : "Dark mode enabled");
      });
    });

    document.querySelectorAll("[data-global-search]").forEach((input) => {
      input.addEventListener("input", debounce(async () => {
        const pageSearch = document.querySelector("[data-filter-search]");
        if (pageSearch) {
          pageSearch.value = input.value;
          state.page = 1;
          await fetchRequests();
          return;
        }

        if (document.querySelector("[data-activity-list]")) {
          await refreshActivity(input.value);
        }
      }, 220));
    });

    document.querySelectorAll("[data-density-toggle]").forEach((button) => {
      button.addEventListener("click", () => {
        const compact = !document.body.classList.contains("density-compact");
        localStorage.setItem("soc5_density", compact ? "compact" : "comfortable");
        applyDensity();
        toast(compact ? "Compact density enabled" : "Comfortable density enabled");
      });
    });
  }

  function bindUserMenu() {
    const menu = document.querySelector("[data-user-menu]");
    const button = document.querySelector("[data-toggle-user-menu]");
    const panel = document.querySelector("[data-user-menu-panel]");
    if (!menu || !button || !panel) {
      return;
    }

    button.addEventListener("click", () => {
      const hidden = panel.classList.toggle("is-hidden");
      button.setAttribute("aria-expanded", String(!hidden));
    });

    document.addEventListener("click", (event) => {
      if (!menu.contains(event.target)) {
        panel.classList.add("is-hidden");
        button.setAttribute("aria-expanded", "false");
      }
    });
  }

  function applyDensity() {
    document.body.classList.toggle("density-compact", localStorage.getItem("soc5_density") === "compact");
  }

  function applyStoredTheme() {
    const theme = localStorage.getItem(themeKey) || "light";
    const isDark = theme === "dark";
    document.documentElement.classList.toggle("theme-dark", isDark);
    document.body.classList.toggle("theme-dark", isDark);
    document.querySelectorAll("[data-toggle-theme]").forEach((button) => {
      const label = isDark ? "Switch to light mode" : "Switch to dark mode";
      button.setAttribute("title", label);
      button.setAttribute("aria-label", label);
    });
  }

  function bindLogin() {
    const modal = document.querySelector("[data-login-modal]");
    const form = document.querySelector("[data-login-form]");
    if (!modal || !form) {
      return;
    }

    if (user) {
      modal.classList.add("is-hidden");
    }

    document.querySelectorAll("[data-open-login]").forEach((button) => {
      button.addEventListener("click", () => openModal(modal));
    });
    document.querySelectorAll("[data-close-login]").forEach((button) => {
      button.addEventListener("click", () => closeModal(modal));
    });

    document.querySelectorAll("[data-login-tab]").forEach((button) => {
      button.addEventListener("click", () => {
        const type = button.getAttribute("data-login-tab");
        document.querySelectorAll("[data-login-tab]").forEach((tab) => tab.classList.toggle("is-active", tab === button));
        document.querySelector("[data-login-type]").value = type;
        document.querySelector("[data-fte-field]").classList.toggle("is-hidden", type !== "fte");
        document.querySelector("[data-backroom-field]").classList.toggle("is-hidden", type !== "backroom");
      });
    });

    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      form.classList.add("is-loading");
      setLoginError("");

      const payload = formToObject(form);
      try {
        const response = await fetch(apiPath("/api/login"), jsonOptions(payload));
        const body = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(body.error || body.message || "Invalid credentials");
        }

        saveUser(body);
        toast("Signed in as " + body.name);
        form.classList.remove("shake");
        form.classList.add("success");
        window.setTimeout(() => {
          location.href = appRoute(body.redirect || "/dashboard");
        }, 420);
      } catch (error) {
        setLoginError(error.message);
        form.classList.remove("shake");
        void form.offsetWidth;
        form.classList.add("shake");
      } finally {
        form.classList.remove("is-loading");
      }
    });
  }

  function bindRequestsPage() {
    const page = document.querySelector("[data-requests-page]");
    if (!page) {
      return;
    }

    applyRequestURLFilters();
    bindQuickFilters();
    bindColumnControls();
    applyColumnVisibility();
    fetchClusters();
    fetchRequests();

    const requestModal = document.querySelector("[data-request-modal]");
    const requestForm = document.querySelector("[data-request-form]");
    document.querySelectorAll("[data-open-request-modal]").forEach((button) => {
      button.addEventListener("click", () => openInlineRequestRow());
    });
    document.querySelectorAll("[data-close-request-modal]").forEach((button) => {
      button.addEventListener("click", () => closeModal(requestModal));
    });

    if (requestForm) {
      const clusterSelect = requestForm.querySelector("[data-cluster-select]");
      if (clusterSelect) {
        clusterSelect.addEventListener("change", () => populateClusterFields(requestForm));
      }

      requestForm.addEventListener("submit", async (event) => {
        event.preventDefault();
        try {
          const requestID = requestForm.querySelector("[data-request-id]");
          const id = requestID ? requestID.value : "";
          const url = id ? `/api/requests/${id}/edit` : "/api/requests";
          const response = await fetch(apiPath(url), jsonOptions(formToObject(requestForm)));
          const body = await response.json().catch(() => ({}));
          if (!response.ok) {
            throw new Error(body.error || body.message || (id ? "Unable to update request" : "Unable to create request"));
          }
          closeModal(requestModal);
          requestForm.reset();
          populateClusterFields(requestForm);
          resetRequestModal();
          toast(id ? "Request updated" : "Request created");
          fetchRequests();
          updateStats();
          refreshRequestTrend();
        } catch (error) {
          toast(error.message);
        }
      });
    }

    const filterNodes = ["[data-filter-search]", "[data-filter-from]", "[data-filter-to]", "[data-filter-status]"];
    filterNodes.forEach((selector) => {
      const node = document.querySelector(selector);
      if (node) {
        node.addEventListener("input", debounce(() => {
          state.page = 1;
          syncQuickFilters();
          syncRequestURL();
          fetchRequests();
        }, 220));
      }
    });

    document.querySelectorAll("[data-sort]").forEach((header) => {
      header.addEventListener("click", () => {
        const key = header.getAttribute("data-sort");
        if (state.sortKey === key) {
          state.sortDirection = state.sortDirection === "asc" ? "desc" : "asc";
        } else {
          state.sortKey = key;
          state.sortDirection = "asc";
        }
        renderTable();
      });
    });

    const prev = document.querySelector("[data-page-prev]");
    const next = document.querySelector("[data-page-next]");
    if (prev) {
      prev.addEventListener("click", () => {
        state.page = Math.max(1, state.page - 1);
        renderTable();
      });
    }
    if (next) {
      next.addEventListener("click", () => {
        const max = Math.max(1, Math.ceil(state.rows.length / state.perPage));
        state.page = Math.min(max, state.page + 1);
        renderTable();
      });
    }

    document.querySelectorAll("[data-export-csv]").forEach((button) => {
      button.addEventListener("click", exportCSV);
    });

    bindActionModal();
  }

  function bindSettingsPage() {
    const page = document.querySelector("[data-settings-page]");
    if (!page) {
      return;
    }

    const currentUser = readUser();
    if (!currentUser || (currentUser.role !== "fte_ops" && currentUser.role !== "fte_mm")) {
      location.href = appRoute("/dashboard");
      return;
    }

    const form = document.querySelector("[data-add-role-form]");
    const roleSelect = document.querySelector("[data-role-select]");
    const actorRole = document.querySelector("[data-actor-role]");
    if (actorRole) {
      actorRole.value = currentUser.role;
    }

    const syncRoleFields = () => {
      const role = roleSelect ? roleSelect.value : "";
      const fteRole = role === "fte_ops" || role === "fte_mm";
      document.querySelectorAll("[data-email-field]").forEach((node) => node.classList.toggle("is-hidden", !role || !fteRole));
      document.querySelectorAll("[data-ops-id-field]").forEach((node) => node.classList.toggle("is-hidden", !role || fteRole));
      if (form && form.elements.email) {
        form.elements.email.required = Boolean(role) && fteRole;
        form.elements.email.disabled = !fteRole;
        if (!fteRole) {
          form.elements.email.value = "";
        }
      }
      if (form && form.elements.ops_id) {
        form.elements.ops_id.required = Boolean(role) && !fteRole;
        form.elements.ops_id.disabled = !role || fteRole;
        if (fteRole) {
          form.elements.ops_id.value = "";
        }
      }
    };

    if (roleSelect) {
      roleSelect.addEventListener("change", syncRoleFields);
      syncRoleFields();
    }

    if (form) {
      form.addEventListener("reset", () => window.setTimeout(syncRoleFields, 0));
      form.addEventListener("submit", async (event) => {
        event.preventDefault();
        form.classList.add("is-loading");
        setAddRoleError("");

        try {
          const response = await fetch(apiPath("/api/users"), jsonOptions(formToObject(form)));
          const body = await response.json().catch(() => ({}));
          if (!response.ok) {
            throw new Error(body.error || body.message || "Unable to add role");
          }
          form.reset();
          if (actorRole) {
            actorRole.value = currentUser.role;
          }
          syncRoleFields();
          toast("Role added for " + body.name);
        } catch (error) {
          setAddRoleError(error.message);
          toast(error.message);
        } finally {
          form.classList.remove("is-loading");
        }
      });
    }

    document.querySelectorAll("[data-copy-share-link]").forEach((button) => {
      button.addEventListener("click", async () => {
        const input = document.querySelector("[data-share-link]");
        const value = new URL(input ? input.value : "./settings.html", location.href).href;
        try {
          await navigator.clipboard.writeText(value);
          toast("Share link copied");
        } catch (_) {
          if (input) {
            input.value = value;
            input.select();
          }
          toast("Share link ready");
        }
      });
    });
  }

  function bindActionModal() {
    const modal = document.querySelector("[data-action-modal]");
    const form = document.querySelector("[data-action-form]");
    if (!modal || !form) {
      return;
    }

    document.querySelectorAll("[data-close-action-modal]").forEach((button) => {
      button.addEventListener("click", () => closeModal(modal));
    });

    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const id = form.querySelector("[data-action-id]").value;
      const action = form.querySelector("[data-action-type]").value;
      try {
        const response = await fetch(apiPath(`/api/requests/${id}/${action}`), jsonOptions(formToObject(form)));
        const body = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(body.error || body.message || "Unable to update request");
        }
        closeModal(modal);
        toast("Request updated");
        fetchRequests();
        updateStats();
        if (action === "assign") {
          openActionForRow(body, "for-docking");
        }
        if (action === "dock") {
          openTruckLabel(body);
        }
      } catch (error) {
        toast(error.message);
      }
    });
  }

  async function fetchRequests() {
    const page = document.querySelector("[data-requests-page]");
    const params = new URLSearchParams();
    const queue = page ? page.getAttribute("data-default-queue") : "";
    if (queue && queue !== "all") {
      params.set("queue", queue);
    }
    addParam(params, "status", valueOf("[data-filter-status]"));
    addParam(params, "search", valueOf("[data-filter-search]"));
    addParam(params, "date_from", valueOf("[data-filter-from]"));
    addParam(params, "date_to", valueOf("[data-filter-to]"));

    setTableLoading(true);
    try {
      const response = await fetch(apiPath("/api/requests?" + params.toString()));
      const body = await response.json().catch(() => ({ requests: [] }));
      state.rows = Array.isArray(body.requests) ? body.requests : [];
      renderTable();
    } catch (_) {
      state.rows = [];
      renderTable();
    } finally {
      setTableLoading(false);
    }
  }

  function applyRequestURLFilters() {
    const params = new URLSearchParams(location.search);
    const status = params.get("status");
    const search = params.get("search");
    const statusSelect = document.querySelector("[data-filter-status]");
    const searchInput = document.querySelector("[data-filter-search]");
    if (status && statusSelect) {
      const hasOption = Array.from(statusSelect.options).some((option) => option.value === status);
      if (hasOption) {
        statusSelect.value = status;
      }
    }
    if (search && searchInput) {
      searchInput.value = search;
    }
    syncQuickFilters();
  }

  function bindQuickFilters() {
    document.querySelectorAll("[data-status-filter]").forEach((button) => {
      button.addEventListener("click", () => {
        const status = button.getAttribute("data-status-filter") || "ALL";
        const select = document.querySelector("[data-filter-status]");
        if (select) {
          select.value = status;
        }
        state.page = 1;
        syncQuickFilters();
        syncRequestURL();
        fetchRequests();
      });
    });
  }

  function syncQuickFilters() {
    const value = valueOf("[data-filter-status]") || "ALL";
    document.querySelectorAll("[data-status-filter]").forEach((button) => {
      button.classList.toggle("is-active", button.getAttribute("data-status-filter") === value);
    });
  }

  function syncRequestURL() {
    const params = new URLSearchParams(location.search);
    const status = valueOf("[data-filter-status]");
    const search = valueOf("[data-filter-search]");
    if (status && status !== "ALL") {
      params.set("status", status);
    } else {
      params.delete("status");
    }
    if (search) {
      params.set("search", search);
    } else {
      params.delete("search");
    }
    const query = params.toString();
    history.replaceState(null, "", location.pathname + (query ? "?" + query : ""));
  }

  function bindColumnControls() {
    document.querySelectorAll("[data-column-toggle]").forEach((input) => {
      input.addEventListener("change", () => {
        const hidden = readHiddenColumns();
        const key = input.getAttribute("data-column-toggle");
        if (!key) {
          return;
        }
        if (input.checked) {
          hidden.delete(key);
        } else {
          hidden.add(key);
        }
        localStorage.setItem("soc5_hidden_columns", JSON.stringify(Array.from(hidden)));
        applyColumnVisibility();
      });
    });
  }

  function readHiddenColumns() {
    try {
      return new Set(JSON.parse(localStorage.getItem("soc5_hidden_columns") || "[]"));
    } catch (_) {
      return new Set();
    }
  }

  function applyColumnVisibility() {
    const hidden = readHiddenColumns();
    document.querySelectorAll("[data-column-toggle]").forEach((input) => {
      const key = input.getAttribute("data-column-toggle");
      input.checked = !hidden.has(key);
    });
    document.querySelectorAll(".table-panel").forEach((panel) => {
      ["region", "dock", "backlogs", "driver", "trip"].forEach((key) => {
        panel.classList.toggle(`hide-col-${key}`, hidden.has(key));
      });
    });
  }

  function setTableLoading(isLoading) {
    document.querySelectorAll(".table-panel").forEach((panel) => {
      panel.classList.toggle("is-loading", isLoading);
    });
    const tbody = document.querySelector("[data-request-table]");
    if (isLoading && tbody && state.rows.length === 0) {
      tbody.innerHTML = Array.from({ length: 6 }).map(() => (
        `<tr class="skeleton-row"><td colspan="12"><span></span></td></tr>`
      )).join("");
    }
  }

  function renderRequestCards(rows) {
    const list = document.querySelector("[data-request-cards]");
    if (!list) {
      return;
    }
    if (rows.length === 0) {
      list.innerHTML = `<div class="empty-state">No requests found.</div>`;
      return;
    }
    list.innerHTML = rows.map((row) => `
      <article class="request-card">
        <div class="request-card-head">
          <div>
            <strong>${escapeHTML(row.cluster || "-")}</strong>
            <small>${escapeHTML(row.request_timestamp || "-")}</small>
          </div>
          <span class="status-pill ${escapeHTML(row.status || "")}">${escapeHTML(row.status_label || row.status || "-")}</span>
        </div>
        <dl>
          <div><dt>Dock</dt><dd>${escapeHTML(row.dock_no || "-")}</dd></div>
          <div><dt>Truck</dt><dd>${escapeHTML([row.truck_size, row.truck_type].filter(Boolean).join(" ") || "-")}</dd></div>
          <div><dt>Plate</dt><dd>${escapeHTML(row.plate_number || "-")}</dd></div>
          <div><dt>Driver</dt><dd>${escapeHTML(row.driver_id || "-")}</dd></div>
        </dl>
        ${renderActions(row)}
      </article>
    `).join("");
    list.querySelectorAll("[data-row-action]").forEach((button) => {
      button.addEventListener("click", () => openAction(button));
    });
    list.querySelectorAll("[data-edit-request]").forEach((button) => {
      button.addEventListener("click", () => openRequestModal(rowByID(button.getAttribute("data-row-id"))));
    });
    list.querySelectorAll("[data-view-truck-label]").forEach((button) => {
      button.addEventListener("click", () => openTruckLabel(rowByID(button.getAttribute("data-row-id"))));
    });
    list.querySelectorAll("[data-view-request-detail]").forEach((button) => {
      button.addEventListener("click", () => openRequestDetail(rowByID(button.getAttribute("data-row-id"))));
    });
  }

  function renderTable() {
    const tbody = document.querySelector("[data-request-table]");
    if (!tbody) {
      return;
    }

    const rows = [...state.rows].sort((a, b) => compareRows(a, b, state.sortKey, state.sortDirection));
    const maxPage = Math.max(1, Math.ceil(rows.length / state.perPage));
    state.page = Math.min(state.page, maxPage);
    const start = (state.page - 1) * state.perPage;
    const visible = rows.slice(start, start + state.perPage);

    if (visible.length === 0) {
      tbody.innerHTML = state.inlineRequestOpen
        ? renderInlineRequestRow()
        : `<tr><td colspan="12" class="empty-state">No requests found.</td></tr>`;
    } else {
      tbody.innerHTML = (state.inlineRequestOpen ? renderInlineRequestRow() : "") + visible.map(renderRow).join("");
    }

    bindInlineRequestRow(tbody);

    tbody.querySelectorAll("[data-row-action]").forEach((button) => {
      button.addEventListener("click", () => openAction(button));
    });
    tbody.querySelectorAll("[data-edit-request]").forEach((button) => {
      button.addEventListener("click", () => openRequestModal(rowByID(button.getAttribute("data-row-id"))));
    });
    tbody.querySelectorAll("[data-view-truck-label]").forEach((button) => {
      button.addEventListener("click", () => openTruckLabel(rowByID(button.getAttribute("data-row-id"))));
    });
    tbody.querySelectorAll("[data-view-request-detail]").forEach((button) => {
      button.addEventListener("click", () => openRequestDetail(rowByID(button.getAttribute("data-row-id"))));
    });

    renderRequestCards(visible);

    const label = document.querySelector("[data-page-label]");
    if (label) {
      label.textContent = `Page ${state.page} of ${maxPage}`;
    }
  }

  function renderRow(row) {
    return `
      <tr data-row-id="${Number(row.id)}">
        <td data-col="requested">${escapeHTML(row.request_timestamp || "-")}</td>
        <td data-col="cluster">${escapeHTML(row.cluster || "-")}</td>
        <td data-col="region">${escapeHTML(row.region || "-")}</td>
        <td data-col="dock">${escapeHTML(row.dock_no || "-")}</td>
        <td data-col="backlogs">${Number(row.backlogs || 0)}</td>
        <td data-col="truck">${escapeHTML([row.truck_size, row.truck_type].filter(Boolean).join(" ") || "-")}</td>
        <td data-col="plate">${escapeHTML(row.plate_number || "-")}</td>
        <td data-col="driver">${escapeHTML(row.driver_id || "-")}</td>
        <td data-col="trip">${escapeHTML(row.linehaul_trip_no || "-")}</td>
        <td data-col="docking">${escapeHTML(row.docking_time || "-")}</td>
        <td data-col="status"><span class="status-pill ${escapeHTML(row.status || "")}">${escapeHTML(row.status_label || row.status || "-")}</span></td>
        <td data-col="actions">${renderActions(row)}</td>
      </tr>
    `;
  }

  function renderActions(row) {
    const currentUser = readUser();
    const role = currentUser ? currentUser.role : "";
    const id = Number(row.id);
    const buttons = [];

    if (role === "fte_ops" && (row.status === "PENDING" || row.status === "REJECTED_BY_MM")) {
      buttons.push(rowButton("edit", "Edit", `data-edit-request data-row-id="${id}"`));
      buttons.push(rowButton("check", "Approve", `data-row-action="approve" data-row-id="${id}"`));
      buttons.push(rowButton("close", "Cancel", `data-row-action="cancel" data-row-id="${id}"`));
    }
    if (role === "fte_mm" && row.status === "APPROVED") {
      buttons.push(rowButton("user-plus", "Assign", `data-row-action="assign" data-row-id="${id}"`));
      buttons.push(rowButton("xcircle", "Reject", `data-row-action="reject" data-row-id="${id}"`));
    }
    if (role === "fte_mm" && row.status === "ASSIGNED") {
      buttons.push(rowButton("truck", "Assign Plate", `data-row-action="for-docking" data-row-id="${id}"`));
    }
    if ((role === "dock_officer" || role === "doc_officer") && row.status === "FOR_DOCKING") {
      buttons.push(rowButton("box", "Dock", `data-row-action="dock" data-row-id="${id}"`));
    }
    if ((role === "dock_officer" || role === "doc_officer") && row.status === "DOCKED") {
      buttons.push(rowButton("printer", "View", `data-view-truck-label data-row-id="${id}"`));
    }

    buttons.unshift(rowButton("search", "Details", `data-view-request-detail data-row-id="${id}"`));
    return `<div class="row-actions">${buttons.join("")}</div>`;
  }

  function rowButton(icon, label, attrs) {
    return `<button type="button" ${attrs}><span class="ui-icon icon-${icon}" aria-hidden="true"></span><span>${escapeHTML(label)}</span></button>`;
  }

  function openAction(button) {
    openActionForRow(rowByID(button.getAttribute("data-row-id")), button.getAttribute("data-row-action"));
  }

  function openActionForRow(row, action) {
    const modal = document.querySelector("[data-action-modal]");
    const form = document.querySelector("[data-action-form]");
    if (!modal || !form || !row) {
      return;
    }

    form.reset();
    form.querySelector("[data-action-id]").value = row.id;
    form.querySelector("[data-action-type]").value = action;
    fillCurrentUserFields(form);
    if (form.elements.plate_number) {
      form.elements.plate_number.value = row.plate_number || "";
    }
    if (form.elements.driver_id) {
      form.elements.driver_id.value = row.driver_id || "";
    }
    if (form.elements.linehaul_trip_no) {
      form.elements.linehaul_trip_no.value = row.linehaul_trip_no || "";
    }
    if (form.elements.docking_time) {
      form.elements.docking_time.value = localDateTimeValue();
    }

    const title = document.querySelector("[data-action-title]");
    if (title) {
      title.textContent = actionTitle(action);
    }

    form.querySelectorAll("[data-approve-field]").forEach((node) => node.classList.toggle("is-hidden", action !== "approve"));
    form.querySelectorAll("[data-assign-field]").forEach((node) => node.classList.toggle("is-hidden", action !== "assign"));
    form.querySelectorAll("[data-plate-field]").forEach((node) => node.classList.toggle("is-hidden", action !== "for-docking"));
    form.querySelectorAll("[data-dock-field]").forEach((node) => node.classList.toggle("is-hidden", action !== "dock"));
    form.querySelectorAll("[data-reject-field]").forEach((node) => node.classList.toggle("is-hidden", action !== "reject" && action !== "cancel"));
    form.querySelectorAll("input, textarea, select").forEach((node) => {
      if (node.type !== "hidden") {
        node.required = !node.closest(".is-hidden") && (node.name === "plate_number" || node.name === "driver_id" || node.name === "linehaul_trip_no" || node.name === "docking_time");
      }
    });

    openModal(modal);
  }

  function rowByID(id) {
    const numericID = Number(id);
    return state.rows.find((row) => Number(row.id) === numericID) || null;
  }

  function openRequestModal(row) {
    const modal = document.querySelector("[data-request-modal]");
    const form = document.querySelector("[data-request-form]");
    if (!modal || !form) {
      return;
    }

    resetRequestModal();
    if (row) {
      const requestID = form.querySelector("[data-request-id]");
      const clusterID = form.querySelector("[data-cluster-id]");
      if (requestID) {
        requestID.value = row.id || "";
      }
      if (clusterID) {
        clusterID.value = "";
      }
      setFormValue(form, "cluster", row.cluster);
      setFormValue(form, "region", row.region);
      setFormValue(form, "dock_no", row.dock_no);
      setFormValue(form, "backlogs", Number(row.backlogs || 0));
      setFormValue(form, "truck_size", row.truck_size || "6W");
      setFormValue(form, "truck_type", row.truck_type);
      setFormValue(form, "ob_ops_pic", row.ob_ops_pic);
      setRequestModalMode("Edit LH Request", "Save");
    } else {
      form.reset();
      populateClusterFields(form);
      fillCurrentUserFields(form);
      setRequestModalMode("New LH Request", "Create");
    }

    openModal(modal);
  }

  function resetRequestModal() {
    const form = document.querySelector("[data-request-form]");
    if (!form) {
      return;
    }
    const requestID = form.querySelector("[data-request-id]");
    if (requestID) {
      requestID.value = "";
    }
    setRequestModalMode("New LH Request", "Create");
  }

  function setRequestModalMode(title, submitLabel) {
    const titleNode = document.querySelector("[data-request-title]");
    const submitNode = document.querySelector("[data-request-submit]");
    if (titleNode) {
      titleNode.textContent = title;
    }
    if (submitNode) {
      submitNode.textContent = submitLabel;
    }
  }

  function setFormValue(form, name, value) {
    if (form.elements[name]) {
      form.elements[name].value = value ?? "";
    }
  }

  function openInlineRequestRow() {
    state.inlineRequestOpen = true;
    state.page = 1;
    renderTable();
    const row = document.querySelector("[data-inline-request-row]");
    if (row) {
      fillCurrentUserFields(row);
      row.scrollIntoView({ block: "nearest", behavior: "smooth" });
      const firstField = row.querySelector("[data-cluster-select]");
      if (firstField) {
        firstField.focus();
      }
    }
  }

  function closeInlineRequestRow() {
    state.inlineRequestOpen = false;
    state.inlineRequestSaving = false;
    renderTable();
  }

  function renderInlineRequestRow() {
    const formID = "inline-request-form";
    const saving = state.inlineRequestSaving;
    const disabled = saving ? "disabled" : "";
    return `
      <tr class="inline-request-row" data-inline-request-row>
        <td data-col="requested">
          <form id="${formID}" data-inline-request-form></form>
          <input form="${formID}" type="hidden" name="cluster_id" data-cluster-id>
          <span class="inline-request-chip">New</span>
        </td>
        <td data-col="cluster">
          <select form="${formID}" name="cluster" required data-cluster-select ${disabled}>
            ${clusterOptionsHTML()}
          </select>
        </td>
        <td data-col="region"><input form="${formID}" name="region" required placeholder="Region" ${disabled}></td>
        <td data-col="dock"><input form="${formID}" name="dock_no" required placeholder="Dock" ${disabled}></td>
        <td data-col="backlogs"><input form="${formID}" name="backlogs" type="number" min="0" value="0" ${disabled}></td>
        <td data-col="truck">
          <div class="inline-truck-fields">
            <select form="${formID}" name="truck_size" ${disabled}>
              <option value="6W">6W</option>
              <option value="6WF">6WF</option>
              <option value="4W">4W</option>
              <option value="10W">10W</option>
            </select>
            <select form="${formID}" name="truck_type" ${disabled}>
              <option value="">Type</option>
              <option value="WETLEASE">WETLEASE</option>
              <option value="DRYLEASE">DRYLEASE</option>
            </select>
          </div>
        </td>
        <td data-col="plate" class="muted">-</td>
        <td data-col="driver" class="muted">-</td>
        <td data-col="trip" class="muted">-</td>
        <td data-col="docking" class="muted">-</td>
        <td data-col="status"><span class="status-pill PENDING">Draft</span></td>
        <td data-col="actions">
          <div class="row-actions">
            <button type="submit" form="${formID}" ${disabled}><span class="ui-icon icon-check" aria-hidden="true"></span><span>${saving ? "Saving" : "Create"}</span></button>
            <button type="button" data-cancel-inline-request ${disabled}><span class="ui-icon icon-close" aria-hidden="true"></span><span>Cancel</span></button>
          </div>
        </td>
      </tr>
    `;
  }

  function bindInlineRequestRow(root) {
    const row = root.querySelector("[data-inline-request-row]");
    const form = root.querySelector("[data-inline-request-form]");
    if (!row || !form) {
      return;
    }

    const clusterSelect = row.querySelector("[data-cluster-select]");
    if (clusterSelect) {
      clusterSelect.addEventListener("change", () => populateClusterFields(row));
    }

    row.querySelectorAll("[data-cancel-inline-request]").forEach((button) => {
      button.addEventListener("click", closeInlineRequestRow);
    });

    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      if (state.inlineRequestSaving) {
        return;
      }

      const payload = formToObject(form);
      state.inlineRequestSaving = true;
      renderTable();
      try {
        const response = await fetch(apiPath("/api/requests"), jsonOptions(payload));
        const body = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(body.error || body.message || "Unable to create request");
        }

        state.inlineRequestOpen = false;
        toast("Request created");
        await fetchRequests();
        updateStats();
        refreshRequestTrend();
      } catch (error) {
        toast(error.message);
      } finally {
        state.inlineRequestSaving = false;
        renderTable();
      }
    });
  }

  function clusterOptionsHTML() {
    return `<option value="">Select cluster</option>` + state.clusters.map((item, index) => (
      `<option value="${escapeHTML(item.cluster)}" data-cluster-index="${index}">${escapeHTML(clusterOptionLabel(item))}</option>`
    )).join("");
  }

  function clusterOptionLabel(item) {
    return [item.cluster, item.hub_name].filter(Boolean).join(" - ");
  }

  function bindRequestDetailModal() {
    const modal = document.querySelector("[data-request-detail-modal]");
    document.querySelectorAll("[data-close-request-detail]").forEach((button) => {
      button.addEventListener("click", () => closeModal(modal));
    });
  }

  function openRequestDetail(row) {
    const modal = document.querySelector("[data-request-detail-modal]");
    const title = document.querySelector("[data-detail-title]");
    const status = document.querySelector("[data-detail-status]");
    const grid = document.querySelector("[data-detail-grid]");
    const timeline = document.querySelector("[data-detail-timeline]");
    if (!modal || !row || !grid || !timeline) {
      return;
    }

    if (title) {
      title.textContent = row.cluster || "Linehaul Request";
    }
    if (status) {
      status.className = `status-pill ${escapeHTML(row.status || "")}`;
      status.textContent = row.status_label || row.status || "-";
    }

    const details = [
      ["Requested", row.request_timestamp],
      ["Region", row.region],
      ["Dock", row.dock_no],
      ["Backlogs", row.backlogs],
      ["Truck", [row.truck_size, row.truck_type].filter(Boolean).join(" ")],
      ["Plate", row.plate_number],
      ["Driver", row.driver_id],
      ["Trip No.", row.linehaul_trip_no],
      ["Docking Time", row.docking_time],
      ["Ops PIC", row.ob_ops_pic],
      ["FTE Ops", row.ob_fte],
      ["FTE MM", row.midmile_fte],
      ["Remarks", row.remarks],
    ];

    grid.innerHTML = details.map(([label, value]) => `
      <div>
        <span>${escapeHTML(label)}</span>
        <strong>${escapeHTML(value || "-")}</strong>
      </div>
    `).join("");

    timeline.innerHTML = buildTimeline(row).map((item) => `
      <div class="timeline-item ${item.done ? "is-done" : ""}">
        <i></i>
        <span>${escapeHTML(item.label)}</span>
        <strong>${escapeHTML(item.value || item.state)}</strong>
      </div>
    `).join("");

    openModal(modal);
  }

  function buildTimeline(row) {
    const order = ["PENDING", "APPROVED", "ASSIGNED", "FOR_DOCKING", "DOCKED"];
    const currentIndex = Math.max(0, order.indexOf(row.status || "PENDING"));
    return [
      { label: "Created", state: "Requested", value: row.request_timestamp, done: true },
      { label: "Ops Approval", state: "Pending Ops", value: row.ob_fte || "", done: currentIndex >= 1 || row.status === "REJECTED_BY_MM" },
      { label: "Midmile Assignment", state: "Pending MM", value: row.midmile_fte || "", done: currentIndex >= 2 },
      { label: "Plate Assigned", state: "For Docking", value: row.plate_number || "", done: currentIndex >= 3 },
      { label: "Docked", state: "Docked", value: row.docking_time || "", done: currentIndex >= 4 },
      { label: "Exception", state: "REJECTED_BY_MM/CANCELLED", value: row.remarks || "", done: row.status === "REJECTED_BY_MM" || row.status === "CANCELLED" },
    ];
  }

  function bindDashboard() {
    if (document.querySelector("[data-request-trend]")) {
      refreshRequestTrend();
      window.setInterval(refreshRequestTrend, 60000);
    }

    document.querySelectorAll("[data-refresh-activity]").forEach((button) => {
      button.addEventListener("click", () => {
        refreshActivity();
        refreshRequestTrend();
      });
    });
  }

  async function refreshRequestTrend() {
    const chart = document.querySelector("[data-request-trend]");
    if (!chart) {
      return;
    }

    const response = await fetch(apiPath("/api/request-trend"));
    const body = await response.json().catch(() => ({ points: [] }));
    const points = Array.isArray(body.points) ? body.points : [];
    renderRequestTrend(points, body.period_label || "6 PM - 6 AM");
  }

  function renderRequestTrend(points, periodLabel) {
    const bars = document.querySelector("[data-trend-bars]");
    const labels = document.querySelector("[data-trend-labels]");
    const scale = document.querySelector("[data-trend-scale]");
    const svg = document.querySelector("[data-trend-svg]");
    const period = document.querySelector("[data-trend-period]");
    if (!bars || !labels || !scale || !svg) {
      return;
    }

    const counts = points.map((point) => Number(point.count || 0));
    const maxCount = Math.max(1, ...counts);
    const roundedMax = Math.max(4, Math.ceil(maxCount / 4) * 4);
    const chartCount = Math.max(points.length, 1);
    bars.style.setProperty("--trend-count", String(chartCount));
    labels.style.setProperty("--trend-count", String(chartCount));
    if (period) {
      period.textContent = periodLabel;
    }

    scale.innerHTML = [4, 3, 2, 1, 0].map((step) => (
      `<span>${Math.round((roundedMax / 4) * step)}</span>`
    )).join("");

    bars.innerHTML = points.map((point) => {
      const count = Number(point.count || 0);
      const height = Math.max(count === 0 ? 2 : 8, (count / roundedMax) * 100);
      return `<div class="trend-bar" style="height: ${height}%"><span>${count}</span></div>`;
    }).join("");

    labels.innerHTML = points.map((point) => `<span>${escapeHTML(point.label || "")}</span>`).join("");
    svg.innerHTML = buildTrendSVG(points, roundedMax);
  }

  function buildTrendSVG(points, maxCount) {
    if (points.length === 0) {
      return "";
    }

    const coordinates = points.map((point, index) => {
      const x = points.length === 1 ? 50 : (index / (points.length - 1)) * 100;
      const y = 100 - (Number(point.count || 0) / maxCount) * 100;
      return { x, y };
    });
    const polyline = coordinates.map((point) => `${point.x.toFixed(2)},${point.y.toFixed(2)}`).join(" ");
    const circles = coordinates.map((point) => (
      `<circle cx="${point.x.toFixed(2)}" cy="${point.y.toFixed(2)}" r="2.2"></circle>`
    )).join("");

    return `<polyline points="${polyline}"></polyline>${circles}`;
  }

  async function refreshActivity(searchTerm) {
    const list = document.querySelector("[data-activity-list]");
    if (!list) {
      return;
    }
    const params = new URLSearchParams();
    const search = searchTerm ?? valueOf("[data-global-search]");
    addParam(params, "search", search);
    const query = params.toString();
    const response = await fetch(apiPath("/api/requests" + (query ? "?" + query : "")));
    const body = await response.json().catch(() => ({ requests: [] }));
    const rows = (body.requests || []).slice(0, 8);
    if (rows.length === 0) {
      list.innerHTML = `<div class="empty-state">${search ? "No matching request activity." : "No recent request activity."}</div>`;
      return;
    }
    list.innerHTML = rows.map((row) => `
      <div class="activity-item">
        <span class="status-dot ${escapeHTML(row.status || "")}"></span>
        <div>
          <strong>${escapeHTML(row.cluster || "-")}</strong>
          <small>${escapeHTML(row.request_timestamp || "-")} - ${escapeHTML(row.status_label || "-")}</small>
        </div>
        <span class="plate-text">${escapeHTML(row.plate_number || "No plate")}</span>
      </div>
    `).join("");
  }

  function bindNotifications() {
    const drawer = document.querySelector("[data-notification-drawer]");
    const scrim = document.querySelector("[data-notification-scrim]");
    const close = () => {
      if (drawer) {
        drawer.classList.add("is-hidden");
      }
      if (scrim) {
        scrim.classList.add("is-hidden");
      }
      document.querySelectorAll("[data-open-notifications]").forEach((button) => {
        button.setAttribute("aria-expanded", "false");
      });
    };

    document.querySelectorAll("[data-open-notifications]").forEach((button) => {
      button.addEventListener("click", async () => {
        await unlockNotificationAudio();
        clearNotificationAlert();
        persistNotificationCounts(readUser());
        if (drawer && scrim) {
          drawer.classList.toggle("is-hidden");
          scrim.classList.toggle("is-hidden");
          button.setAttribute("aria-expanded", String(!drawer.classList.contains("is-hidden")));
        }
      });
    });

    document.querySelectorAll("[data-close-notifications], [data-notification-scrim], [data-clear-notifications]").forEach((node) => {
      node.addEventListener("click", () => {
        clearNotificationAlert();
        close();
      });
    });
  }

  function bindWorkflowEvents() {
    if (!window.EventSource) {
      return;
    }

    const source = new EventSource(apiPath("/api/events"), { withCredentials: true });
    ["request.created", "request.updated", "request.status_changed"].forEach((eventName) => {
      source.addEventListener(eventName, () => {
        scheduleWorkflowRefresh();
      });
    });
    source.addEventListener("error", () => {
      return;
    });
  }

  function scheduleWorkflowRefresh() {
    window.clearTimeout(workflowRefreshTimer);
    workflowRefreshTimer = window.setTimeout(() => {
      updateStats();
      if (document.querySelector("[data-requests-page]")) {
        fetchRequests();
      }
      if (document.querySelector("[data-request-trend]")) {
        refreshRequestTrend();
      }
      if (document.querySelector("[data-activity-list]")) {
        refreshActivity();
      }
    }, 120);
  }

  function bindTruckLabelModal() {
    const modal = document.querySelector("[data-truck-label-modal]");
    if (!modal) {
      return;
    }

    document.querySelectorAll("[data-close-truck-label]").forEach((button) => {
      button.addEventListener("click", () => closeModal(modal));
    });
    document.querySelectorAll("[data-print-truck-label]").forEach((button) => {
      button.addEventListener("click", () => window.print());
    });
  }

  function bindNotificationAudioUnlock() {
    const unlock = () => {
      unlockNotificationAudio();
    };
    ["pointerdown", "keydown", "touchstart"].forEach((eventName) => {
      document.addEventListener(eventName, unlock, { passive: true });
    });
  }

  async function updateStats() {
    try {
      const response = await fetch(apiPath("/api/stats"));
      const stats = await response.json();
      const ops = Number(stats.pending_ops || stats.pending || stats.PENDING || 0);
      const mm = Number(stats.pending_mm || stats.approved || stats.APPROVED || 0);
      const dock = Number(stats.for_docking || 0);
      setBadge("ops", ops);
      setBadge("mm", mm);
      setBadge("dock", dock);
      document.querySelectorAll("[data-ops-pending-count]").forEach((node) => node.textContent = ops);
      document.querySelectorAll("[data-mm-pending-count]").forEach((node) => node.textContent = mm);
      document.querySelectorAll("[data-dock-pending-count]").forEach((node) => node.textContent = dock);
      setText("[data-stat-total-today]", stats.total_today || 0);
      setText("[data-stat-pending-ops]", ops);
      setText("[data-stat-pending-mm]", mm);
      setText("[data-stat-for-docking]", dock);
      setText("[data-stat-confirmed-trucks]", stats.confirmed_trucks || 0);
      setText("[data-stat-rejected]", stats.rejected || 0);
      setText("[data-stat-open-alerts]", ops + mm + dock);
      setNotificationCount(ops + mm + dock);
      renderDashboardWidgets({
        ops,
        mm,
        dock,
        confirmed: Number(stats.confirmed_trucks || 0),
        rejected: Number(stats.rejected || 0),
      });

      const currentUser = readUser();
      if (currentUser && currentUser.role === "fte_ops" && ops > state.lastOps) {
        triggerNotificationAlert(`${ops - state.lastOps} new Ops approval item${ops - state.lastOps === 1 ? "" : "s"}`);
      }
      if (currentUser && currentUser.role === "fte_mm" && mm > state.lastMM) {
        triggerNotificationAlert(`${mm - state.lastMM} new Midmile item${mm - state.lastMM === 1 ? "" : "s"}`);
      }
      if (currentUser && (currentUser.role === "dock_officer" || currentUser.role === "doc_officer") && dock > state.lastDock) {
        triggerNotificationAlert(`${dock - state.lastDock} new docking item${dock - state.lastDock === 1 ? "" : "s"}`);
      }

      state.lastOps = ops;
      state.lastMM = mm;
      state.lastDock = dock;
      persistNotificationCounts(currentUser);
    } catch (_) {
      return;
    }
  }

  function persistNotificationCounts(currentUser) {
    if (!currentUser) {
      return;
    }
    if (currentUser.role === "fte_ops") {
      sessionStorage.setItem("last_ops_count", String(state.lastOps));
    }
    if (currentUser.role === "fte_mm") {
      sessionStorage.setItem("last_mm_count", String(state.lastMM));
    }
    if (currentUser.role === "dock_officer" || currentUser.role === "doc_officer") {
      sessionStorage.setItem("last_dock_count", String(state.lastDock));
    }
  }

  function setNotificationCount(count) {
    const total = Number(count || 0);
    document.querySelectorAll("[data-open-notifications]").forEach((button) => {
      button.classList.toggle("has-count", total > 0);
      if (total === 0) {
        button.classList.remove("has-unread", "is-alerting");
      }
      button.setAttribute("aria-label", total > 0 ? `${total} active queue alerts` : "No active queue alerts");
      button.setAttribute("title", total > 0 ? `${total} active queue alerts` : "Notifications");
    });
    document.querySelectorAll("[data-notification-count]").forEach((node) => {
      node.textContent = total;
    });
  }

  function renderDashboardWidgets(stats) {
    const total = Math.max(1, stats.ops + stats.mm + stats.dock + stats.rejected);
    const ops = (stats.ops / total) * 100;
    const mm = (stats.mm / total) * 100;
    const dock = (stats.dock / total) * 100;
    document.querySelectorAll("[data-status-donut] .donut-chart").forEach((node) => {
      node.style.background = `conic-gradient(var(--gt-warning) 0 ${ops}%, var(--gt-info) ${ops}% ${ops + mm}%, var(--gt-success) ${ops + mm}% ${ops + mm + dock}%, var(--gt-danger) ${ops + mm + dock}% 100%)`;
    });

    const workloadMax = Math.max(1, stats.ops, stats.mm, stats.dock);
    document.querySelectorAll("[data-workload-bars] a").forEach((row) => {
      const value = Number(row.querySelector("strong")?.textContent || 0);
      row.style.setProperty("--bar", `${Math.max(4, (value / workloadMax) * 100)}%`);
    });

    const score = Math.round((stats.confirmed / Math.max(1, stats.confirmed + stats.ops + stats.mm + stats.dock + stats.rejected)) * 100);
    document.querySelectorAll("[data-sla-score]").forEach((node) => {
      node.textContent = `${score}%`;
      const ring = node.closest(".sla-ring");
      if (ring) {
        ring.style.setProperty("--score", `${score}%`);
      }
    });
  }

  function triggerNotificationAlert(message) {
    chime();
    toast(message);
    document.querySelectorAll("[data-open-notifications]").forEach((button) => {
      button.classList.remove("is-alerting");
      void button.offsetWidth;
      button.classList.add("has-unread", "is-alerting");
      window.setTimeout(() => button.classList.remove("is-alerting"), 1600);
    });
  }

  function clearNotificationAlert() {
    document.querySelectorAll("[data-open-notifications]").forEach((button) => {
      button.classList.remove("has-unread", "is-alerting");
    });
  }

  async function fetchClusters() {
    const selects = document.querySelectorAll("[data-cluster-select]");
    if (selects.length === 0) {
      return;
    }
    const response = await fetch(apiPath("/api/clusters"));
    const clusters = await response.json().catch(() => []);
    state.clusters = Array.isArray(clusters) ? clusters : [];
    document.querySelectorAll("[data-cluster-select]").forEach((select) => {
      const currentValue = select.value;
      select.innerHTML = clusterOptionsHTML();
      if (currentValue) {
        select.value = currentValue;
      }
    });
  }

  function populateClusterFields(form) {
    const select = form.querySelector("[data-cluster-select]");
    if (!select) {
      return;
    }

    const option = select.selectedOptions[0];
    const index = option ? Number(option.getAttribute("data-cluster-index")) : -1;
    const cluster = Number.isInteger(index) && index >= 0 ? state.clusters[index] : null;
    const clusterID = form.querySelector("[data-cluster-id]");
    const region = form.elements ? form.elements.region : form.querySelector(`[name="region"]`);
    const dockNo = form.elements ? form.elements.dock_no : form.querySelector(`[name="dock_no"]`);
    const backlogs = form.elements ? form.elements.backlogs : form.querySelector(`[name="backlogs"]`);

    if (clusterID) {
      clusterID.value = cluster ? cluster.id || "" : "";
    }
    if (region) {
      region.value = cluster ? cluster.region || "" : "";
    }
    if (dockNo) {
      dockNo.value = cluster ? cluster.dock_no || "" : "";
    }
    if (backlogs) {
      backlogs.value = cluster ? Number(cluster.backlogs || 0) : 0;
    }
  }

  function exportCSV() {
    if (state.rows.length === 0) {
      toast("No rows to export");
      return;
    }
    const headers = ["Requested", "Cluster", "Region", "Dock", "Backlogs", "Truck Size", "Truck Type", "Plate", "Driver", "Trip No", "Docking Time", "Status"];
    const lines = [headers.join(",")];
    state.rows.forEach((row) => {
      lines.push([
        row.request_timestamp,
        row.cluster,
        row.region,
        row.dock_no,
        row.backlogs,
        row.truck_size,
        row.truck_type,
        row.plate_number,
        row.driver_id,
        row.linehaul_trip_no,
        row.docking_time,
        row.status_label,
      ].map(csvCell).join(","));
    });

    const blob = new Blob([lines.join("\n")], { type: "text/csv;charset=utf-8" });
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.download = "linehaul-requests.csv";
    link.click();
    URL.revokeObjectURL(link.href);
  }

  function openTruckLabel(row) {
    const modal = document.querySelector("[data-truck-label-modal]");
    const sheet = document.querySelector("[data-truck-label-sheet]");
    const image = document.querySelector("[data-truck-label-image]");
    if (!modal || !sheet || !image || !row) {
      return;
    }

    const hubs = splitHubs(row.cluster);
    const type = hubs.length >= 3 ? "triload" : hubs.length === 2 ? "coload" : "single";
    const src = type === "triload"
      ? "./truck_label/triload_lh.jpg"
      : type === "coload"
        ? "./truck_label/coload_lh.jpg"
        : "./truck_label/single_lh.jpg";

    sheet.classList.remove("is-single", "is-coload", "is-triload");
    sheet.classList.add(`is-${type}`);
    image.src = src;

    setText("[data-label-plate]", row.plate_number || "-");
    setText("[data-label-driver-text]", row.driver_id || "");
    setText("[data-label-dock]", row.dock_no || "-");
    setText("[data-label-time]", compactDockingTime(row.docking_time));
    setText("[data-label-load='1']", hubs[0] || "");
    setText("[data-label-load='2']", hubs[1] || "");
    setText("[data-label-load='3']", hubs[2] || "");
    setDriverQR(row.driver_id || "");

    openModal(modal);
  }

  function splitHubs(value) {
    return String(value || "")
      .split(",")
      .map((part) => part.trim())
      .filter(Boolean)
      .slice(0, 3);
  }

  function compactDockingTime(value) {
    const text = String(value || "").trim();
    if (!text || text === "-") {
      return "-";
    }

    const parsed = new Date(text);
    if (!Number.isNaN(parsed.getTime())) {
      return parsed.toLocaleString([], {
        month: "short",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      });
    }
    return text.replace(/\s*\d{4}\s*/, " ");
  }

  function setText(selector, value) {
    document.querySelectorAll(selector).forEach((node) => {
      node.textContent = value;
    });
  }

  function setDriverQR(driverID) {
    const image = document.querySelector("[data-label-driver-qr]");
    if (!image) {
      return;
    }

    const value = String(driverID || "").trim();
    image.alt = value ? `Driver ID QR Code: ${value}` : "Driver ID QR Code";
    image.src = value ? apiPath(`/api/qr?value=${encodeURIComponent(value)}`) : "";
  }

  function openModal(modal) {
    if (!modal) {
      return;
    }
    fillCurrentUserFields(modal);
    modal.classList.remove("is-hidden");
  }

  function closeModal(modal) {
    if (modal) {
      modal.classList.add("is-hidden");
    }
  }

  function fillCurrentUserFields(root) {
    const currentUser = readUser();
    root.querySelectorAll("[data-current-user-name]").forEach((node) => {
      node.value = currentUser ? currentUser.name : "";
    });
  }

  function formToObject(form) {
    const data = new FormData(form);
    const payload = {};
    data.forEach((value, key) => {
      payload[key] = value;
    });
    if (payload.backlogs !== undefined) {
      payload.backlogs = Number(payload.backlogs || 0);
    }
    if (payload.cluster_id !== undefined) {
      payload.cluster_id = Number(payload.cluster_id || 0);
    }
    return payload;
  }

  function jsonOptions(payload) {
    return {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    };
  }

  function setLoginError(message) {
    const node = document.querySelector("[data-login-error]");
    if (node) {
      node.textContent = message;
    }
  }

  function setAddRoleError(message) {
    const node = document.querySelector("[data-add-role-error]");
    if (node) {
      node.textContent = message;
    }
  }

  function setBadge(name, count) {
    document.querySelectorAll(`[data-badge="${name}"]`).forEach((node) => {
      node.textContent = count;
      node.classList.toggle("has-items", count > 0);
    });
  }

  function valueOf(selector) {
    const node = document.querySelector(selector);
    return node ? node.value : "";
  }

  function addParam(params, key, value) {
    if (value && value !== "ALL") {
      params.set(key, value);
    }
  }

  function compareRows(a, b, key, direction) {
    const left = a[key] || "";
    const right = b[key] || "";
    const result = typeof left === "number" || typeof right === "number"
      ? Number(left) - Number(right)
      : String(left).localeCompare(String(right));
    return direction === "asc" ? result : result * -1;
  }

  function actionTitle(action) {
    switch (action) {
      case "approve":
        return "Approve Request";
      case "assign":
        return "Assign Truck";
      case "for-docking":
        return "Assign Plate #";
      case "dock":
        return "Docking Details";
      case "reject":
        return "Reject Request";
      case "cancel":
        return "Cancel Request";
      default:
        return "Update Request";
    }
  }

  function roleLabel(role) {
    switch (role) {
      case "ops_pic":
        return "Ops PIC";
      case "fte_ops":
        return "FTE Ops";
      case "fte_mm":
        return "FTE MM";
      case "dock_officer":
      case "doc_officer":
        return "Dock Officer";
      default:
        return role || "Guest";
    }
  }

  function localDateTimeValue() {
    const value = new Date();
    value.setMinutes(value.getMinutes() - value.getTimezoneOffset());
    return value.toISOString().slice(0, 16);
  }

  function initials(name) {
    return String(name || "U").split(/\s+/).map((part) => part.charAt(0)).join("").slice(0, 2).toUpperCase();
  }

  function escapeHTML(value) {
    return String(value ?? "").replace(/[&<>"']/g, (char) => ({
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#039;",
    }[char]));
  }

  function csvCell(value) {
    return `"${String(value ?? "").replace(/"/g, '""')}"`;
  }

  function debounce(fn, wait) {
    let timeout;
    return function (...args) {
      window.clearTimeout(timeout);
      timeout = window.setTimeout(() => fn.apply(this, args), wait);
    };
  }

  function toast(message) {
    const region = document.querySelector("[data-toast-region]");
    if (!region) {
      return;
    }
    const node = document.createElement("div");
    node.className = "toast";
    node.textContent = message;
    region.appendChild(node);
    window.setTimeout(() => node.remove(), 3200);
  }

  function chime() {
    try {
      const ctx = getNotificationAudioContext();
      if (!ctx) {
        return;
      }
      if (ctx.state !== "running") {
        pendingNotificationChimes += 1;
        if (!audioUnlockToastShown) {
          audioUnlockToastShown = true;
          toast("Click anywhere to enable notification sound");
        }
        return;
      }

      playNotificationTone(ctx);
    } catch (_) {
      return;
    }
  }

  async function unlockNotificationAudio() {
    try {
      const ctx = getNotificationAudioContext();
      if (!ctx) {
        return;
      }
      if (ctx.state === "suspended") {
        await ctx.resume();
      }
      notificationAudioReady = ctx.state === "running";
      if (!notificationAudioReady) {
        return;
      }

      const silent = ctx.createOscillator();
      const gain = ctx.createGain();
      gain.gain.setValueAtTime(0.0001, ctx.currentTime);
      silent.connect(gain);
      gain.connect(ctx.destination);
      silent.start(ctx.currentTime);
      silent.stop(ctx.currentTime + 0.02);

      if (pendingNotificationChimes > 0) {
        pendingNotificationChimes = 0;
        playNotificationTone(ctx);
      }
    } catch (_) {
      return;
    }
  }

  function getNotificationAudioContext() {
    const AudioContext = window.AudioContext || window.webkitAudioContext;
    if (!AudioContext) {
      return null;
    }
    if (!notificationAudioContext) {
      notificationAudioContext = new AudioContext();
    }
    return notificationAudioContext;
  }

  function playNotificationTone(ctx) {
    try {
      const gain = ctx.createGain();
      gain.connect(ctx.destination);
      gain.gain.setValueAtTime(0.0001, ctx.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.18, ctx.currentTime + 0.02);
      gain.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + 0.7);

      [660, 880, 990].forEach((frequency, index) => {
        const osc = ctx.createOscillator();
        osc.type = "sine";
        osc.frequency.value = frequency;
        osc.connect(gain);
        osc.start(ctx.currentTime + index * 0.09);
        osc.stop(ctx.currentTime + 0.46 + index * 0.09);
      });
    } catch (_) {
      return;
    }
  }
})();

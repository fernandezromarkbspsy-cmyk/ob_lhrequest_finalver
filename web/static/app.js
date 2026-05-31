(function () {
  const storageKey = "soc5_user";
  const themeKey = "soc5_theme";
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
    lastOps: Number(sessionStorage.getItem("last_ops_count") || 0),
    lastMM: Number(sessionStorage.getItem("last_mm_count") || 0),
    lastDock: Number(sessionStorage.getItem("last_dock_count") || 0),
  };

  applyStoredTheme();

  document.addEventListener("DOMContentLoaded", () => {
    applyRoleVisibility();
    bindNavigation();
    bindTopbar();
    bindLogin();
    bindRequestsPage();
    bindSettingsPage();
    bindDashboard();
    bindNotifications();
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
      button.addEventListener("click", async () => {
        await fetch("/api/logout", { method: "POST" }).catch(() => {});
        localStorage.removeItem(storageKey);
        location.href = "/";
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
  }

  function applyStoredTheme() {
    const theme = localStorage.getItem(themeKey) || "light";
    const isDark = theme === "dark";
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

    enforceLoginModal();

    document.querySelectorAll("[data-open-login]").forEach((button) => {
      button.addEventListener("click", () => openModal(modal));
    });
    document.querySelectorAll("[data-close-login]").forEach((button) => {
      button.addEventListener("click", () => {
        if (readUser()) {
          closeModal(modal);
        } else {
          enforceLoginModal();
        }
      });
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
        const response = await fetch("/api/login", jsonOptions(payload));
        const body = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(body.message || "Invalid credentials");
        }

        saveUser(body);
        toast("Signed in as " + body.name);
        form.classList.remove("shake");
        form.classList.add("success");
        enforceLoginModal();
        window.setTimeout(() => {
          location.href = body.redirect || "/dashboard";
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

  function enforceLoginModal() {
    const modal = document.querySelector("[data-login-modal]");
    if (!modal) {
      return;
    }

    const currentUser = readUser();
    modal.classList.toggle("is-auth-required", !currentUser);
    modal.classList.toggle("is-hidden", Boolean(currentUser));
    document.body.classList.toggle("login-locked", !currentUser);

    document.querySelectorAll("[data-close-login]").forEach((button) => {
      button.hidden = !currentUser;
      button.setAttribute("aria-hidden", String(!currentUser));
      button.tabIndex = currentUser ? 0 : -1;
    });
  }

  function bindRequestsPage() {
    const page = document.querySelector("[data-requests-page]");
    if (!page) {
      return;
    }

    fetchClusters();
    fetchRequests();

    const requestModal = document.querySelector("[data-request-modal]");
    const requestForm = document.querySelector("[data-request-form]");
    document.querySelectorAll("[data-open-request-modal]").forEach((button) => {
      button.addEventListener("click", () => openRequestModal());
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
          const response = await apiFetch(url, jsonOptions(formToObject(requestForm)));
          const body = await response.json().catch(() => ({}));
          if (!response.ok) {
            throw new Error(body.message || (id ? "Unable to update request" : "Unable to create request"));
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
      location.href = "/dashboard";
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
          const response = await apiFetch("/api/users", jsonOptions(formToObject(form)));
          const body = await response.json().catch(() => ({}));
          if (!response.ok) {
            throw new Error(body.message || "Unable to add role");
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
        const value = new URL(input ? input.value : "/settings", location.origin).href;
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
        const response = await apiFetch(`/api/requests/${id}/${action}`, jsonOptions(formToObject(form)));
        const body = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(body.message || "Unable to update request");
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

    const response = await apiFetch("/api/requests?" + params.toString());
    const body = await response.json().catch(() => ({ requests: [] }));
    state.rows = Array.isArray(body.requests) ? body.requests : [];
    renderTable();
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
      tbody.innerHTML = `<tr><td colspan="12" class="empty-state">No requests found.</td></tr>`;
    } else {
      tbody.innerHTML = visible.map(renderRow).join("");
    }

    tbody.querySelectorAll("[data-row-action]").forEach((button) => {
      button.addEventListener("click", () => openAction(button));
    });
    tbody.querySelectorAll("[data-edit-request]").forEach((button) => {
      button.addEventListener("click", () => openRequestModal(rowByID(button.getAttribute("data-row-id"))));
    });
    tbody.querySelectorAll("[data-view-truck-label]").forEach((button) => {
      button.addEventListener("click", () => openTruckLabel(rowByID(button.getAttribute("data-row-id"))));
    });

    const label = document.querySelector("[data-page-label]");
    if (label) {
      label.textContent = `Page ${state.page} of ${maxPage}`;
    }
  }

  function renderRow(row) {
    return `
      <tr>
        <td>${escapeHTML(row.request_timestamp || "-")}</td>
        <td>${escapeHTML(row.cluster || "-")}</td>
        <td>${escapeHTML(row.region || "-")}</td>
        <td>${escapeHTML(row.dock_no || "-")}</td>
        <td>${Number(row.backlogs || 0)}</td>
        <td>${escapeHTML([row.truck_size, row.truck_type].filter(Boolean).join(" ") || "-")}</td>
        <td>${escapeHTML(row.plate_number || "-")}</td>
        <td>${escapeHTML(row.driver_id || "-")}</td>
        <td>${escapeHTML(row.linehaul_trip_no || "-")}</td>
        <td>${escapeHTML(row.docking_time || "-")}</td>
        <td><span class="status-pill ${escapeHTML(row.status || "")}">${escapeHTML(row.status_label || row.status || "-")}</span></td>
        <td>${renderActions(row)}</td>
      </tr>
    `;
  }

  function renderActions(row) {
    const currentUser = readUser();
    const role = currentUser ? currentUser.role : "";
    const id = Number(row.id);
    const buttons = [];

    if (role === "fte_ops" && (row.status === "PENDING_OPS" || row.status === "REJECTED")) {
      buttons.push(rowButton("edit", "Edit", `data-edit-request data-row-id="${id}"`));
      buttons.push(rowButton("check", "Approve", `data-row-action="approve" data-row-id="${id}"`));
      buttons.push(rowButton("close", "Cancel", `data-row-action="cancel" data-row-id="${id}"`));
    }
    if (role === "fte_mm" && row.status === "PENDING_MM") {
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

    if (buttons.length === 0) {
      return `<span class="muted">-</span>`;
    }
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
      setFormValue(form, "truck_size", row.truck_size || "6WH");
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

    const response = await apiFetch("/api/request-trend");
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
    const response = await apiFetch("/api/requests" + (query ? "?" + query : ""));
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
    document.querySelectorAll("[data-open-notifications]").forEach((button) => {
      button.addEventListener("click", async () => {
        await unlockNotificationAudio();
        clearNotificationAlert();
        persistNotificationCounts(readUser());
        const count = Number(document.querySelector("[data-notification-count]")?.textContent || 0);
        toast(count > 0 ? `${count} active queue alert${count === 1 ? "" : "s"}` : "No active queue alerts");
      });
    });
  }

  function bindWorkflowEvents() {
    if (!window.EventSource) {
      return;
    }

    const source = new EventSource("/api/events");
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
      const response = await apiFetch("/api/stats");
      const stats = await response.json();
      const ops = Number(stats.pending_ops || 0);
      const mm = Number(stats.pending_mm || 0);
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
    const select = document.querySelector("[data-cluster-select]");
    if (!select) {
      return;
    }
    const response = await apiFetch("/api/clusters");
    const clusters = await response.json().catch(() => []);
    state.clusters = Array.isArray(clusters) ? clusters : [];
    select.innerHTML = `<option value="">Select cluster</option>` + state.clusters.map((item, index) => (
      `<option value="${escapeHTML(item.cluster)}" data-cluster-index="${index}">${escapeHTML(item.cluster)}</option>`
    )).join("");
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
    const region = form.elements.region;
    const dockNo = form.elements.dock_no;
    const backlogs = form.elements.backlogs;

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
      ? "/truck_label/triload_lh.jpg"
      : type === "coload"
        ? "/truck_label/coload_lh.jpg"
        : "/truck_label/single_lh.jpg";

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
    image.src = value ? `/api/qr?value=${encodeURIComponent(value)}` : "";
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
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    };
  }

  async function apiFetch(url, options) {
    const response = await fetch(url, options);
    if (response.status === 401) {
      localStorage.removeItem(storageKey);
      applyRoleVisibility();
      enforceLoginModal();
      throw new Error("Session expired. Please sign in again.");
    }
    return response;
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
      case "data_team":
        return "Data Team";
      case "admin":
        return "Admin";
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

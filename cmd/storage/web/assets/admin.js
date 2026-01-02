(() => {
  const sessionKey = "adminSession";
  const apiBaseKey = "apiBase";

  const apiBaseInput = document.getElementById("api-base");
  const saveApiBaseBtn = document.getElementById("save-api-base");
  const loginPanel = document.getElementById("login-panel");
  const adminPanel = document.getElementById("admin-panel");
  const loginMessage = document.getElementById("login-message");
  const adminMessage = document.getElementById("admin-message");

  const otpInput = document.getElementById("otp-code");
  const sendOtpBtn = document.getElementById("send-otp");
  const verifyOtpBtn = document.getElementById("verify-otp");
  const logoutBtn = document.getElementById("logout");

  const statRequests = document.getElementById("stat-requests");
  const statUsers = document.getElementById("stat-users");
  const statWindow = document.getElementById("stat-window");
  const statWindowSelect = document.getElementById("stat-window-select");

  const logsTable = document.getElementById("logs-table");
  const refreshLogsBtn = document.getElementById("refresh-logs");

  const keysTable = document.getElementById("keys-table");
  const refreshKeysBtn = document.getElementById("refresh-keys");
  const createKeyBtn = document.getElementById("create-key");
  const createKeyMessage = document.getElementById("create-key-message");
  const newClientId = document.getElementById("new-client-id");
  const newRateLimit = document.getElementById("new-rate-limit");
  const newStatus = document.getElementById("new-status");

  const promptText = document.getElementById("prompt-text");
  const refreshPromptBtn = document.getElementById("refresh-prompt");
  const savePromptBtn = document.getElementById("save-prompt");
  const promptMessage = document.getElementById("prompt-message");

  function setMessage(target, text, tone = "muted") {
    if (!target) return;
    target.textContent = text || "";
    target.className = tone === "error" ? "text-danger" : tone === "success" ? "text-success" : "text-muted";
  }

  function normalizeBase(value) {
    return value.replace(/\/$/, "");
  }

  function getApiBase() {
    const saved = localStorage.getItem(apiBaseKey);
    if (saved && saved.trim() !== "") {
      return normalizeBase(saved.trim());
    }
    return normalizeBase(window.location.origin);
  }

  function setApiBase(value) {
    if (!value) {
      localStorage.removeItem(apiBaseKey);
      return;
    }
    localStorage.setItem(apiBaseKey, normalizeBase(value));
  }

  async function apiFetch(path, options = {}) {
    const url = `${getApiBase()}${path}`;
    const headers = Object.assign({}, options.headers || {});
    const session = localStorage.getItem(sessionKey);
    if (session) {
      headers["X-ADMIN-SESSION"] = session;
    }
    if (options.body && !headers["Content-Type"]) {
      headers["Content-Type"] = "application/json";
    }
    const response = await fetch(url, Object.assign({}, options, { headers }));
    const text = await response.text();
    let data = null;
    try {
      data = text ? JSON.parse(text) : null;
    } catch (err) {
      data = null;
    }
    if (!response.ok) {
      const message = data && data.message ? data.message : `Request failed (${response.status})`;
      throw new Error(message);
    }
    return data;
  }

  async function sendOtp() {
    setMessage(loginMessage, "Отправляем код...");
    try {
      await apiFetch("/v1/admin/otp/send", { method: "POST" });
      setMessage(loginMessage, "Код отправлен. Проверьте Telegram.", "success");
    } catch (err) {
      setMessage(loginMessage, err.message, "error");
    }
  }

  async function verifyOtp() {
    const code = otpInput.value.trim();
    if (!code) {
      setMessage(loginMessage, "Введите код", "error");
      return;
    }
    setMessage(loginMessage, "Проверяем код...");
    try {
      const data = await apiFetch("/v1/admin/otp/verify", {
        method: "POST",
        body: JSON.stringify({ code })
      });
      localStorage.setItem(sessionKey, data.session_token);
      setMessage(loginMessage, "Вход выполнен", "success");
      await showAdmin();
    } catch (err) {
      setMessage(loginMessage, err.message, "error");
    }
  }

  async function loadStats() {
    const windowValue = statWindowSelect.value || "24h";
    const data = await apiFetch(`/v1/admin/stats?window=${encodeURIComponent(windowValue)}`);
    statRequests.textContent = data.total_requests ?? "—";
    statUsers.textContent = data.active_users ?? "—";
    const hours = Math.round((data.window_seconds || 0) / 3600);
    statWindow.textContent = hours ? `${hours} часа` : "—";
  }

  async function loadLogs() {
    const windowValue = statWindowSelect.value || "24h";
    const data = await apiFetch(`/v1/admin/logs?limit=50&window=${encodeURIComponent(windowValue)}`);
    logsTable.innerHTML = "";
    if (!data.items || data.items.length === 0) {
      logsTable.innerHTML = "<tr><td colspan=\"5\" class=\"text-muted\">Нет данных</td></tr>";
      return;
    }
    data.items.forEach((item) => {
      const row = document.createElement("tr");
      const createdAt = item.timestamp ? new Date(item.timestamp).toLocaleString("ru-RU") : "—";
      const resource = [item.resource_type, item.resource_id].filter(Boolean).join(": ");
      row.innerHTML = `
        <td>${createdAt}</td>
        <td>${item.level || "—"}</td>
        <td><small>${resource || "—"}</small></td>
        <td><small>${item.stream_name || "—"}</small></td>
        <td><small>${item.message || "—"}</small></td>
      `;
      logsTable.appendChild(row);
    });
  }

  async function loadKeys() {
    const data = await apiFetch("/v1/admin/keys?include_revoked=true");
    keysTable.innerHTML = "";
    if (!data.items || data.items.length === 0) {
      keysTable.innerHTML = "<tr><td colspan=\"6\" class=\"text-muted\">Нет ключей</td></tr>";
      return;
    }
    data.items.forEach((item) => {
      const row = document.createElement("tr");
      const createdAt = item.CreatedAt ? new Date(item.CreatedAt).toLocaleString("ru-RU") : "—";
      const isRevoked = item.Status === "revoked";
      row.innerHTML = `
        <td><small>${item.APIKeyHash || "—"}</small></td>
        <td>${item.ClientID || "—"}</td>
        <td>${item.Status || "—"}</td>
        <td>${item.RateLimitPerMinute ?? "—"}</td>
        <td>${createdAt}</td>
        <td>${isRevoked ? "" : `<button class=\"btn btn-outline-sand btn-sm\" data-revoke=\"${item.APIKeyHash}\">Удалить</button>`}</td>
      `;
      keysTable.appendChild(row);
    });

    keysTable.querySelectorAll("button[data-revoke]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        const hash = btn.getAttribute("data-revoke");
        if (!hash) return;
        if (!confirm("Удалить пользователя и отозвать ключ?")) return;
        try {
          await apiFetch(`/v1/admin/keys/${hash}`, { method: "DELETE" });
          await loadKeys();
        } catch (err) {
          setMessage(adminMessage, err.message, "error");
        }
      });
    });
  }

  async function createKey() {
    setMessage(createKeyMessage, "Создаем ключ...");
    const payload = {
      client_id: newClientId.value.trim(),
      rate_limit_per_minute: parseInt(newRateLimit.value, 10) || 0,
      status: newStatus.value
    };
    try {
      const data = await apiFetch("/v1/admin/keys", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      setMessage(createKeyMessage, `Новый ключ: ${data.api_key}`, "success");
      await loadKeys();
    } catch (err) {
      setMessage(createKeyMessage, err.message, "error");
    }
  }

  async function loadPrompt() {
    setMessage(promptMessage, "Загружаем...");
    try {
      const data = await apiFetch("/v1/admin/system-prompt");
      promptText.value = data.prompt || "";
      setMessage(promptMessage, `Версия: ${data.version || "—"}`, "success");
    } catch (err) {
      setMessage(promptMessage, err.message, "error");
    }
  }

  async function savePrompt() {
    setMessage(promptMessage, "Сохраняем...");
    try {
      await apiFetch("/v1/admin/system-prompt", {
        method: "POST",
        body: JSON.stringify({ prompt: promptText.value })
      });
      setMessage(promptMessage, "Промт обновлен", "success");
    } catch (err) {
      setMessage(promptMessage, err.message, "error");
    }
  }

  async function showAdmin() {
    loginPanel.classList.add("d-none");
    adminPanel.classList.remove("d-none");
    await refreshAll();
  }

  function showLogin() {
    adminPanel.classList.add("d-none");
    loginPanel.classList.remove("d-none");
  }

  async function refreshAll() {
    setMessage(adminMessage, "Загружаем данные...");
    try {
      await Promise.all([loadStats(), loadLogs(), loadKeys(), loadPrompt()]);
      setMessage(adminMessage, "Данные обновлены", "success");
    } catch (err) {
      setMessage(adminMessage, err.message, "error");
    }
  }

  function logout() {
    localStorage.removeItem(sessionKey);
    showLogin();
  }

  saveApiBaseBtn.addEventListener("click", () => {
    const value = apiBaseInput.value.trim();
    setApiBase(value);
    setMessage(adminMessage, "Адрес сохранен", "success");
  });

  sendOtpBtn.addEventListener("click", sendOtp);
  verifyOtpBtn.addEventListener("click", verifyOtp);
  logoutBtn.addEventListener("click", logout);
  refreshLogsBtn.addEventListener("click", () => loadLogs());
  refreshKeysBtn.addEventListener("click", () => loadKeys());
  createKeyBtn.addEventListener("click", createKey);
  refreshPromptBtn.addEventListener("click", loadPrompt);
  savePromptBtn.addEventListener("click", savePrompt);
  statWindowSelect.addEventListener("change", () => {
    loadStats();
    loadLogs();
  });

  const savedBase = localStorage.getItem(apiBaseKey);
  apiBaseInput.value = savedBase || window.location.origin;

  if (localStorage.getItem(sessionKey)) {
    showAdmin().catch(() => logout());
  }
})();

const escapeHTML = value => String(value ?? "").replace(
  /[&<>"']/g,
  character => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[character],
);

const age = seconds => {
  if (!seconds) return "—";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 48) return `${hours}h`;
  return `${Math.floor(hours / 24)}d`;
};

const table = (headers, rows) => rows.length
  ? `<table><thead><tr>${headers.map(header => `<th>${header}</th>`).join("")}</tr></thead><tbody>${rows.join("")}</tbody></table>`
  : '<div class="empty">Nothing to show.</div>';

function render(state) {
  const services = state.services || [];
  const policies = state.policies || [];
  const pendingUpdates = state.pendingUpdates || {};

  document.querySelector("#serviceCount").textContent = services.length;
  document.querySelector("#clientCount").textContent = services.reduce((count, service) => count + (service.clients?.length || 0), 0);
  document.querySelector("#policyCount").textContent = policies.length;
  document.querySelector("#pendingCount").textContent = Object.values(pendingUpdates).reduce((count, updates) => count + updates.length, 0);
  document.querySelector("#updated").textContent = `Updated ${new Date(state.generatedAt).toLocaleTimeString()}`;

  document.querySelector("#services").innerHTML = table(
    ["Service", "Namespace", "Type", "Active", "Validity", "Clients"],
    services.map(service => `<tr>
      <td><b>${escapeHTML(service.name)}</b><br><span class="muted">${escapeHTML(service.secretName)}</span></td>
      <td>${escapeHTML(service.namespace)}</td>
      <td><span class="pill">${escapeHTML(service.kubeType)}</span></td>
      <td class="${service.active ? "good" : "warn"}">${service.active ? "active" : "inactive"}</td>
      <td>${escapeHTML(service.validity)} days</td>
      <td>${(service.clients || []).map(client => `<span class="pill">${escapeHTML(client.namespace)}/${escapeHTML(client.name)}</span>`).join(" ") || "—"}</td>
    </tr>`),
  );

  document.querySelector("#policies").innerHTML = table(
    ["Policy", "Source", "Age", "Valid"],
    policies.map(policy => `<tr>
      <td><b>${escapeHTML(policy.name)}</b><br><span class="muted">${escapeHTML(policy.namespace)}</span></td>
      <td>${escapeHTML(policy.sourceKind || "legacy")}<br><span class="muted">${escapeHTML(`${policy.sourceNamespace ? `${policy.sourceNamespace}/` : ""}${policy.sourceName || ""}`)}</span></td>
      <td>${age(policy.ageSeconds)}</td>
      <td class="${policy.status === "True" ? "good" : "warn"}">${escapeHTML(policy.status || "—")}</td>
    </tr>`),
  );

  renderEvents(state.events);
  const errorPanel = document.querySelector("#errorPanel");
  errorPanel.hidden = !state.errors?.length;
  document.querySelector("#errors").innerHTML = (state.errors || []).map(error => `<div class="bad">${escapeHTML(error)}</div>`).join("");
}

function renderEvents(events) {
  document.querySelector("#events").innerHTML = (events || [])
    .slice()
    .reverse()
    .map(event => `<div class="event">
      <span class="muted">${new Date(event.time).toLocaleString()}</span>
      <span class="pill">${escapeHTML(event.type)}</span>
      <span>${escapeHTML(event.message)}${event.namespace ? `<br><span class="muted">${escapeHTML(`${event.namespace}${event.name ? `/${event.name}` : ""}`)}</span>` : ""}</span>
    </div>`)
    .join("") || '<div class="empty">Waiting for controller events.</div>';
}

async function refresh() {
  try {
    const response = await fetch("/perikles/v1/state", { cache: "no-store" });
    render(await response.json());
  } catch {
    document.querySelector("#connection").textContent = "● state unavailable";
    document.querySelector("#connection").className = "bad";
  }
}

function connect() {
  const source = new EventSource("/perikles/v1/events");
  source.onopen = () => {
    document.querySelector("#connection").textContent = "● live";
    document.querySelector("#connection").className = "live";
  };
  source.addEventListener("perikles", refresh);
  source.onerror = () => {
    document.querySelector("#connection").textContent = "● reconnecting";
    document.querySelector("#connection").className = "warn";
  };
}

refresh();
connect();
setInterval(refresh, 30000);

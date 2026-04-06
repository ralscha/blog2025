import { RealtimeChannel } from "@supabase/supabase-js";
import { getStableSessionId } from "../src/shared/session-id";
import { renderAppNav } from "../src/shared/app-nav";
import { supabase } from "../src/shared/supabase";
import "../src/styles/base.css";

type PresenceState = {
  id: string;
};

const userId = getStableSessionId("P");
const identity = { id: userId };

const root = document.body;

function requireElement<ElementType extends Element>(selector: string) {
  const element = document.querySelector<ElementType>(selector);

  if (!element) {
    throw new Error(`Missing element: ${selector}`);
  }

  return element;
}

root.innerHTML = `
  <main class="page">
    <header class="header">
      <div>
        <h1>Presence</h1>
        <p>Tracks who is connected to the shared channel.</p>
      </div>
      ${renderAppNav("presence")}
    </header>
    <p class="status" id="presence-status">Connecting...</p>
    <section class="grid">
      <article class="panel stack">
        <div>
          <h2>Your id</h2>
          <p class="value">${identity.id}</p>
        </div>
        <div>
          <h2>Connected users</h2>
          <p class="value" id="presence-count">0</p>
          <p class="meta" id="presence-caption">Waiting for people...</p>
        </div>
      </article>
      <section class="panel">
        <h2>Roster</h2>
        <ul class="list" id="presence-legend"></ul>
      </section>
    </section>
  </main>
`;

const countElement = requireElement<HTMLElement>("#presence-count");
const captionElement = requireElement<HTMLElement>("#presence-caption");
const legendElement = requireElement<HTMLElement>("#presence-legend");
const statusElement = requireElement<HTMLElement>("#presence-status");

function setStatus(message: string, isLive = false) {
  statusElement.textContent = message;
  statusElement.classList.toggle("status-live", isLive);
}

function flattenPresence(channel: RealtimeChannel) {
  const state = channel.presenceState<PresenceState>();

  return Object.values(state)
    .flat()
    .map((entry) => ({ id: entry.id }))
    .sort((left, right) => left.id.localeCompare(right.id));
}

function renderPresence(channel: RealtimeChannel) {
  const users = flattenPresence(channel);

  countElement.textContent = String(users.length);
  captionElement.textContent = users.length === 1 ? "One person is here" : `${users.length} people are here`;

  legendElement.innerHTML = users
    .map((user) => `<li>${user.id}</li>`)
    .join("");
}

const channel = supabase.channel("presence-demo", {
  config: {
    presence: {
      key: identity.id,
    },
  },
});

channel
  .on("presence", { event: "sync" }, () => {
    renderPresence(channel);
  })
  .subscribe(async (status) => {
    setStatus(status === "SUBSCRIBED" ? "Connected" : `Realtime: ${status}`, status === "SUBSCRIBED");

    if (status === "SUBSCRIBED") {
      await channel.track(identity);
      renderPresence(channel);
    }
  });

window.addEventListener("beforeunload", () => {
  void channel.untrack();
  void supabase.removeChannel(channel);
});
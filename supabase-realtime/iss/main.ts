import { supabase } from "../src/shared/supabase";
import { renderAppNav } from "../src/shared/app-nav";
import "../src/styles/base.css";

type RealtimeStatus = "CLOSED" | "CHANNEL_ERROR" | "SUBSCRIBED" | "TIMED_OUT";

type IssPayload = {
  requestedAt?: string;
  latitude?: number;
  longitude?: number;
};

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
        <h1>ISS broadcast</h1>
        <p>Listens for <code>iss-update</code> events on <code>iss-position</code>.</p>
      </div>
      ${renderAppNav("iss")}
    </header>
    <p class="status" id="iss-status">Connecting...</p>
    <section class="grid two">
      <article class="panel">
        <h2>Latitude</h2>
        <p class="value" id="iss-latitude">--</p>
      </article>
      <article class="panel">
        <h2>Longitude</h2>
        <p class="value" id="iss-longitude">--</p>
      </article>
    </section>
  </main>
`;

const latitudeElement = requireElement<HTMLElement>("#iss-latitude");
const longitudeElement = requireElement<HTMLElement>("#iss-longitude");
const statusElement = requireElement<HTMLElement>("#iss-status");

function setStatus(message: string, isLive = false) {
  statusElement.textContent = message;
  statusElement.classList.toggle("status-live", isLive);
}

function formatCoordinate(value?: number) {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return "--";
  }

  return value.toFixed(4);
}

function renderPayload(payload: IssPayload) {
  latitudeElement.textContent = formatCoordinate(payload.latitude);
  longitudeElement.textContent = formatCoordinate(payload.longitude);

  const requestedAt = payload.requestedAt
    ? new Date(payload.requestedAt).toLocaleTimeString()
    : "just now";

  setStatus(`Updated ${requestedAt}`, true);
}

const channel = supabase
  .channel("iss-position", {
    config: {
      private: false,
    },
  })
  .on("broadcast", { event: "iss-update" }, (event: { payload: IssPayload }) => {
    const { payload } = event;
    renderPayload(payload as IssPayload);
  })
  .subscribe((status: RealtimeStatus) => {
    setStatus(status === "SUBSCRIBED" ? "Listening" : `Realtime: ${status}`, status === "SUBSCRIBED");
  });

window.addEventListener("beforeunload", () => {
  void supabase.removeChannel(channel);
});
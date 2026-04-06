import { supabase } from "../src/shared/supabase";
import { renderAppNav } from "../src/shared/app-nav";
import "../src/styles/base.css";

type RealtimeStatus = "CLOSED" | "CHANNEL_ERROR" | "SUBSCRIBED" | "TIMED_OUT";

type Todo = {
  id: number;
  task: string;
  done: boolean;
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
        <h1>Todos</h1>
        <p>Reads and listens to <code>public.todos</code>.</p>
      </div>
      ${renderAppNav("todos")}
    </header>
    <p class="status" id="todos-status">Connecting...</p>
    <section class="grid">
      <form class="form" id="todo-form">
        <input id="todo-input" name="task" maxlength="120" placeholder="New todo" autocomplete="off" />
        <button type="submit">Add</button>
      </form>
      <div class="panel">
        <div class="todo-list" id="todo-list"></div>
      </div>
    </section>
  </main>
`;

const formElement = requireElement<HTMLFormElement>("#todo-form");
const inputElement = requireElement<HTMLInputElement>("#todo-input");
const listElement = requireElement<HTMLElement>("#todo-list");
const statusElement = requireElement<HTMLElement>("#todos-status");

function setStatus(message: string, isLive = false) {
  statusElement.textContent = message;
  statusElement.classList.toggle("status-live", isLive);
}

async function loadTodos() {
  const { data, error } = await supabase
    .from("todos")
    .select("id, task, done")
    .order("inserted_at", { ascending: false });

  if (error) {
    setStatus(error.message);
    return;
  }

  renderTodos((data ?? []) as Todo[]);
}

function renderTodos(todos: Todo[]) {
  if (todos.length === 0) {
    listElement.innerHTML = `<p class="hint">No todos yet.</p>`;
    return;
  }

  listElement.innerHTML = todos
    .map((todo) => {
      return `
        <label class="todo-item ${todo.done ? "done" : ""}" data-id="${todo.id}">
          <input class="todo-check" type="checkbox" ${todo.done ? "checked" : ""} aria-label="Mark ${todo.task} as done" />
          <span class="grow">${escapeHtml(todo.task)}</span>
          <button class="icon-button" type="button" aria-label="Delete ${todo.task}">Delete</button>
        </label>
      `;
    })
    .join("");
}

function escapeHtml(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

formElement.addEventListener("submit", async (event) => {
  event.preventDefault();

  const task = inputElement.value.trim();

  if (!task) {
    return;
  }

  inputElement.disabled = true;

  const { error } = await supabase.from("todos").insert({ task });

  inputElement.disabled = false;

  if (error) {
    setStatus(error.message);
    return;
  }

  inputElement.value = "";
  inputElement.focus();
  setStatus("Saved", true);
});

listElement.addEventListener("change", async (event) => {
  const target = event.target;

  if (!(target instanceof HTMLInputElement) || !target.classList.contains("todo-check")) {
    return;
  }

  const item = target.closest<HTMLElement>(".todo-item");
  const id = item?.dataset.id;

  if (!id) {
    return;
  }

  const { error } = await supabase
    .from("todos")
    .update({ done: target.checked })
    .eq("id", Number(id));

  if (error) {
    setStatus(error.message);
    target.checked = !target.checked;
  }
});

listElement.addEventListener("click", async (event) => {
  const target = event.target;

  if (!(target instanceof HTMLButtonElement) || !target.classList.contains("icon-button")) {
    return;
  }

  const item = target.closest<HTMLElement>(".todo-item");
  const id = item?.dataset.id;

  if (!id) {
    return;
  }

  const { error } = await supabase.from("todos").delete().eq("id", Number(id));

  if (error) {
    setStatus(error.message);
  }
});

const channel = supabase
  .channel("public:todos")
  .on(
    "postgres_changes",
    { event: "*", schema: "public", table: "todos" },
    async () => {
      await loadTodos();
    },
  )
  .subscribe(async (status: RealtimeStatus) => {
    setStatus(status === "SUBSCRIBED" ? "Connected" : `Realtime: ${status}`, status === "SUBSCRIBED");

    if (status === "SUBSCRIBED") {
      await loadTodos();
    }
  });

window.addEventListener("beforeunload", () => {
  void supabase.removeChannel(channel);
});
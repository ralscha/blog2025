export type DemoPage = "iss" | "presence" | "todos";

const pages: Array<{ id: DemoPage; href: string; label: string }> = [
  { id: "iss", href: "/iss/", label: "ISS" },
  { id: "presence", href: "/presence/", label: "Presence" },
  { id: "todos", href: "/todos/", label: "Todos" },
];

export function renderAppNav(activePage: DemoPage) {
  return `
    <nav class="nav" aria-label="Realtime demos">
      ${pages
        .map(
          (page) => `
            <a href="${page.href}" ${page.id === activePage ? 'aria-current="page"' : ""}>${page.label}</a>
          `,
        )
        .join("")}
    </nav>
  `;
}
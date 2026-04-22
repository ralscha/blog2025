const loginButton = document.getElementById("loginButton");
const protectedButton = document.getElementById("protectedButton");
const logoutButton = document.getElementById("logoutButton");
const sessionOutput = document.getElementById("sessionOutput");
const resourceOutput = document.getElementById("resourceOutput");

loginButton.addEventListener("click", async () => {
    const state = await loadState();
    window.location.assign(state.loginUrl);
});

protectedButton.addEventListener("click", async () => {
    resourceOutput.textContent = "Calling protected resource...";

    const response = await fetch("/api/protected", {
        credentials: "same-origin"
    });

    const payload = await response.json();
    resourceOutput.textContent = JSON.stringify(payload, null, 2);

    if (response.status === 401) {
        await renderState();
    }
});

logoutButton.addEventListener("click", async () => {
    logoutButton.disabled = true;
    resourceOutput.textContent = "Signing out...";

    try {
        const response = await fetch("/api/logout", {
            method: "POST",
            credentials: "same-origin"
        });

        if (response.redirected) {
            window.location.assign(response.url);
            return;
        }

        const payload = await response.json();
        window.location.assign(payload.redirectUrl || "/signed-out.html");
    } catch (error) {
        resourceOutput.textContent = "Logout failed. Reload and try again.";
        logoutButton.disabled = false;
    }
});

async function renderState() {
    const state = await loadState();
    sessionOutput.textContent = JSON.stringify(state, null, 2);

    const loggedIn = Boolean(state.authenticated);
    loginButton.disabled = loggedIn;
    protectedButton.disabled = !loggedIn;
    logoutButton.disabled = !loggedIn;
}

async function loadState() {
    const response = await fetch("/api/me", {
        credentials: "same-origin"
    });

    return response.json();
}

renderState();
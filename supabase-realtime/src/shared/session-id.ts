const storagePrefix = "supabase-realtime-demo";

function randomSuffix() {
  return crypto.randomUUID().slice(0, 6).toUpperCase();
}

export function getStableSessionId(scope: string) {
  const storageKey = `${storagePrefix}:${scope}`;
  const existing = sessionStorage.getItem(storageKey);

  if (existing) {
    return existing;
  }

  const created = `${scope.toUpperCase()}-${randomSuffix()}`;
  sessionStorage.setItem(storageKey, created);
  return created;
}

export function hashText(input: string) {
  let hash = 0;

  for (let index = 0; index < input.length; index += 1) {
    hash = (hash << 5) - hash + input.charCodeAt(index);
    hash |= 0;
  }

  return Math.abs(hash);
}
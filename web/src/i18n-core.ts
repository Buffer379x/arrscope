export type Catalog = {
  locale: string;
  name: string;
  messages: Record<string, string>;
};
export type Parameters = Record<string, string | number>;
export function translate(
  catalog: Catalog,
  fallback: Catalog,
  key: string,
  parameters: Parameters = {},
) {
  const message = catalog.messages[key] ?? fallback.messages[key] ?? key;
  return message.replace(/\{([a-zA-Z][\w]*)\}/g, (match, name: string) =>
    Object.hasOwn(parameters, name) ? String(parameters[name]) : match,
  );
}
export function resolveLocale(
  preferred: readonly string[],
  supported: readonly string[],
) {
  for (const preference of preferred) {
    const exact = supported.find(
      (locale) => locale.toLowerCase() === preference.toLowerCase(),
    );
    if (exact) return exact;
    const base = supported.find(
      (locale) =>
        locale.toLowerCase() === preference.toLowerCase().split("-")[0],
    );
    if (base) return base;
  }
  return "en";
}

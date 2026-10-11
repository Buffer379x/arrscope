import { createContext, type ComponentChildren } from "preact";
import { useContext, useEffect, useState } from "preact/hooks";
import {
  resolveLocale,
  translate,
  type Catalog,
  type Parameters,
} from "./i18n-core";
// A new JSON catalog is discovered automatically; no component changes needed.
const modules = import.meta.glob<Catalog>("./locales/*.json", {
  eager: true,
  import: "default",
});
export const catalogs = Object.values(modules).sort((a, b) =>
  a.name.localeCompare(b.name),
);
const fallback = catalogs.find((catalog) => catalog.locale === "en")!;
let active = fallback;
export const t = (key: string, parameters?: Parameters) =>
  translate(active, fallback, key, parameters);
export const getLocale = () => active.locale;
function initialLocale() {
  let saved: string | null = null;
  try {
    saved = localStorage.getItem("scopearr-language") || localStorage.getItem("arrscope-language");
  } catch {}
  return resolveLocale(
    saved ? [saved] : navigator.languages,
    catalogs.map((catalog) => catalog.locale),
  );
}
const Context = createContext({
  locale: "en",
  setLocale: (_locale: string) => {},
});
export function I18nProvider({ children }: { children: ComponentChildren }) {
  const [locale, setLocale] = useState(initialLocale);
  active = catalogs.find((catalog) => catalog.locale === locale) || fallback;
  useEffect(() => {
    document.documentElement.lang = active.locale;
    document.title = t("Scopearr · Your library at a glance");
    document
      .querySelector('meta[name="description"]')
      ?.setAttribute(
        "content",
        t("Your movie and TV library. Clearly filtered."),
      );
    try {
      localStorage.setItem("scopearr-language", locale);
    } catch {}
  }, [locale]);
  return (
    <Context.Provider value={{ locale, setLocale }}>
      {children}
    </Context.Provider>
  );
}
export function useI18n() {
  return useContext(Context);
}
export function LanguageSelect() {
  const { locale, setLocale } = useI18n();
  return (
    <label class="language-select">
      <span>{t("Interface language")}</span>
      <select
        aria-label={t("Interface language")}
        value={locale}
        onChange={(event) => setLocale(event.currentTarget.value)}
      >
        {catalogs.map((catalog) => (
          <option key={catalog.locale} value={catalog.locale}>
            {catalog.name}
          </option>
        ))}
      </select>
    </label>
  );
}

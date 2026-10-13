// Locally bundled palette adaptations; layout and media-tag colors are independent.
export const interfaceThemes = ["default", "nord", "catppuccin"] as const;
export type InterfaceTheme = (typeof interfaceThemes)[number];
export type ThemeMode = "dark" | "light";
export const themeLabels = {
  default: "Default",
  nord: "Nordic",
  catppuccin: "Catppuccin",
};
export const themePalettes = {
  nord: {
    dark: {
      bg: "#2e3440",
      panel: "#3b4252",
      panel2: "#434c5e",
      line: "#5e6a80",
      text: "#eceff4",
      muted: "#c0c8d8",
      accent: "#88c0d0",
      "accent-bg": "#344955",
      "accent-text": "#2e3440",
    },
    light: {
      bg: "#eceff4",
      panel: "#ffffff",
      panel2: "#e5e9f0",
      line: "#b4bfce",
      text: "#2e3440",
      muted: "#4c566a",
      accent: "#42668e",
      "accent-bg": "#dce8f2",
      "accent-text": "#ffffff",
    },
  },
  catppuccin: {
    dark: {
      bg: "#1e1e2e",
      panel: "#313244",
      panel2: "#45475a",
      line: "#6c7086",
      text: "#cdd6f4",
      muted: "#bac2de",
      accent: "#89b4fa",
      "accent-bg": "#303d5c",
      "accent-text": "#1e1e2e",
    },
    light: {
      bg: "#eff1f5",
      panel: "#ffffff",
      panel2: "#e6e9ef",
      line: "#a3a7b9",
      text: "#4c4f69",
      muted: "#5c5f77",
      accent: "#1757cd",
      "accent-bg": "#e1e9fa",
      "accent-text": "#ffffff",
    },
  },
};
export function applyInterfaceTheme(
  root: HTMLElement,
  value: string | undefined,
  mode: string,
) {
  const theme = interfaceThemes.includes(value as InterfaceTheme)
    ? (value as InterfaceTheme)
    : "default";
  root.dataset.palette = theme;
  const keys = Object.keys(themePalettes.nord.dark);
  for (const key of keys) root.style.removeProperty("--" + key);
  if (theme !== "default") {
    for (const [key, color] of Object.entries(
      themePalettes[theme][mode === "light" ? "light" : "dark"],
    ))
      root.style.setProperty("--" + key, color);
  }
}

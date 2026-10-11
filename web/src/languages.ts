// Aliases are display-only. Never substitute these for stored API/filter values.
export const audioLanguages = [
  {
    name: "German",
    flag: "de",
    aliases: ["Deutsch", "German", "de", "deu", "ger"],
  },
  {
    name: "English",
    flag: "gb",
    aliases: ["Englisch", "English", "en", "eng"],
  },
  {
    name: "Spanish",
    flag: "es",
    aliases: ["Spanisch", "Spanish", "Español", "es", "spa"],
  },
  {
    name: "Italian",
    flag: "it",
    aliases: ["Italienisch", "Italian", "Italiano", "it", "ita"],
  },
  {
    name: "French",
    flag: "fr",
    aliases: ["Französisch", "French", "Français", "fr", "fra", "fre"],
  },
  {
    name: "Japanese",
    flag: "jp",
    aliases: ["Japanisch", "Japanese", "ja", "jpn"],
  },
  {
    name: "Portuguese",
    flag: "pt",
    aliases: ["Portugiesisch", "Portuguese", "Português", "pt", "por"],
  },
  {
    name: "Dutch",
    flag: "nl",
    aliases: ["Niederländisch", "Dutch", "nl", "nld", "dut"],
  },
  {
    name: "Russian",
    flag: "ru",
    aliases: ["Russisch", "Russian", "ru", "rus"],
  },
  { name: "Polish", flag: "pl", aliases: ["Polnisch", "Polish", "pl", "pol"] },
];
export function audioLanguage(value: string) {
  return audioLanguages.find((language) =>
    language.aliases.some(
      (alias) => alias.toLowerCase() === value.trim().toLowerCase(),
    ),
  );
}
export function audioLanguageKey(value: string) {
  return audioLanguage(value)?.name || value || "Unknown";
}

// Compare known aliases for UI selection without rewriting saved filter values.
export function sameAudioLanguage(left: string, right: string) {
  return (
    left === right ||
    (!!audioLanguage(left) &&
      audioLanguage(left)?.name === audioLanguage(right)?.name)
  );
}

import { colorStyle } from "./color";
import { createContext } from "preact";
import { useContext } from "preact/hooks";
import type { Results, Service } from "./types";
export const labels: Record<string, string> = {
  resolution: "Auflösung",
  codec: "Codec / Encoder",
  family: "Codec-Familie",
  source: "Source",
  processing: "Verarbeitung",
  hdr: "HDR",
  bitDepth: "Bit-Tiefe",
  audio: "Audio-Codec",
  channels: "Kanäle",
  language: "Sprache",
};
export const tableColumns = [
  ["title", "Titel / Episode"],
  ["year", "Jahr"],
  ["resolution", "Auflösung"],
  ["codec", "Codec"],
  ["source", "Source"],
  ["size", "Größe"],
  ["hdr", "HDR"],
  ["bitDepth", "Bit-Tiefe"],
  ["audio", "Audio"],
  ["channels", "Kanäle"],
  ["language", "Sprache"],
];
export type ConnectionService = Service | "emby";
export const embyMatchLabels: Record<string, string> = {
  missing: "In Emby nicht eindeutig gefunden",
  missing_size: "Dateigröße fehlt für die Zuordnung",
  ambiguous: "Mehrdeutige Datei- oder Tonspurzuordnung",
  no_tracks: "Emby liefert keine Tonspurmetadaten",
};
export type FilterOperator = "eq" | "not" | "all" | "notAll";
export const operatorLabels: Record<FilterOperator, string> = {
  eq: "ist",
  not: "ist nicht",
  all: "enthält alle",
  notAll: "enthält nicht alle",
};
export const compoundFields = ["hdr", "audio", "language"];
export type FilterUpdate = string[] | ((current: string[]) => string[]);
export const defaultColumnOrder = tableColumns.map(([field]) => field);
export const columnLabels = Object.fromEntries(tableColumns);
export const defaultColumns = [
  "title",
  "year",
  "resolution",
  "codec",
  "source",
];
export const emptyResults: Results = {
  files: [],
  total: 0,
  displayTotal: 0,
  page: 1,
  pageSize: 100,
};
export const count = (n: number) => new Intl.NumberFormat("de-DE").format(n);
export const bytes = (n: number) =>
  n
    ? new Intl.NumberFormat("de-DE", { maximumFractionDigits: 1 }).format(
        n / 1024 ** 3,
      ) + " GiB"
    : "Unbekannt";
export const date = (s: string | null | undefined) =>
  s
    ? new Date(s).toLocaleString("de-DE", {
        day: "2-digit",
        month: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      })
    : "Noch nicht synchronisiert";
export const TagColors = createContext<Record<string, string>>({});
export const colorFields = [
  "resolution",
  "codec",
  "source",
  "processing",
  "hdr",
  "audio",
];
export const defaultColor = (field: string, value: string) =>
  field === "resolution"
    ? value === "2160p" || value === "4320p"
      ? "#b7a0e8"
      : value === "720p"
        ? "#a6c582"
        : "#85bce5"
    : (
        {
          codec: "#72c9a4",
          source: "#e4c16d",
          processing: "#d894b0",
          hdr: "#b7a0e8",
          audio: "#87bad3",
        } as Record<string, string>
      )[field] || "#b0bbc5";
export function Tag({
  value,
  kind = "source",
  title,
  colors,
}: {
  colors?: Record<string, string>;
  value: string;
  kind?: string;
  title?: string;
}) {
  const inherited = useContext(TagColors);
  const color = (colors || inherited)[
    kind.split(" ")[0] + ":" + (value || "__unknown__")
  ];
  return (
    <span
      style={colorStyle(color)}
      class={"tag " + kind + (!value ? " unknown" : "")}
      title={title}
    >
      {value || "Unbekannt"}
    </span>
  );
}

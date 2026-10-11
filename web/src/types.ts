export type Service = "radarr" | "sonarr";
export type Connection = {
  enabled: boolean;
  url: string;
  apiKey?: string;
  hasKey: boolean;
  clearKey?: boolean;
  pathMappings?: string;
  interval: number;
};
export type Settings = {
  columnOrder: Record<Service, string[]>;
  visibleColumns: Record<Service, string[]>;
  warningColors: Record<string, string>;
  tagColors: Record<string, string>;
  tagTextColors: Record<string, string>;
  radarr: Connection;
  sonarr: Connection;
  emby: Connection;
  seasonMode: "matches" | "season";
};
export type Evidence = {
  filename: string;
  api: string;
  emby?: string;
  origin: string;
  conflict: boolean;
  rules?: string[];
};
export type CustomTag = { id: number; name: string };
export type CustomSelection = {
  total: number;
  counts: Record<string, number>;
  tags: CustomTag[];
};
export type MediaFile = {
  custom: CustomTag[];
  id: number;
  service: Service;
  mediaId: number;
  title: string;
  series: string;
  season: number;
  episodes: { id: number; number: number; title: string; year: number }[];
  years: number[];
  filename: string;
  resolution: string;
  codec: string;
  family: string;
  source: string;
  processing: string;
  hdr: string;
  bitDepth: string;
  audio: string;
  audioStreamCount: number;
  audioTracks?: {
    index: number;
    language: string;
    codec: string;
    channels: number;
    channelLayout: string;
    title: string;
    default: boolean;
  }[];
  embyMatch?: string;
  channels: string;
  language: string;
  size: number;
  conflict: boolean;
  ruleConflict: boolean;
  recognitionConflicts?: RecognitionConflict[];
  evidence: Record<string, Evidence>;
  matched: boolean;
  unknownTraits?: string[];
  metadataIncomplete: boolean;
};
export type Results = {
  files: MediaFile[];
  total: number;
  displayTotal: number;
  page: number;
  pageSize: number;
};
export type SyncStatus = {
  running: boolean;
  queued: boolean;
  lastAttempt: string | null;
  lastSuccess: string | null;
  error: string;
  count: number;
};
export type Status = {
  initialized: boolean;
  authenticated: boolean;
  version: string;
  demo: boolean;
  recognitionRevision?: string;
  csrf?: string;
  sync?: Record<Service, SyncStatus>;
};
export type Preset = {
  id: number;
  name: string;
  service: Service;
  query: string;
};

export type Statistics = {
  languageTotals: Record<string, number>;
  movies: number;
  series: number;
  episodes: number;
  files: number;
  emby: Record<string, number>;
  incomplete: number;
  distributions: Record<
    string,
    Record<string, { value: string; count: number }[]>
  >;
};
export type FilmSort = { field: string; direction: "asc" | "desc" };

export type RecognitionRule = {
  id: string;
  name: string;
  field: string;
  match: "token" | "regex";
  pattern: string;
  value: string;
  mode: "direct" | "fallback" | "override";
  priority: number;
  enabled: boolean;
  consume: boolean;
  unless?: string;
};
export type RecognitionConfig = { rules: RecognitionRule[]; revision: string };
export type RecognitionConflict = {
  field: string;
  values: string[];
  rules: string[];
  ruleNames: string[];
};
export type RecognitionResult = {
  values: Record<string, string>;
  matches: Record<string, string[]>;
  origins: Record<string, string>;
  conflicts: RecognitionConflict[];
};

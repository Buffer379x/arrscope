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
  tagColors: Record<string, string>;
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
};
export type MediaFile = {
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

export interface Episode {
  id: number;
  seriesTitle: string;
  seasonNumber: number;
  episodeNumber: number;
  title: string;
  monitored: boolean;
}

export interface MissingResult {
  episodes: Episode[];
  totalCount: number;
  page: number;
  pageSize: number;
}

export interface SonarrInstance {
  id: string;
  name: string;
  url: string;
  api_key_set: boolean;
  api_key_hint: string;
}

export interface InstanceSummary {
  id: string;
  name: string;
  missingCount: number;
  error?: string;
}

export interface VersionInfo {
  current: string;
  latest: string;
  hasUpdate: boolean;
}

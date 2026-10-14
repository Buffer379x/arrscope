export const mediaStateFilters: Record<string, string> = {
  mediaMatched: "Matched",
  mediaMissing: "Not found",
  mediaAmbiguous: "Ambiguous",
  mediaConflicting: "Conflicting media server metadata",
  mediaNoTracks: "Audio tracks missing",
  mediaPending: "Awaiting media server matching",
  mediaUnconfigured: "Without media server enrichment",
};
export const seasonTechnicalHelp =
  "Compares resolution, codec family, audio codecs and channels with the strict majority of the entire cached season. At least three known episodes are needed, with at least two agreeing episodes and more than half of all episodes supporting a value. Only known differing values are highlighted. Unknown values and ties are not outliers. Multi-episode files count by episode. Active filters do not change the comparison.";
export const seasonLanguageHelp =
  "Compares the complete known audio-language set with the strict majority of the entire cached season. For example, German + English in nine episodes and English alone in one episode marks that episode. Extra languages can also differ. At least three episodes and a strict majority are required. Partially unknown languages and ties are not outliers. Active filters do not change the comparison.";

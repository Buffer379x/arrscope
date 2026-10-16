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

export const mediaStateHelp: Record<string, string> = {
  mediaMatched:
    "At least one enabled media server uniquely matches this file by name and size. Excluding this selects files without a successful match.",
  mediaMissing:
    "No matching file was found, or a required file size is missing. Excluding this removes those cases.",
  mediaAmbiguous:
    "Multiple files or audio tracks prevent a unique match. Excluding this removes ambiguous matches.",
  mediaConflicting:
    "Enabled media servers report different audio-track inventories for the same file. Excluding this removes these disagreements.",
  mediaNoTracks:
    "A matched media-server item has no usable audio-track metadata. Excluding this removes these items.",
  mediaPending:
    "The file is awaiting a media-server match after a cache or file update. Excluding this removes pending items.",
  mediaUnconfigured:
    "The file has no media-server enrichment configured. Excluding this removes those items.",
};

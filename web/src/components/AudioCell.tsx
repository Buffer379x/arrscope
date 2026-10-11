import { Info } from "lucide-preact";
import { t, useI18n } from "../i18n";
import type { MediaFile } from "../types";
import { Tag } from "../ui";
import { LanguageFlag, LanguageFlags } from "./LanguageFlag";

export function AudioCell({
  file,
  open,
}: {
  file: MediaFile;
  open: () => void;
}) {
  useI18n();
  const tracks = file.audioTracks || [];
  const mapping = tracks.length > 0;
  const knownLanguages = new Set(tracks.map((track) => track.language));
  const knownCodecs = new Set(tracks.map((track) => track.codec));
  const unassignedLanguages = file.language
    .split(", ")
    .filter((value) => value && !knownLanguages.has(value));
  const unassignedCodecs = file.audio
    .split(" + ")
    .filter((value) => value && !knownCodecs.has(value));
  const hasUnassigned =
    mapping && (unassignedLanguages.length > 0 || unassignedCodecs.length > 0);
  return (
    <div class="audio-cell">
      {mapping ? (
        <div class="track-language-list">
          {tracks.map((track) => (
            <div key={track.index}>
              <LanguageFlag value={track.language} />
              <Tag value={track.codec} kind="audio" />
            </div>
          ))}
        </div>
      ) : (
        <>
          <LanguageFlags value={file.language} />
          <div class="source-tags">
            {(file.audio ? file.audio.split(" + ") : [""]).map((value) => (
              <Tag key={value} value={value} kind="audio" />
            ))}
          </div>
        </>
      )}
      {hasUnassigned && (
        <div class="unassigned-audio">
          {!!unassignedLanguages.length && (
            <LanguageFlags value={unassignedLanguages.join(", ")} />
          )}
          <div class="source-tags">
            {unassignedCodecs.map((value) => (
              <Tag key={value} value={value} kind="audio" />
            ))}
          </div>
          <small class="quiet">
            {t("Additional reported values without track attribution")}
          </small>
        </div>
      )}
      <button
        class="text-button audio-mapping"
        onClick={open}
        aria-label={t("Audio track information: {title}", {
          title: file.title,
        })}
        title={t(
          mapping
            ? "Audio tracks from Emby. Open file details."
            : "Language → codec mapping unavailable. Open file details.",
        )}
      >
        <Info size={13} />
        {!mapping && <small>{t("Track mapping unavailable")}</small>}
      </button>
    </div>
  );
}

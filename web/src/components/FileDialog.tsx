import { Modal } from "./Modal";

import { AlertTriangle, FileVideo2, Info } from "lucide-preact";
import type { MediaFile } from "../types";

import { embyMatchLabels, labels, Tag } from "../ui";
export function FileDialog({
  file,
  close,
}: {
  file: MediaFile;
  close: () => void;
}) {
  return (
    <Modal title="Datei & Metadaten" close={close} wide>
      <h3>
        {file.series && file.series + " · "}
        {file.title}
      </h3>
      <div class="filename-box">
        <FileVideo2 size={20} />
        <code>{file.filename}</code>
      </div>
      <p class="quiet">
        Dateiname zuerst. Eindeutige Emby-Tonspuren liefern präzisere Audio- und
        Sprachangaben. Abweichungen sind markiert.
      </p>
      {file.metadataIncomplete && (
        <p class="notice">
          <Info size={15} /> Metadaten unvollständig. Nicht erkannte
          Eigenschaften sind kein sicherer Nachweis für deren Abwesenheit.
        </p>
      )}
      <section class="audio-information" aria-label="Tonspurinformationen">
        <h4>Tonspuren</h4>
        {file.audioTracks?.length ? (
          <>
            <p class="quiet">
              Zuordnung aus vorhandenen Emby-Metadaten. Die Datei wurde anhand
              ihres Namens und ihrer Größe eindeutig zugeordnet.
            </p>
            <ul class="audio-track-list">
              {file.audioTracks.map((track) => (
                <li key={track.index}>
                  <span>#{track.index}</span>
                  <strong>{track.language || "Unbekannt"}</strong>
                  <Tag value={track.codec} kind="audio" />
                  <span>
                    {track.channelLayout ||
                      (track.channels
                        ? track.channels + " Kanäle"
                        : "Kanäle unbekannt")}
                  </span>
                  <span>{track.default ? "Standard" : ""}</span>
                  {track.title && <small>{track.title}</small>}
                </li>
              ))}
            </ul>
          </>
        ) : (
          <>
            <p>
              Erkannte Sprachen: <b>{file.language || "Unbekannt"}</b> ·
              Erkannte Codecs: <b>{file.audio || "Unbekannt"}</b>
              {file.audioStreamCount > 0 && (
                <> · {file.audioStreamCount} Tonspuren laut API</>
              )}
            </p>
            <p class="quiet">
              {file.embyMatch && embyMatchLabels[file.embyMatch] && (
                <>
                  <b>{embyMatchLabels[file.embyMatch]}.</b>{" "}
                </>
              )}
              Zuordnung Sprache → Codec nicht verfügbar. Radarr und Sonarr
              liefern keine Eigenschaften pro Tonspur. Emby lässt sich in den
              Einstellungen ergänzend verbinden. Weitere Codecs können in den
              Angaben fehlen.
            </p>
          </>
        )}
        <p class="quiet">
          Die Spalte „Kanäle“ verwendet weiterhin die Dateinamenangabe oder die
          primäre Tonspur aus Radarr/Sonarr.
        </p>
      </section>
      <div class="evidence-table">
        <div class="evidence-head">
          <span>Eigenschaft</span>
          <span>Dateiname</span>
          <span>Radarr / Sonarr</span>
          <span>Emby</span>
          <span>Verwendet</span>
        </div>
        {Object.entries(file.evidence).map(([k, e]) => (
          <div class={e.conflict ? "evidence-conflict" : ""}>
            <b>{labels[k] || k}</b>
            <span data-label="Dateiname">{e.filename || "—"}</span>
            <span data-label="Radarr / Sonarr">{e.api || "—"}</span>
            <span data-label="Emby">{e.emby || "—"}</span>
            <span data-label="Verwendet">
              {e.origin === "filename"
                ? "Dateiname"
                : e.origin === "api"
                  ? "API"
                  : e.origin === "emby"
                    ? "Emby"
                    : e.origin === "combined"
                      ? "Ergänzt"
                      : "Unbekannt"}
              {e.conflict && <AlertTriangle size={14} />}
            </span>
          </div>
        ))}
      </div>
    </Modal>
  );
}

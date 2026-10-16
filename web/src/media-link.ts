import type { Connection, MediaFile, Settings } from "./types";

export function mediaEntryLink(
  connection: Connection | undefined,
  provider: string,
  match: NonNullable<MediaFile["mediaServers"]>[string] | undefined,
): string | undefined {
  if (!connection?.enabled || !["emby", "jellyfin"].includes(provider)) return;
  const identity = match?.link;
  if (
    !identity ||
    !/^[A-Za-z0-9_-]{1,128}$/.test(identity.itemId) ||
    !/^[A-Za-z0-9_-]{1,128}$/.test(identity.serverId)
  )
    return;
  try {
    const base = new URL(connection.url);
    if (
      !["http:", "https:"].includes(base.protocol) ||
      base.username ||
      base.password ||
      base.search ||
      base.hash
    )
      return;
    return (
      base.href.replace(/\/+$/, "") +
      (provider === "emby" ? "/web/index.html#!/item?" : "/web/#/details?") +
      new URLSearchParams({ id: identity.itemId, serverId: identity.serverId })
    );
  } catch {
    /* Invalid addresses cannot become navigable links. */
  }
}
export function mediaEntryChoices(settings: Settings | null, file: MediaFile) {
  return (["emby", "jellyfin"] as const)
    .filter((provider) => settings?.[provider].enabled)
    .map((provider) => ({
      provider,
      label: provider === "emby" ? "Emby" : "Jellyfin",
      href: mediaEntryLink(
        settings?.[provider],
        provider,
        file.mediaServers?.[provider],
      ),
    }));
}

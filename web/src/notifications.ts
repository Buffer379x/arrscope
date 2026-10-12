export type NotificationKind = "success" | "warning" | "error";
export function notify(message: string, kind: NotificationKind = "success") {
  window.dispatchEvent(
    new CustomEvent("scopearr-notification", { detail: { message, kind } }),
  );
}

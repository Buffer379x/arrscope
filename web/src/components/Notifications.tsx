import { useEffect, useRef, useState } from "preact/hooks";
import { createPortal } from "preact/compat";
import { Check, AlertTriangle, X } from "lucide-preact";
import { t, useI18n } from "../i18n";
import type { NotificationKind } from "../notifications";
type Item = { id: number; message: string; kind: NotificationKind };
function Toast({ item, dismiss }: { item: Item; dismiss: () => void }) {
  const remaining = useRef(item.kind === "success" ? 5000 : 8000);
  const started = useRef(0);
  const timer = useRef<ReturnType<typeof setTimeout>>();
  const hovering = useRef(false),
    focused = useRef(false);
  const resume = () => {
    if (hovering.current || focused.current || timer.current) return;
    started.current = Date.now();
    timer.current = setTimeout(dismiss, remaining.current);
  };
  const pause = () => {
    if (!timer.current) return;
    clearTimeout(timer.current);
    timer.current = undefined;
    remaining.current = Math.max(
      0,
      remaining.current - (Date.now() - started.current),
    );
  };
  useEffect(() => {
    resume();
    return pause;
  }, []);
  return (
    <div
      class={"toast toast-" + item.kind}
      role={item.kind === "error" ? "alert" : "status"}
      onMouseEnter={() => {
        hovering.current = true;
        pause();
      }}
      onMouseLeave={() => {
        hovering.current = false;
        resume();
      }}
      onFocusIn={() => {
        focused.current = true;
        pause();
      }}
      onFocusOut={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node | null)) {
          focused.current = false;
          resume();
        }
      }}
    >
      {item.kind === "success" ? (
        <Check size={18} />
      ) : (
        <AlertTriangle size={18} />
      )}
      <span>{item.message}</span>
      <button
        type="button"
        class="icon-button"
        onClick={dismiss}
        aria-label={t("Dismiss notification")}
      >
        <X size={16} />
      </button>
    </div>
  );
}
export function Notifications() {
  useI18n();
  const [items, setItems] = useState<Item[]>([]);
  const next = useRef(0);
  useEffect(() => {
    const receive = (event: Event) => {
      const detail = (event as CustomEvent).detail;
      setItems((items) => {
        if (
          items.some(
            (item) =>
              item.message === detail.message && item.kind === detail.kind,
          )
        )
          return items;
        return [...items.slice(-3), { ...detail, id: ++next.current }];
      });
    };
    window.addEventListener("scopearr-notification", receive);
    return () => window.removeEventListener("scopearr-notification", receive);
  }, []);
  return createPortal(
    <div class="toast-region" aria-label={t("Notifications")}>
      {items.map((item) => (
        <Toast
          key={item.id}
          item={item}
          dismiss={() =>
            setItems((items) => items.filter((value) => value.id !== item.id))
          }
        />
      ))}
    </div>,
    document.querySelector("dialog[open]") || document.body,
  );
}

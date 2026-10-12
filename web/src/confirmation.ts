export type Confirmation = {
  message: string;
  resolve: (accepted: boolean) => void;
};
let active: Confirmation | null = null;
const listeners = new Set<() => void>();
const emit = () => listeners.forEach((listener) => listener());
export function currentConfirmation() {
  return active;
}
export function subscribeConfirmation(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
export function confirmDeletion(message: string): Promise<boolean> {
  if (active) return Promise.resolve(false);
  return new Promise((resolve) => {
    active = { message, resolve };
    emit();
  });
}
export function answerConfirmation(accepted: boolean) {
  const pending = active;
  if (!pending) return;
  active = null;
  emit();
  pending.resolve(accepted);
}

import test from "node:test";
import assert from "node:assert/strict";
import { readPreference, writePreference } from "../src/browser-storage.ts";

test("blocked or full browser storage leaves preferences optional", () => {
  const previous = Object.getOwnPropertyDescriptor(globalThis, "localStorage");
  try {
    Object.defineProperty(globalThis, "localStorage", {
      configurable: true,
      get() {
        throw new Error("Storage blocked");
      },
    });
    assert.equal(readPreference("scopearr-theme"), null);
    assert.doesNotThrow(() => writePreference("scopearr-theme", "light"));
    Object.defineProperty(globalThis, "localStorage", {
      configurable: true,
      value: {
        getItem: () => "dark",
        setItem() {
          throw new Error("Storage full");
        },
      },
    });
    assert.equal(readPreference("scopearr-theme"), "dark");
    assert.doesNotThrow(() => writePreference("scopearr-theme", "light"));
  } finally {
    if (previous) Object.defineProperty(globalThis, "localStorage", previous);
    else delete globalThis.localStorage;
  }
});

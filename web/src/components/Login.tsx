import { ArrowUpRight, LoaderCircle, ShieldCheck } from "lucide-preact";
import { useState } from "preact/hooks";
import type { Status } from "../types";

import { api } from "../api";
export function Login({
  status,
  done,
}: {
  status: Status;
  done: () => Promise<void>;
}) {
  const [user, setUser] = useState(status.demo ? "demo" : ""),
    [password, setPassword] = useState(status.demo ? "ArrScope-demo-2026" : ""),
    [token, setToken] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const submit = async (e: Event) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(status.initialized ? "login" : "setup", "POST", {
        username: user,
        password,
        token,
      });
      await done();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <main class="login-page">
      <div class="login-art">
        <div class="login-orbit" />
        <img src="/icon-1024.png" alt="ArrScope Icon" />
        <div>
          <span class="eyebrow">DEINE BIBLIOTHEK. KLAR GEFILTERT.</span>
          <h1>
            Die Details machen
            <br />
            den Unterschied.
          </h1>
          <p>
            Filme und Serien aus Radarr und Sonarr.
            <br />
            Alle Eigenschaften. Ein klarer Überblick.
          </p>
        </div>
        <span class="login-foot">
          <ShieldCheck size={16} /> Lokal gespeichert · Ohne Medien-Scans
        </span>
      </div>
      <div class="login-form">
        <div class="brand">
          <img
            src="/icon-64.png"
            srcSet="/icon-64.png 1x, /icon-256.png 2x"
            width="34"
            height="34"
            alt=""
          />
          <b>
            ArrScope<span> / </span>
          </b>
        </div>
        <h2>
          {status.initialized
            ? "Willkommen zurück."
            : "Deine Bibliothek beginnt hier."}
        </h2>
        <p>
          {status.initialized
            ? "Melde dich an, um deine Bibliothek zu entdecken."
            : "Erstelle dein lokales Admin-Konto. Danach verbindest du deine Dienste."}
        </p>
        {status.demo && (
          <div class="notice">
            Interaktive Demo mit Beispieldaten. Zugang ist vorausgefüllt.
          </div>
        )}
        <form onSubmit={submit}>
          {!status.initialized && (
            <label>
              Einrichtungscode
              <input
                value={token}
                onInput={(e) => setToken(e.currentTarget.value)}
                required
                autoComplete="off"
              />
              <small>
                Einmalig aus <code>/data/setup-token</code> in der
                Container-Konsole kopieren.
              </small>
            </label>
          )}
          <label>
            Benutzername
            <input
              value={user}
              onInput={(e) => setUser(e.currentTarget.value)}
              autoComplete="username"
              required
            />
          </label>
          <label>
            Passwort
            <input
              type="password"
              value={password}
              onInput={(e) => setPassword(e.currentTarget.value)}
              autoComplete={
                status.initialized ? "current-password" : "new-password"
              }
              minLength={status.initialized ? undefined : 12}
              required
            />
            {!status.initialized && (
              <small>Mindestens 12 Zeichen, maximal 72 UTF-8-Bytes.</small>
            )}
          </label>
          {error && (
            <p class="error" role="alert">
              {error}
            </p>
          )}
          <button class="primary" disabled={busy}>
            {busy ? (
              <LoaderCircle class="spin" size={18} />
            ) : (
              <ArrowUpRight size={18} />
            )}{" "}
            {status.initialized ? "Anmelden" : "Konto erstellen"}
          </button>
        </form>
        <div class="quiet login-version">
          ArrScope {status.version} · Für deinen eigenen Server
        </div>
      </div>
    </main>
  );
}

import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
const version = readFileSync("VERSION", "utf8").trim();
if (!/^\d+\.\d+\.\d+$/.test(version))
  throw new Error(
    "VERSION must contain a semantic version without a v prefix.",
  );
const pkg = JSON.parse(readFileSync("web/package.json", "utf8"));
if (pkg.version !== version)
  throw new Error("web/package.json must match VERSION.");
const docker = readFileSync("Dockerfile", "utf8");
const dockerVersions = [...docker.matchAll(/ARG VERSION=(.*)/g)];
if (dockerVersions.length !== 2 || dockerVersions.some((m) => m[1] !== version))
  throw new Error("Docker defaults must match VERSION.");
const notes = readFileSync(`docs/releases/${version}.md`, "utf8");
if (!notes.trim()) throw new Error("English release notes are required.");
if (process.env.VERSION && process.env.VERSION !== version)
  throw new Error("Release input must match VERSION.");
if (process.env.CI && !process.argv.includes("--release")) {
  const tags = execFileSync(
    "git",
    ["tag", "--list", "v*", "--sort=-v:refname"],
    { encoding: "utf8" },
  )
    .trim()
    .split("\n")
    .filter((t) => /^v\d+\.\d+\.\d+$/.test(t));
  if (tags[0]) {
    const old = tags[0].slice(1).split(".").map(Number),
      next = version.split(".").map(Number);
    const greater = next.findIndex((n, i) => n !== old[i]);
    const sameReleasedCommit =
      execFileSync("git", ["rev-parse", "HEAD"], {
        encoding: "utf8",
      }).trim() ===
      execFileSync("git", ["rev-list", "-n", "1", tags[0]], {
        encoding: "utf8",
      }).trim();
    if ((greater < 0 || next[greater] < old[greater]) && !sameReleasedCommit)
      throw new Error(
        "Changes after a release require a new VERSION and release notes.",
      );
  }
}
console.log(
  `Version ${version}: source, package, Docker and release notes agree.`,
);

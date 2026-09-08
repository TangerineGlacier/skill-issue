// The site reads catalog.json and nothing else. If the generator ever drops a
// field the site renders, this fails before the build does something subtle
// like print "undefined" on every card.
import { readFileSync } from "node:fs";

let catalog;
try {
  catalog = JSON.parse(readFileSync(new URL("../catalog.json", import.meta.url), "utf8"));
} catch {
  console.error("catalog.json is missing. It is generated, not committed:\n\n  go run ./tui catalog\n");
  process.exit(1);
}

const fail = [];
const need = (cond, msg) => cond || fail.push(msg);

need(Array.isArray(catalog.skills) && catalog.skills.length > 0, "catalog.skills is empty — run `si catalog`");
need(Array.isArray(catalog.groups), "catalog.groups missing");
need(typeof catalog.generated === "string", "catalog.generated missing");

for (const s of catalog.skills ?? []) {
  const at = `skills[${s.name ?? "?"}]`;
  for (const f of ["name", "group", "path", "hash", "body", "updated"]) {
    need(typeof s[f] === "string", `${at}.${f} is not a string`);
  }
  need(typeof s.size === "number", `${at}.size is not a number`);
  need(Array.isArray(s.files), `${at}.files is not an array`);
  for (const f of s.files ?? []) {
    need(typeof f.Rel === "string" && typeof f.Size === "number", `${at}.files entry is not {Rel,Size}`);
  }
  need(s.problems === null || Array.isArray(s.problems), `${at}.problems is not an array or null`);
  for (const p of s.problems ?? []) {
    need(typeof p.code === "string" && typeof p.level === "number", `${at}.problems entry missing code/level`);
  }
  need(catalog.groups.includes(s.group), `${at}.group ${s.group} is not in catalog.groups`);
}

if (fail.length) {
  console.error("catalog.json does not match what the site reads:");
  for (const f of fail) console.error("  ·", f);
  process.exit(1);
}
console.log(`catalog ok — ${catalog.skills.length} skills, ${catalog.groups.length} groups, ${catalog.problems?.length ?? 0} problems`);

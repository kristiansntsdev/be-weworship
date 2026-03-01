/**
 * Bulk ChordPro converter
 * Converts all songs whose lyrics_and_chords contain HTML (span tags) to ChordPro format.
 *
 * Usage:
 *   node scripts/convert-chordpro.mjs              # dry-run (shows preview, no DB writes)
 *   node scripts/convert-chordpro.mjs --write       # actually update the DB
 *   node scripts/convert-chordpro.mjs --write --id=135  # convert a single song by ID
 */

import pg from "pg";
import { readFileSync } from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// ──────────────────────────────────────────────────────────────────────────────
// Same algorithm as song-form-page.tsx
// ──────────────────────────────────────────────────────────────────────────────

function isChordOnlyLine(line) {
  const stripped = line
    .replace(/\[[^\]]+\]/g, "")
    .replace(/\//g, "")
    .trim();
  return stripped.length === 0 && /\[[^\]]+\]/.test(line);
}

function buildChordArray(chordLine) {
  const raw = chordLine.replace(/\[([^\]]+)\]/g, "$1");
  const c = [];
  raw.split(" ").forEach((token) => {
    if (token.length > 0) {
      c.push(token);
      for (let k = 0; k < token.length; k++) c.push("");
    } else {
      c.push("");
    }
  });
  return c;
}

function mergeChordWithLyric(chordLine, lyricLine) {
  const c = buildChordArray(chordLine);
  const lyric = lyricLine.split("");
  while (lyric.length < c.length) lyric.push(" ");
  let result = "";
  const len = Math.max(c.length, lyric.length);
  for (let i = 0; i < len; i++) {
    if (c[i]) result += `[${c[i]}]`;
    if (lyric[i]) result += lyric[i];
  }
  return result.trimEnd();
}

function mergeChordLines(text) {
  const lines = text.split("\n");
  const out = [];
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (isChordOnlyLine(line) && i + 1 < lines.length && !isChordOnlyLine(lines[i + 1])) {
      out.push(mergeChordWithLyric(line, lines[i + 1]));
      i += 2;
    } else {
      out.push(line);
      i++;
    }
  }
  return out.join("\n");
}

function htmlToChordPro(html) {
  const text = html
    .replace(/<span class="c"[^>]*>([^<]*)<\/span>/g, "[$1]")
    .replace(/<span class="on"[^>]*>([^<]*)<\/span>/g, "$1")
    .replace(/\[([^\]]+)\]\/\[([^\]]+)\]/g, "[$1/$2]")
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/<\/?(div|p|pre)[^>]*>/gi, "\n")
    .replace(/<[^>]+>/g, "")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&nbsp;/g, " ")
    .replace(/\n{3,}/g, "\n\n")
    .trim();

  return mergeChordLines(text);
}

function isHtml(text) {
  return text && /<span/i.test(text);
}

function isChordPro(text) {
  return text && /\[/.test(text) && !/<span/i.test(text);
}

// ──────────────────────────────────────────────────────────────────────────────
// Main
// ──────────────────────────────────────────────────────────────────────────────

const args = process.argv.slice(2);
const DRY_RUN = !args.includes("--write");
const targetId = args.find((a) => a.startsWith("--id="))?.split("=")[1] ?? null;

// Load DB_URL from .env
function loadEnv() {
  try {
    const envPath = path.join(__dirname, "../.env");
    const lines = readFileSync(envPath, "utf8").split("\n");
    for (const line of lines) {
      const match = line.match(/^DB_URL=(.+)$/);
      if (match) return match[1].trim();
    }
  } catch {}
  return process.env.DB_URL;
}

const DB_URL = loadEnv();
if (!DB_URL) {
  console.error("❌ DB_URL not found in .env or environment");
  process.exit(1);
}

const client = new pg.Client({ connectionString: DB_URL });

async function main() {
  await client.connect();
  console.log("✅ Connected to DB\n");

  let query = "SELECT id, title, lyrics_and_chords FROM songs WHERE lyrics_and_chords LIKE '%<span%'";
  const qparams = [];
  if (targetId) {
    query = "SELECT id, title, lyrics_and_chords FROM songs WHERE id = $1";
    qparams.push(targetId);
  }

  const { rows } = await client.query(query, qparams);
  console.log(`Found ${rows.length} song(s) with HTML lyrics${targetId ? ` (id=${targetId})` : ""}\n`);

  if (rows.length === 0) {
    console.log("Nothing to convert.");
    await client.end();
    return;
  }

  let converted = 0;
  let skipped = 0;
  let errors = 0;

  for (const row of rows) {
    try {
      const result = htmlToChordPro(row.lyrics_and_chords);

      // Quick sanity check
      if (!result.trim()) {
        console.warn(`  ⚠️  [${row.id}] "${row.title}" — result is empty, skipping`);
        skipped++;
        continue;
      }

      if (DRY_RUN) {
        console.log(`── [${row.id}] ${row.title}`);
        console.log("BEFORE:", row.lyrics_and_chords.slice(0, 120).replace(/\n/g, "↵"));
        console.log("AFTER: ", result.slice(0, 200).replace(/\n/g, "↵"));
        console.log();
      } else {
        await client.query(
          "UPDATE songs SET lyrics_and_chords = $1, \"updatedAt\" = NOW() WHERE id = $2",
          [result, row.id]
        );
        console.log(`  ✅ [${row.id}] "${row.title}" — converted`);
      }
      converted++;
    } catch (err) {
      console.error(`  ❌ [${row.id}] "${row.title}" — error: ${err.message}`);
      errors++;
    }
  }

  console.log("\n──────────────────────────────");
  if (DRY_RUN) {
    console.log(`DRY RUN — ${converted} songs would be converted, ${skipped} skipped, ${errors} errors`);
    console.log('Run with --write to apply changes: node scripts/convert-chordpro.mjs --write');
  } else {
    console.log(`Done — ${converted} converted, ${skipped} skipped, ${errors} errors`);
  }

  await client.end();
}

main().catch((err) => {
  console.error("Fatal:", err);
  process.exit(1);
});

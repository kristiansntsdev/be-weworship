# Plain Lyrics Migration

This migration adds a `plain_lyrics` column to the `songs` table and populates it with clean text extracted from `lyrics_and_chords`.

## Why?

The `lyrics_and_chords` column contains HTML markup, ChordPro notation (`[C]`, `[Am]`), and formatting directives (`{textcolor}`), which makes multi-word phrase searching difficult. The `plain_lyrics` column stores clean text for faster, more accurate search.

## Running the Migration

### Step 1: Add the column

```bash
# Connect to your PostgreSQL database
psql $DATABASE_URL -f migrations/add_plain_lyrics.sql
```

This will:
- Add `plain_lyrics TEXT` column to `songs` table
- Create a GIN index for full-text search

### Step 2: Extract lyrics from existing songs

```bash
# Run the extraction script
cd scripts
DATABASE_URL="your_connection_string" go run extract_lyrics.go
```

This will:
- Read all songs with `lyrics_and_chords` data
- Strip HTML tags, ChordPro chords, and formatting directives
- Populate `plain_lyrics` column
- Print progress every 100 songs

## Auto-extraction for new songs

The backend automatically extracts plain lyrics when:
- Creating a new song with lyrics
- Updating an existing song's lyrics

No manual intervention needed for new songs after migration!

## Search behavior

After migration, search queries will use `COALESCE(plain_lyrics, lyrics_and_chords)`:
- If `plain_lyrics` exists → searches clean text (fast, accurate)
- If `plain_lyrics` is NULL → falls back to `lyrics_and_chords` (for old songs not yet migrated)

## Example

Before:
```
lyrics_and_chords: "<div><pre>[C]Ku [G]Sembah Kau {textcolor}Dalam Roh{/textcolor}</pre></div>"
```

After:
```
plain_lyrics: "Ku Sembah Kau Dalam Roh"
```

Search for "Ku Sembah Kau Dalam Roh" now works perfectly! ✅

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	_ "github.com/lib/pq"
)

// ExtractPlainLyrics strips HTML, ChordPro markup, and chord tags from lyrics
func ExtractPlainLyrics(input string) string {
	if input == "" {
		return ""
	}

	text := input

	// Remove HTML tags
	htmlRegex := regexp.MustCompile(`<[^>]+>`)
	text = htmlRegex.ReplaceAllString(text, "")

	// Remove ChordPro format chords: [C], [Am], [G/B], etc.
	chordProRegex := regexp.MustCompile(`\[[^\]]+\]`)
	text = chordProRegex.ReplaceAllString(text, "")

	// Remove {directive} tags like {textcolor}, {sot}, {eot}
	directiveRegex := regexp.MustCompile(`\{[^}]+\}`)
	text = directiveRegex.ReplaceAllString(text, "")

	// Remove extra whitespace/newlines
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Connected to database successfully")

	// Fetch all songs with lyrics
	rows, err := db.Query(`
		SELECT id, title, lyrics_and_chords 
		FROM songs 
		WHERE lyrics_and_chords IS NOT NULL 
		AND lyrics_and_chords != ''
		ORDER BY id
	`)
	if err != nil {
		log.Fatalf("Failed to query songs: %v", err)
	}
	defer rows.Close()

	updated := 0
	skipped := 0

	// Prepare update statement
	updateStmt, err := db.Prepare(`UPDATE songs SET plain_lyrics = $1 WHERE id = $2`)
	if err != nil {
		log.Fatalf("Failed to prepare update statement: %v", err)
	}
	defer updateStmt.Close()

	for rows.Next() {
		var id int
		var title string
		var lyricsAndChords sql.NullString

		if err := rows.Scan(&id, &title, &lyricsAndChords); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		if !lyricsAndChords.Valid || lyricsAndChords.String == "" {
			skipped++
			continue
		}

		plainLyrics := ExtractPlainLyrics(lyricsAndChords.String)

		if plainLyrics == "" {
			skipped++
			continue
		}

		_, err := updateStmt.Exec(plainLyrics, id)
		if err != nil {
			log.Printf("Failed to update song ID %d (%s): %v", id, title, err)
			continue
		}

		updated++
		if updated%100 == 0 {
			fmt.Printf("Processed %d songs...\n", updated)
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating rows: %v", err)
	}

	fmt.Printf("\n✅ Migration complete!\n")
	fmt.Printf("   Updated: %d songs\n", updated)
	fmt.Printf("   Skipped: %d songs (no lyrics)\n", skipped)
}

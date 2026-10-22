// Backfill song embeddings for semantic search.
//
// Usage (from the repo root, with DB_URL, EMBEDDING_URL and EMBEDDING_TOKEN in .env):
//
//	go run ./scripts/backfill_embeddings          # embed songs whose title/artist/lyrics changed
//	go run ./scripts/backfill_embeddings --force  # re-embed everything (e.g. after changing EMBEDDING_MODEL)
//
// Safe to re-run: unchanged songs are skipped via their content hash. Requires migrations/add_song_embeddings.sql.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"be-songbanks-v1/api/platform"
	"be-songbanks-v1/api/providers"
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/services"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	force := flag.Bool("force", false, "re-embed every song, even if unchanged")
	flag.Parse()
	_ = godotenv.Load(".env")

	embedder := providers.NewEmbedderFromEnv()
	if embedder == nil {
		log.Fatal("EMBEDDING_URL and EMBEDDING_TOKEN must be set")
	}
	appCtx, err := platform.NewContext()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	svc := services.NewSongEmbeddingService(repositories.NewEmbeddingRepository(appCtx.DB), embedder)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	start := time.Now()
	result, err := svc.Backfill(ctx, *force, func(done, total int) {
		if done%25 == 0 || done == total {
			log.Printf("%d/%d songs checked (%s)", done, total, time.Since(start).Round(time.Second))
		}
	})
	for id, ferr := range result.Failed {
		log.Printf("FAILED song %d: %v", id, ferr)
	}
	log.Printf("done: %d embedded, %d unchanged, %d failed in %s", result.Embedded, result.Skipped, len(result.Failed), time.Since(start).Round(time.Second))
	if err != nil {
		log.Fatalf("stopped early: %v", err)
	}
	if len(result.Failed) > 0 {
		os.Exit(1)
	}
}

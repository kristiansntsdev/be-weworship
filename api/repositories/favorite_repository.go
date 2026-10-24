package repositories

import (
	"github.com/jmoiron/sqlx"
)

type FavoriteRepository struct {
	db *sqlx.DB
}

func NewFavoriteRepository(db *sqlx.DB) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

func (r *FavoriteRepository) Add(userID, songID int) error {
	_, err := r.db.Exec(r.db.Rebind(`
INSERT INTO song_favorites (user_id, song_id) VALUES (?, ?)
ON CONFLICT (user_id, song_id) DO NOTHING`), userID, songID)
	return err
}

func (r *FavoriteRepository) Remove(userID, songID int) error {
	_, err := r.db.Exec(r.db.Rebind(`DELETE FROM song_favorites WHERE user_id = ? AND song_id = ?`), userID, songID)
	return err
}

func (r *FavoriteRepository) IsFavorite(userID, songID int) (bool, error) {
	var n int
	err := r.db.QueryRow(r.db.Rebind(`SELECT COUNT(*) FROM song_favorites WHERE user_id = ? AND song_id = ?`), userID, songID).Scan(&n)
	return n > 0, err
}

// ListSongIDs returns favorited song ids newest-first.
func (r *FavoriteRepository) ListSongIDs(userID int) ([]int, error) {
	ids := []int{}
	err := r.db.Select(&ids, r.db.Rebind(`
SELECT song_id FROM song_favorites WHERE user_id = ? ORDER BY "createdAt" DESC`), userID)
	return ids, err
}

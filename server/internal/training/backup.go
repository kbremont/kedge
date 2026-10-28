package training

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Backup is a consistent snapshot of the database, ready to stream. Closing
// it removes the snapshot.
type Backup struct {
	// Name is a filename for the download, such as 531-2026-10-28.db.
	Name string
	*os.File
}

// Close closes and removes the snapshot.
func (b *Backup) Close() error {
	err := b.File.Close()
	if rmErr := os.Remove(b.File.Name()); err == nil {
		err = rmErr
	}
	return err
}

// Backup takes a snapshot of the database. VACUUM INTO takes its own read
// transaction, so this is safe to call while the app is running: there is no
// need to stop the container to take a backup.
func (s *Service) Backup(ctx context.Context) (*Backup, error) {
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("531-backup-%d.db", time.Now().UnixNano()))

	if _, err := s.conn.ExecContext(ctx, "VACUUM INTO ?", tmp); err != nil {
		os.Remove(tmp)
		return nil, fmt.Errorf("vacuum into %s: %w", tmp, err)
	}
	f, err := os.Open(tmp)
	if err != nil {
		os.Remove(tmp)
		return nil, fmt.Errorf("open snapshot: %w", err)
	}
	return &Backup{Name: fmt.Sprintf("531-%s.db", time.Now().Format("2006-01-02")), File: f}, nil
}

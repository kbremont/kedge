package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// handleBackup streams a consistent snapshot of the database. VACUUM INTO
// takes its own read transaction, so this is safe to call while the app is
// running -- no need to stop the container to take a backup.
func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("531-backup-%d.db", time.Now().UnixNano()))
	defer os.Remove(tmp)

	if _, err := s.conn.ExecContext(r.Context(), "VACUUM INTO ?", tmp); err != nil {
		s.fail(w, r, fmt.Errorf("vacuum into %s: %w", tmp, err))
		return
	}

	f, err := os.Open(tmp)
	if err != nil {
		s.fail(w, r, fmt.Errorf("open snapshot: %w", err))
		return
	}
	defer f.Close()

	name := fmt.Sprintf("531-%s.db", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	if _, err := io.Copy(w, f); err != nil {
		s.log.ErrorContext(r.Context(), "backup stream interrupted", "error", err)
	}
}

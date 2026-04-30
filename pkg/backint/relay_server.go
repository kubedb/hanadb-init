/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Free Trial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Free-Trial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backint

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

func StartRelayServer(bindAddr string, cfg config) (*RelayServer, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/backup", func(w http.ResponseWriter, r *http.Request) {
		if err := requireRelayAuth(cfg, r); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		sourcePath := r.URL.Query().Get("sourcePath")
		if sourcePath == "" {
			http.Error(w, "missing sourcePath", http.StatusBadRequest)
			return
		}
		meta := metadata{
			ID:         randomID(),
			SourcePath: sourcePath,
			SpoolPath:  sourcePath,
			UserID:     r.URL.Query().Get("userID"),
			BackupID:   r.URL.Query().Get("backupID"),
			Level:      r.URL.Query().Get("level"),
		}
		snapshotID, size, err := resticBackupStream(cfg, meta, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		meta.SnapshotID = snapshotID
		meta.Size = size
		_ = saveMeta(cfg, meta)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(relayBackupResponse{ID: meta.ID, SnapshotID: snapshotID, Size: size})
	})
	mux.HandleFunc("/restore", func(w http.ResponseWriter, r *http.Request) {
		if err := requireRelayAuth(cfg, r); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		id := r.URL.Query().Get("id")
		src := r.URL.Query().Get("sourcePath")
		meta, err := resolveRestoreMeta(cfg, id, src)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err := resticRestoreToWriter(cfg, meta, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	mux.HandleFunc("/inquire", func(w http.ResponseWriter, r *http.Request) {
		if err := requireRelayAuth(cfg, r); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		id := r.URL.Query().Get("id")
		src := r.URL.Query().Get("sourcePath")
		meta, err := resolveRestoreMeta(cfg, id, src)
		resp := relayInquireResponse{Found: err == nil}
		if err == nil {
			resp.ID = meta.ID
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		if err := requireRelayAuth(cfg, r); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		meta, err := loadMeta(cfg, id)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(relayDeleteResponse{Deleted: false})
			return
		}
		if err := resticDelete(cfg, meta.SnapshotID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = os.Remove(metaPath(cfg, id))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(relayDeleteResponse{Deleted: true})
	})

	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return nil, err
	}
	srv := &http.Server{Handler: mux}
	go func() {
		_ = srv.Serve(ln)
	}()
	return &RelayServer{server: srv, addr: "http://" + ln.Addr().String()}, nil
}

func (s *RelayServer) URL() string {
	return s.addr
}

func (s *RelayServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func requireRelayAuth(cfg config, r *http.Request) error {
	if cfg.RelayToken == "" {
		return nil
	}
	if r.Header.Get("X-Backint-Relay-Token") != cfg.RelayToken {
		return fmt.Errorf("unauthorized")
	}
	return nil
}

func resolveRestoreMeta(cfg config, id, src string) (metadata, error) {
	if id != "" {
		return loadMeta(cfg, id)
	}
	if src != "" {
		return loadLatestMetaBySourcePath(cfg, src)
	}
	return metadata{}, os.ErrNotExist
}

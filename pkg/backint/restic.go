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
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func resticBackupStream(cfg config, meta metadata, in io.Reader) (string, int64, error) {
	var snapshotID string
	var size int64
	args := []string{"backup", "--json", "--stdin", "--stdin-filename", meta.SpoolPath}
	for _, tag := range resticTags(meta) {
		args = append(args, "--tag", tag)
	}
	if meta.UserID != "" {
		args = append(args, "--host", meta.UserID)
	}
	cmd := exec.Command(cfg.ResticBin, args...)
	cmd.Env = resticEnv(cfg)
	cmd.Stdin = in
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", 0, fmt.Errorf("restic backup failed: %s", out.String())
	}
	type summary struct {
		MessageType         string `json:"message_type"`
		SnapshotID          string `json:"snapshot_id"`
		TotalBytesProcessed int64  `json:"total_bytes_processed"`
	}
	sc := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	for sc.Scan() {
		var s summary
		if json.Unmarshal(sc.Bytes(), &s) == nil && s.MessageType == "summary" && s.SnapshotID != "" {
			snapshotID = s.SnapshotID
			size = s.TotalBytesProcessed
		}
	}
	if snapshotID == "" {
		return "", 0, fmt.Errorf("could not parse snapshot id from restic output: %s", out.String())
	}
	return snapshotID, size, nil
}

func resticRestoreToWriter(cfg config, meta metadata, out io.Writer) error {
	cmd := exec.Command(cfg.ResticBin, "dump", meta.SnapshotID, meta.SpoolPath)
	cmd.Env = resticEnv(cfg)
	cmd.Stdout = out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restic dump failed: %s", stderr.String())
	}
	return nil
}

func resticDelete(cfg config, snapshotID string) error {
	return withRepoLock(cfg, func() error {
		cmd := exec.Command(cfg.ResticBin, "forget", snapshotID, "--prune")
		cmd.Env = resticEnv(cfg)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("restic forget failed: %s", out.String())
		}
		return nil
	})
}

func loadMetaFromResticByID(cfg config, id string) (metadata, error) {
	snaps, err := resticSnapshots(cfg, "ebid:"+id)
	if err != nil {
		return metadata{}, err
	}
	return snapshotToMeta(latestSnapshot(snaps), id)
}

func loadLatestMetaFromResticBySourcePath(cfg config, src string) (metadata, error) {
	snaps, err := resticSnapshots(cfg, "source:"+encodeTagValue(src))
	if err != nil {
		return metadata{}, err
	}
	return snapshotToMeta(latestSnapshot(snaps), "")
}

func resticSnapshots(cfg config, tag string) ([]resticSnapshot, error) {
	var snaps []resticSnapshot
	cmd := exec.Command(cfg.ResticBin, "snapshots", "--json", "--tag", tag)
	cmd.Env = resticEnv(cfg)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("restic snapshots failed: %s", out.String())
	}
	if err := json.Unmarshal(out.Bytes(), &snaps); err != nil {
		return nil, fmt.Errorf("failed to parse restic snapshots output: %w", err)
	}
	return snaps, nil
}

func latestSnapshot(snaps []resticSnapshot) resticSnapshot {
	var chosen resticSnapshot
	for _, snap := range snaps {
		if chosen.ID == "" || snap.Time.After(chosen.Time) {
			chosen = snap
		}
	}
	return chosen
}

func snapshotToMeta(snap resticSnapshot, fallbackID string) (metadata, error) {
	if snap.ID == "" {
		return metadata{}, os.ErrNotExist
	}
	m := metadata{
		ID:         fallbackID,
		SnapshotID: snap.ID,
		UserID:     snap.Hostname,
	}
	if len(snap.Paths) > 0 {
		m.SpoolPath = snap.Paths[0]
	}
	for _, tag := range snap.Tags {
		switch {
		case strings.HasPrefix(tag, "ebid:") && m.ID == "":
			m.ID = strings.TrimPrefix(tag, "ebid:")
		case strings.HasPrefix(tag, "source:"):
			m.SourcePath = decodeTagValue(strings.TrimPrefix(tag, "source:"))
		case strings.HasPrefix(tag, "backup:"):
			m.BackupID = strings.TrimPrefix(tag, "backup:")
		case strings.HasPrefix(tag, "level:"):
			m.Level = strings.TrimPrefix(tag, "level:")
		}
	}
	if m.ID == "" || m.SpoolPath == "" {
		return metadata{}, os.ErrNotExist
	}
	return m, nil
}

func resticTags(meta metadata) []string {
	tags := []string{"backint", "ebid:" + meta.ID, "source:" + encodeTagValue(meta.SourcePath)}
	if meta.BackupID != "" {
		tags = append(tags, "backup:"+meta.BackupID)
	}
	if meta.Level != "" {
		tags = append(tags, "level:"+meta.Level)
	}
	return tags
}

func encodeTagValue(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func decodeTagValue(s string) string {
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(data)
}

func randomID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

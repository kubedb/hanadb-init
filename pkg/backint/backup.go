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
	"fmt"
	"os"
)

func handleBackup(cfg config, a args, entries []inputEntry) []string {
	if cfg.RelayURL != "" {
		return relayHandleBackup(cfg, a, entries)
	}
	var outputs []string
	for _, e := range entries {
		if e.Keyword != "PIPE" || len(e.Args) < 1 {
			continue
		}
		src := e.Args[0]
		id := randomID()
		meta := metadata{
			ID:         id,
			SourcePath: src,
			SpoolPath:  src,
			UserID:     a.userID,
			BackupID:   a.backupID,
			Level:      a.backupLevel,
		}
		snapID, size, err := resticBackup(cfg, meta)
		if err != nil {
			return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
		}
		meta.SnapshotID = snapID
		meta.Size = size
		if err := saveMeta(cfg, meta); err != nil {
			return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
		}
		outputs = append(outputs, fmt.Sprintf(`#SAVED "%s" "%s" "%d"`, id, src, size))
	}
	if len(outputs) == 0 {
		outputs = append(outputs, `#ERROR "" "no PIPE entries found"`)
	}
	return outputs
}

func resticBackup(cfg config, meta metadata) (string, int64, error) {
	in, err := os.Open(meta.SourcePath)
	if err != nil {
		return "", 0, err
	}
	defer func() {
		_ = in.Close()
	}()
	return resticBackupStream(cfg, meta, in)
}

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

func handleRestore(cfg config, entries []inputEntry) []string {
	if cfg.RelayURL != "" {
		return relayHandleRestore(cfg, entries)
	}
	var outputs []string
	for _, e := range entries {
		switch e.Keyword {
		case "EBID":
			if len(e.Args) < 2 {
				continue
			}
			id := e.Args[0]
			src := e.Args[1]
			dest := src
			if len(e.Args) > 2 {
				dest = e.Args[2]
			}
			meta, err := loadMeta(cfg, id)
			if err != nil {
				return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
			}
			if err := resticRestore(cfg, meta, dest); err != nil {
				return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
			}
			outputs = append(outputs, fmt.Sprintf(`#RESTORED "%s" "%s"`, id, src))
		case "NULL":
			if len(e.Args) < 1 {
				continue
			}
			src := e.Args[0]
			dest := src
			if len(e.Args) > 1 {
				dest = e.Args[1]
			}
			meta, err := loadLatestMetaBySourcePath(cfg, src)
			if err != nil {
				outputs = append(outputs, fmt.Sprintf(`#NOTFOUND "%s"`, src))
				continue
			}
			if err := resticRestore(cfg, meta, dest); err != nil {
				return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
			}
			outputs = append(outputs, fmt.Sprintf(`#RESTORED "%s" "%s"`, meta.ID, src))
		}
	}
	if len(outputs) == 0 {
		outputs = append(outputs, `#ERROR "" "no EBID or NULL entries found"`)
	}
	return outputs
}

func resticRestore(cfg config, meta metadata, dest string) error {
	out, err := os.OpenFile(dest, os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	return resticRestoreToWriter(cfg, meta, out)
}

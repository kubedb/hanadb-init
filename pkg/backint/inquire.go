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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func handleInquire(cfg config, entries []inputEntry) []string {
	if cfg.RelayURL != "" {
		return relayHandleInquire(cfg, entries)
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
			if _, err := loadMeta(cfg, id); err == nil {
				outputs = append(outputs, fmt.Sprintf(`#BACKUP "%s" "%s"`, id, src))
			} else {
				outputs = append(outputs, fmt.Sprintf(`#NOTFOUND "%s" "%s"`, id, src))
			}
		case "NULL":
			if len(e.Args) == 0 {
				matches, _ := filepath.Glob(filepath.Join(cfg.MetaRoot, "*.json"))
				if len(matches) == 0 {
					outputs = append(outputs, "#NOTFOUND")
				}
				for _, p := range matches {
					var m metadata
					data, _ := os.ReadFile(p)
					if json.Unmarshal(data, &m) == nil {
						outputs = append(outputs, fmt.Sprintf(`#BACKUP "%s" "%s"`, m.ID, m.SourcePath))
					}
				}
			} else {
				src := e.Args[0]
				if m, err := loadLatestMetaBySourcePath(cfg, src); err == nil {
					outputs = append(outputs, fmt.Sprintf(`#BACKUP "%s" "%s"`, m.ID, src))
				} else {
					outputs = append(outputs, fmt.Sprintf(`#NOTFOUND "%s"`, src))
				}
			}
		}
	}
	if len(outputs) == 0 {
		outputs = append(outputs, "#NOTFOUND")
	}
	return outputs
}

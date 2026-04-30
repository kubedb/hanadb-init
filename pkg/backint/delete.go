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

func handleDelete(cfg config, entries []inputEntry) []string {
	if cfg.RelayURL != "" {
		return relayHandleDelete(cfg, entries)
	}
	var outputs []string
	for _, e := range entries {
		if e.Keyword != "EBID" || len(e.Args) < 2 {
			continue
		}
		id := e.Args[0]
		src := e.Args[1]
		meta, err := loadMeta(cfg, id)
		if err != nil {
			outputs = append(outputs, fmt.Sprintf(`#NOTFOUND "%s" "%s"`, id, src))
			continue
		}
		if err := resticDelete(cfg, meta.SnapshotID); err != nil {
			outputs = append(outputs, fmt.Sprintf(`#ERROR "%s" "%s"`, id, err.Error()))
			continue
		}
		_ = os.Remove(metaPath(cfg, id))
		outputs = append(outputs, fmt.Sprintf(`#DELETED "%s" "%s"`, id, src))
	}
	if len(outputs) == 0 {
		outputs = append(outputs, "#NOTFOUND")
	}
	return outputs
}

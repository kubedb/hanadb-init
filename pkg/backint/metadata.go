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
	"os"
	"path/filepath"
)

func metaPath(cfg config, id string) string {
	return filepath.Join(cfg.MetaRoot, id+".json")
}

func saveMeta(cfg config, m metadata) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath(cfg, m.ID), data, 0o644)
}

func loadMeta(cfg config, id string) (metadata, error) {
	var m metadata
	data, err := os.ReadFile(metaPath(cfg, id))
	if err == nil {
		err = json.Unmarshal(data, &m)
		return m, err
	}
	return loadMetaFromResticByID(cfg, id)
}

func loadLatestMetaBySourcePath(cfg config, src string) (metadata, error) {
	var chosen metadata
	var chosenTime int64
	matches, err := filepath.Glob(filepath.Join(cfg.MetaRoot, "*.json"))
	if err != nil {
		return chosen, err
	}
	for _, p := range matches {
		var m metadata
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if json.Unmarshal(data, &m) != nil || m.SourcePath != src {
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		mt := info.ModTime().UnixNano()
		if chosenTime == 0 || mt > chosenTime {
			chosen = m
			chosenTime = mt
		}
	}
	if chosenTime != 0 {
		return chosen, nil
	}
	return loadLatestMetaFromResticBySourcePath(cfg, src)
}

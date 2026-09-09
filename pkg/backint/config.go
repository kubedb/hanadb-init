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
	"errors"
	"os"
	"strings"
	"time"
)

func loadConfig(path string) (config, error) {
	var cfg config
	if path == "" {
		return cfg, errors.New("missing Backint relay parameter file")
	}
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer func() {
		_ = f.Close()
	}()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		switch key {
		case "BACKINT_RELAY_URL":
			cfg.RelayURL = strings.TrimSuffix(value, "/")
		case "BACKINT_RELAY_TOKEN":
			cfg.RelayToken = value
		case "BACKINT_RELAY_CA":
			cfg.RelayCA, err = decodeRelayCA(value)
			if err != nil {
				return cfg, err
			}
		case "RESTIC_COMMAND_TIMEOUT":
			// Kept as a wire-compatible timeout key for already deployed relays;
			// this agent never starts a Restic process.
			cfg.CommandTimeout, err = time.ParseDuration(value)
			if err != nil || cfg.CommandTimeout <= 0 {
				return cfg, errors.New("invalid relay request timeout: expected a positive duration")
			}
		case "BACKINT_META_ROOT", "BACKINT_SPOOL_ROOT", "TMPDIR":
			// Older relays emit these directory hints. Relay-only agents do not
			// create metadata/spool directories or apply environment settings.
		case "RESTIC_BIN", "RESTIC_REPOSITORY", "RESTIC_PASSWORD", "RESTIC_PASSWORD_FILE", "NICE_ADJUSTMENT", "IONICE_CLASS", "IONICE_CLASS_DATA":
			return cfg, errors.New("standalone Restic configuration is no longer supported; configure an authenticated Backint relay")
		default:
			return cfg, errors.New("unsupported Backint relay setting")
		}
	}
	if err := sc.Err(); err != nil {
		return cfg, err
	}
	if _, err := validateRelayClientConfig(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

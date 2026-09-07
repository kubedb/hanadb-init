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
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func loadConfig(path string) (config, error) {
	cfg := config{
		ResticBin: "restic",
		MetaRoot:  filepath.Join(os.TempDir(), "hdbbackint-meta"),
		SpoolRoot: filepath.Join(os.TempDir(), "hdbbackint-spool"),
		Env:       map[string]string{},
	}
	if path != "" {
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
			k := strings.TrimSpace(parts[0])
			v := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			switch k {
			case "RESTIC_BIN":
				cfg.ResticBin = v
			case "RESTIC_COMMAND_TIMEOUT":
				cfg.CommandTimeout, err = time.ParseDuration(v)
				if err != nil {
					return cfg, fmt.Errorf("invalid RESTIC_COMMAND_TIMEOUT %q: %w", v, err)
				}
			case "NICE_ADJUSTMENT":
				cfg.NiceAdjustment, err = parseInt32(v)
				if err != nil {
					return cfg, fmt.Errorf("invalid NICE_ADJUSTMENT %q: %w", v, err)
				}
			case "IONICE_CLASS":
				cfg.IONiceClass, err = parseInt32(v)
				if err != nil {
					return cfg, fmt.Errorf("invalid IONICE_CLASS %q: %w", v, err)
				}
			case "IONICE_CLASS_DATA":
				cfg.IONiceClassData, err = parseInt32(v)
				if err != nil {
					return cfg, fmt.Errorf("invalid IONICE_CLASS_DATA %q: %w", v, err)
				}
			case "RESTIC_REPOSITORY":
				cfg.Repo = v
			case "RESTIC_PASSWORD":
				cfg.Password = v
			case "RESTIC_PASSWORD_FILE":
				cfg.PasswordFile = v
			case "BACKINT_RELAY_URL":
				cfg.RelayURL = strings.TrimRight(v, "/")
			case "BACKINT_RELAY_TOKEN":
				cfg.RelayToken = v
			case "BACKINT_META_ROOT":
				cfg.MetaRoot = v
			case "BACKINT_SPOOL_ROOT":
				cfg.SpoolRoot = v
			default:
				cfg.Env[k] = v
			}
		}
		if err := sc.Err(); err != nil {
			return cfg, err
		}
	}
	// Relay mode is used by the KubeStash job: HANA calls the in-pod Backint
	// binary, and that binary streams data to the job-side relay where Restic
	// credentials live. Without relay mode, this agent talks to Restic directly
	// and therefore needs repository and password configuration locally.
	if cfg.RelayURL == "" {
		if cfg.Repo == "" {
			return cfg, errors.New("missing RESTIC_REPOSITORY in config")
		}
		if cfg.Password == "" && cfg.PasswordFile == "" {
			return cfg, errors.New("missing RESTIC_PASSWORD or RESTIC_PASSWORD_FILE in config")
		}
	}
	if err := os.MkdirAll(cfg.MetaRoot, 0o755); err != nil {
		return cfg, err
	}
	if err := os.MkdirAll(cfg.SpoolRoot, 0o755); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func resticEnv(cfg config) []string {
	env := append([]string{}, os.Environ()...)
	env = append(env,
		"RESTIC_REPOSITORY="+cfg.Repo,
		"RESTIC_CACHE_DIR="+filepath.Join(cfg.MetaRoot, "restic-cache"),
	)
	if cfg.Password != "" {
		env = append(env, "RESTIC_PASSWORD="+cfg.Password)
	}
	if cfg.PasswordFile != "" {
		env = append(env, "RESTIC_PASSWORD_FILE="+cfg.PasswordFile)
	}
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}
	return env
}

type resticCmd struct {
	*exec.Cmd
	cancel context.CancelFunc
}

func (cmd *resticCmd) Run() error {
	defer cmd.cancel()
	return cmd.Cmd.Run()
}

func resticCommand(cfg config, args ...string) *resticCmd {
	commandArgs := make([]string, 0, len(args)+len(cfg.ResticArgs))
	commandArgs = append(commandArgs, args...)
	commandArgs = append(commandArgs, cfg.ResticArgs...)
	commandName := cfg.ResticBin
	if cfg.NiceAdjustment != nil {
		commandArgs = append([]string{"-n", strconv.FormatInt(int64(*cfg.NiceAdjustment), 10), commandName}, commandArgs...)
		commandName = "nice"
	}
	if cfg.IONiceClass != nil || cfg.IONiceClassData != nil {
		prefix := make([]string, 0, 5)
		if cfg.IONiceClass != nil {
			prefix = append(prefix, "-c", strconv.FormatInt(int64(*cfg.IONiceClass), 10))
		}
		if cfg.IONiceClassData != nil {
			prefix = append(prefix, "-n", strconv.FormatInt(int64(*cfg.IONiceClassData), 10))
		}
		prefix = append(prefix, commandName)
		commandArgs = append(prefix, commandArgs...)
		commandName = "ionice"
	}

	ctx := context.Background()
	cancel := func() {}
	if cfg.CommandTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.CommandTimeout)
	}
	return &resticCmd{
		Cmd:    exec.CommandContext(ctx, commandName, commandArgs...),
		cancel: cancel,
	}
}

func parseInt32(value string) (*int32, error) {
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil, err
	}
	result := int32(parsed)
	return &result, nil
}

func withRepoLock(cfg config, fn func() error) error {
	lockPath := filepath.Join(cfg.MetaRoot, "repo.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	}()
	// HANA can invoke DELETE/INQUIRE while backup metadata still exists in the
	// same mounted agent directory. Serialize Restic mutations so prune does not
	// race another Backint operation in the pod.
	return fn()
}

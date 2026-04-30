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

import "testing"

func TestShouldRun(t *testing.T) {
	tests := []struct {
		name    string
		program string
		argv    []string
		want    bool
	}{
		{name: "hdbbackint name", program: "/usr/sap/HXE/SYS/global/hdb/opt/hdbbackint", want: true},
		{name: "function flag", program: "hanadb-restic-plugin", argv: []string{"-f", "backup"}, want: true},
		{name: "backint version flag", program: "hanadb-restic-plugin", argv: []string{"-v"}, want: true},
		{name: "plugin command", program: "hanadb-restic-plugin", argv: []string{"version"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldRun(tt.program, tt.argv); got != tt.want {
				t.Fatalf("ShouldRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	got, err := parseArgs([]string{
		"-f", "backup",
		"-i", "/tmp/in",
		"-o", "/tmp/out",
		"-p", "/tmp/param",
		"-u", "HXE",
		"-s", "123",
		"-c", "1",
		"-l", "full",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.function != "backup" || got.inputFile != "/tmp/in" || got.outputFile != "/tmp/out" {
		t.Fatalf("unexpected parse result: %#v", got)
	}
	if got.paramFile != "/tmp/param" || got.userID != "HXE" || got.backupID != "123" || got.backupLevel != "full" {
		t.Fatalf("unexpected parse result: %#v", got)
	}
}

func TestResticTagsRoundTripSource(t *testing.T) {
	meta := metadata{
		ID:         "ebid-1",
		SourcePath: "/usr/sap/HXE/SYS/global/hdb/backint/DB_HXE/log_backup_0_0_0_0",
		BackupID:   "backup-1",
		Level:      "log",
	}
	tags := resticTags(meta)
	var encoded string
	for _, tag := range tags {
		if len(tag) > len("source:") && tag[:len("source:")] == "source:" {
			encoded = tag[len("source:"):]
		}
	}
	if encoded == "" {
		t.Fatalf("source tag not found in %v", tags)
	}
	if got := decodeTagValue(encoded); got != meta.SourcePath {
		t.Fatalf("decodeTagValue() = %q, want %q", got, meta.SourcePath)
	}
}

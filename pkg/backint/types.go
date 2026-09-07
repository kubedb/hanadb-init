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

import "time"

const BackintVersion = "backint 1.04"

var ToolVersion = "KubeDB HANA Backint Restic Plugin 0.1"

type args struct {
	function      string
	inputFile     string
	outputFile    string
	userID        string
	paramFile     string
	backupID      string
	objectCount   string
	backupLevel   string
	version       bool
	versionDetail bool
}

type Config struct {
	ResticBin       string
	ResticArgs      []string
	Repo            string
	Password        string
	PasswordFile    string
	RelayURL        string
	RelayToken      string
	MetaRoot        string
	SpoolRoot       string
	Env             map[string]string
	NiceAdjustment  *int32
	IONiceClass     *int32
	IONiceClassData *int32
	CommandTimeout  time.Duration
}

type config = Config

type metadata struct {
	ID         string `json:"id"`
	SourcePath string `json:"sourcePath"`
	SpoolPath  string `json:"spoolPath"`
	SnapshotID string `json:"snapshotId"`
	Size       int64  `json:"size"`
	UserID     string `json:"userId"`
	BackupID   string `json:"backupId"`
	Level      string `json:"level"`
}

type inputEntry struct {
	Keyword string
	Args    []string
}

type resticSnapshot struct {
	ID       string    `json:"id"`
	ShortID  string    `json:"short_id"`
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Paths    []string  `json:"paths"`
	Tags     []string  `json:"tags"`
}

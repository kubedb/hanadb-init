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
	"context"
	"time"
)

const BackintVersion = "backint 1.04"

var ToolVersion = "KubeDB HANA Backint Agent 0.1"

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
	// Context bounds relay requests; nil uses the command timeout alone.
	Context    context.Context
	RelayURL   string
	RelayToken string
	// RelayCA contains only the operation's pinned relay trust certificates.
	RelayCA        []byte
	CommandTimeout time.Duration
}

type config = Config

type inputEntry struct {
	Keyword string
	Args    []string
}

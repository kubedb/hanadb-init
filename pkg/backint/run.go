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
	"errors"
	"fmt"
	"os"
	"strings"
)

func Run(argv []string) int {
	a, err := parseArgs(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if a.version {
		fmt.Printf("%q %q\n", BackintVersion, ToolVersion)
		return 0
	}
	if a.versionDetail {
		fmt.Printf("%q %q\n", BackintVersion, ToolVersion)
		fmt.Println("Backint-compatible relay agent for SAP HANA")
		fmt.Println(RelayTLSCapability)
		return 0
	}

	cfg, err := loadConfig(a.paramFile)
	if err != nil {
		writeOutput(a.outputFile, fmt.Sprintf(`#ERROR "%s" "%s"`, a.inputFile, err.Error()))
		return 1
	}

	entries, err := readInput(a.inputFile)
	if err != nil {
		writeOutput(a.outputFile, fmt.Sprintf(`#ERROR "%s" "%s"`, a.inputFile, err.Error()))
		return 1
	}

	outputs := []string{fmt.Sprintf(`#SOFTWAREID "%s" "%s"`, BackintVersion, ToolVersion)}
	switch strings.ToUpper(a.function) {
	case "BACKUP":
		outputs = append(outputs, relayHandleBackup(cfg, a, entries)...)
	case "RESTORE":
		outputs = append(outputs, relayHandleRestore(cfg, entries)...)
	case "INQUIRE":
		outputs = append(outputs, relayHandleInquire(cfg, entries)...)
	case "DELETE":
		outputs = append(outputs, relayHandleDelete(cfg, entries)...)
	default:
		outputs = append(outputs, fmt.Sprintf(`#ERROR "%s" "unsupported function %q"`, a.inputFile, a.function))
		writeOutput(a.outputFile, strings.Join(outputs, "\n"))
		return 1
	}

	for _, line := range outputs {
		if strings.HasPrefix(line, "#ERROR") {
			writeOutput(a.outputFile, strings.Join(outputs, "\n"))
			return 1
		}
	}
	writeOutput(a.outputFile, strings.Join(outputs, "\n"))
	return 0
}

func parseArgs(argv []string) (args, error) {
	var a args
	next := func(i *int, flag string) (string, error) {
		*i = *i + 1
		if *i >= len(argv) {
			return "", fmt.Errorf("missing value for %s", flag)
		}
		return argv[*i], nil
	}
	for i := 0; i < len(argv); i++ {
		var err error
		switch argv[i] {
		case "-f":
			a.function, err = next(&i, argv[i])
		case "-i":
			a.inputFile, err = next(&i, argv[i])
		case "-o":
			a.outputFile, err = next(&i, argv[i])
		case "-u":
			a.userID, err = next(&i, argv[i])
		case "-p":
			a.paramFile, err = next(&i, argv[i])
		case "-s":
			a.backupID, err = next(&i, argv[i])
		case "-c":
			a.objectCount, err = next(&i, argv[i])
		case "-l":
			a.backupLevel, err = next(&i, argv[i])
		case "-v":
			a.version = true
		case "-V":
			a.versionDetail = true
		}
		if err != nil {
			return a, err
		}
	}
	if !a.version && !a.versionDetail {
		if a.function == "" || a.inputFile == "" || a.outputFile == "" {
			return a, errors.New("required flags: -f, -i, -o")
		}
	}
	return a, nil
}

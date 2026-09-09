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

func relayHandleBackup(cfg config, a args, entries []inputEntry) []string {
	var outputs []string
	for _, e := range entries {
		if e.Keyword != "PIPE" || len(e.Args) < 1 {
			continue
		}
		src := e.Args[0]
		f, err := os.Open(src)
		if err != nil {
			return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
		}
		resp, err := relayBackup(cfg, a, src, f)
		_ = f.Close()
		if err != nil {
			return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
		}
		outputs = append(outputs, fmt.Sprintf(`#SAVED "%s" "%s" "%d"`, resp.ID, src, resp.Size))
	}
	if len(outputs) == 0 {
		outputs = append(outputs, `#ERROR "" "no PIPE entries found"`)
	}
	return outputs
}

func relayHandleRestore(cfg config, entries []inputEntry) []string {
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
			out, err := os.OpenFile(dest, os.O_WRONLY, 0o644)
			if err != nil {
				return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
			}
			err = relayRestore(cfg, id, src, out)
			if closeErr := out.Close(); err == nil {
				err = closeErr
			}
			if err != nil {
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
			inquire, err := relayInquire(cfg, "", src)
			if err != nil {
				return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
			}
			if !inquire.Found {
				outputs = append(outputs, fmt.Sprintf(`#NOTFOUND "%s"`, src))
				continue
			}
			out, err := os.OpenFile(dest, os.O_WRONLY, 0o644)
			if err != nil {
				return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
			}
			err = relayRestore(cfg, inquire.ID, src, out)
			if closeErr := out.Close(); err == nil {
				err = closeErr
			}
			if err != nil {
				return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
			}
			outputs = append(outputs, fmt.Sprintf(`#RESTORED "%s" "%s"`, inquire.ID, src))
		}
	}
	if len(outputs) == 0 {
		outputs = append(outputs, `#ERROR "" "no EBID or NULL entries found"`)
	}
	return outputs
}

func relayHandleInquire(cfg config, entries []inputEntry) []string {
	var outputs []string
	for _, e := range entries {
		switch e.Keyword {
		case "EBID":
			if len(e.Args) < 2 {
				continue
			}
			id := e.Args[0]
			src := e.Args[1]
			resp, err := relayInquire(cfg, id, src)
			if err != nil {
				return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
			}
			if resp.Found {
				outputs = append(outputs, fmt.Sprintf(`#BACKUP "%s" "%s"`, resp.ID, src))
			} else {
				outputs = append(outputs, fmt.Sprintf(`#NOTFOUND "%s" "%s"`, id, src))
			}
		case "NULL":
			if len(e.Args) < 1 {
				continue
			}
			src := e.Args[0]
			resp, err := relayInquire(cfg, "", src)
			if err != nil {
				return []string{fmt.Sprintf(`#ERROR "%s" "%s"`, src, err.Error())}
			}
			if resp.Found {
				outputs = append(outputs, fmt.Sprintf(`#BACKUP "%s" "%s"`, resp.ID, src))
			} else {
				outputs = append(outputs, fmt.Sprintf(`#NOTFOUND "%s"`, src))
			}
		}
	}
	if len(outputs) == 0 {
		outputs = append(outputs, "#NOTFOUND")
	}
	return outputs
}

func relayHandleDelete(cfg config, entries []inputEntry) []string {
	var outputs []string
	for _, e := range entries {
		if e.Keyword != "EBID" || len(e.Args) < 2 {
			continue
		}
		id := e.Args[0]
		src := e.Args[1]
		resp, err := relayDelete(cfg, id)
		if err != nil {
			outputs = append(outputs, fmt.Sprintf(`#ERROR "%s" "%s"`, id, err.Error()))
			continue
		}
		if resp.Deleted {
			outputs = append(outputs, fmt.Sprintf(`#DELETED "%s" "%s"`, id, src))
		} else {
			outputs = append(outputs, fmt.Sprintf(`#NOTFOUND "%s" "%s"`, id, src))
		}
	}
	if len(outputs) == 0 {
		outputs = append(outputs, "#NOTFOUND")
	}
	return outputs
}

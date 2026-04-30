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
	"io"
	"net/http"
	"net/url"
	"strings"
)

func relayBackup(cfg config, a args, src string, body io.Reader) (relayBackupResponse, error) {
	q := url.Values{}
	q.Set("sourcePath", src)
	q.Set("userID", a.userID)
	q.Set("backupID", a.backupID)
	q.Set("level", a.backupLevel)
	req, err := http.NewRequest(http.MethodPost, cfg.RelayURL+"/backup?"+q.Encode(), body)
	if err != nil {
		return relayBackupResponse{}, err
	}
	setRelayAuth(req, cfg)
	resp, err := relayClient().Do(req)
	if err != nil {
		return relayBackupResponse{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return relayBackupResponse{}, fmt.Errorf("relay backup failed: %s", strings.TrimSpace(string(data)))
	}
	var out relayBackupResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return relayBackupResponse{}, err
	}
	return out, nil
}

func relayRestore(cfg config, id, src string, out io.Writer) error {
	q := url.Values{}
	if id != "" {
		q.Set("id", id)
	}
	if src != "" {
		q.Set("sourcePath", src)
	}
	req, err := http.NewRequest(http.MethodGet, cfg.RelayURL+"/restore?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	setRelayAuth(req, cfg)
	resp, err := relayClient().Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("relay restore failed: %s", strings.TrimSpace(string(data)))
	}
	_, err = io.Copy(out, resp.Body)
	return err
}

func relayInquire(cfg config, id, src string) (relayInquireResponse, error) {
	q := url.Values{}
	if id != "" {
		q.Set("id", id)
	}
	if src != "" {
		q.Set("sourcePath", src)
	}
	req, err := http.NewRequest(http.MethodGet, cfg.RelayURL+"/inquire?"+q.Encode(), nil)
	if err != nil {
		return relayInquireResponse{}, err
	}
	setRelayAuth(req, cfg)
	resp, err := relayClient().Do(req)
	if err != nil {
		return relayInquireResponse{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return relayInquireResponse{}, fmt.Errorf("relay inquire failed: %s", strings.TrimSpace(string(data)))
	}
	var out relayInquireResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return relayInquireResponse{}, err
	}
	return out, nil
}

func relayDelete(cfg config, id string) (relayDeleteResponse, error) {
	q := url.Values{}
	q.Set("id", id)
	req, err := http.NewRequest(http.MethodDelete, cfg.RelayURL+"/delete?"+q.Encode(), nil)
	if err != nil {
		return relayDeleteResponse{}, err
	}
	setRelayAuth(req, cfg)
	resp, err := relayClient().Do(req)
	if err != nil {
		return relayDeleteResponse{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return relayDeleteResponse{}, fmt.Errorf("relay delete failed: %s", strings.TrimSpace(string(data)))
	}
	var out relayDeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return relayDeleteResponse{}, err
	}
	return out, nil
}

func setRelayAuth(req *http.Request, cfg config) {
	if cfg.RelayToken != "" {
		req.Header.Set("X-Backint-Relay-Token", cfg.RelayToken)
	}
}

func relayClient() *http.Client {
	return &http.Client{}
}

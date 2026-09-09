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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const (
	relayStreamVersionHeader = "X-Backint-Stream-Version"
	relayStreamVersion       = "1"
	relayStreamStatusTrailer = "X-Backint-Stream-Status"
	relayStreamSizeTrailer   = "X-Backint-Stream-Bytes"
	relayStreamHashTrailer   = "X-Backint-Stream-Sha256"
)

// The body is never buffered or spooled. New agents require the completion
// contract, so deploy the updated relay before deploying the updated agent.
func readRelayRestoreStream(resp *http.Response, out io.Writer) error {
	if resp.StatusCode != http.StatusOK || resp.Header.Get(relayStreamVersionHeader) != relayStreamVersion {
		return fmt.Errorf("relay restore does not provide the required completion protocol")
	}
	for _, name := range []string{relayStreamStatusTrailer, relayStreamSizeTrailer, relayStreamHashTrailer} {
		if _, ok := resp.Trailer[http.CanonicalHeaderKey(name)]; !ok {
			return fmt.Errorf("relay restore is missing its completion trailer declaration")
		}
	}
	digest := sha256.New()
	size, err := io.Copy(io.MultiWriter(out, digest), resp.Body)
	if err != nil {
		return fmt.Errorf("relay restore stream failed: %w", err)
	}
	for _, name := range []string{relayStreamStatusTrailer, relayStreamSizeTrailer, relayStreamHashTrailer} {
		if len(resp.Trailer.Values(name)) != 1 {
			return fmt.Errorf("relay restore has missing or ambiguous completion trailers")
		}
	}
	if resp.Trailer.Get(relayStreamStatusTrailer) != "complete" {
		return fmt.Errorf("relay restore producer did not confirm completion")
	}
	expectedSize, err := strconv.ParseInt(resp.Trailer.Get(relayStreamSizeTrailer), 10, 64)
	if err != nil || expectedSize < 0 || expectedSize != size {
		return fmt.Errorf("relay restore stream size does not match its completion trailer")
	}
	if resp.Trailer.Get(relayStreamHashTrailer) != hex.EncodeToString(digest.Sum(nil)) {
		return fmt.Errorf("relay restore stream checksum does not match its completion trailer")
	}
	return nil
}

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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// RelayTLSCapability lets the job reject older agents before starting HANA SQL.
const RelayTLSCapability = "BACKINT_RELAY_TLS_V1"

func decodeRelayCA(value string) ([]byte, error) {
	ca, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid BACKINT_RELAY_CA: expected base64-encoded PEM certificates")
	}
	if _, err := relayTrustPool(ca); err != nil {
		return nil, err
	}
	return ca, nil
}

func relayTrustPool(ca []byte) (*x509.CertPool, error) {
	// Do not augment system roots: each operation trusts only its pinned relay.
	pool := x509.NewCertPool()
	remaining := bytes.TrimSpace(ca)
	count := 0
	for len(remaining) > 0 {
		if !bytes.HasPrefix(remaining, []byte("-----BEGIN CERTIFICATE-----")) {
			return nil, errors.New("invalid BACKINT_RELAY_CA: expected PEM certificates only")
		}
		block, rest := pem.Decode(remaining)
		if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			return nil, errors.New("invalid BACKINT_RELAY_CA: expected PEM certificates only")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.New("invalid BACKINT_RELAY_CA certificate")
		}
		pool.AddCert(cert)
		count++
		remaining = bytes.TrimSpace(rest)
	}
	if count == 0 {
		return nil, errors.New("missing BACKINT_RELAY_CA pinned trust")
	}
	return pool, nil
}

func validateRelayClientConfig(cfg config) (*x509.CertPool, error) {
	u, err := url.Parse(cfg.RelayURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Opaque != "" ||
		u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" ||
		u.RawFragment != "" || strings.Contains(cfg.RelayURL, "#") || u.RawPath != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("invalid BACKINT_RELAY_URL: expected an HTTPS origin without credentials, query, fragment, or path")
	}
	host := u.Hostname()
	if !validRelayHost(host) || strings.HasSuffix(u.Host, ":") {
		return nil, errors.New("invalid BACKINT_RELAY_URL host")
	}
	if port := u.Port(); port != "" {
		n, err := strconv.ParseUint(port, 10, 16)
		if err != nil || n == 0 {
			return nil, errors.New("invalid BACKINT_RELAY_URL port")
		}
	}
	if strings.TrimSpace(cfg.RelayToken) == "" {
		return nil, errors.New("missing BACKINT_RELAY_TOKEN")
	}
	return relayTrustPool(cfg.RelayCA)
}

func validRelayHost(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}
	host = strings.TrimSuffix(host, ".")
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			switch {
			case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-':
			default:
				return false
			}
		}
	}
	return true
}

func relayRequestContext(cfg config) (context.Context, context.CancelFunc) {
	ctx := cfg.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.CommandTimeout > 0 {
		return context.WithTimeout(ctx, cfg.CommandTimeout)
	}
	return context.WithCancel(ctx)
}

func relayClient(cfg config) (*http.Client, error) {
	roots, err := validateRelayClientConfig(cfg)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		// Never route relay credentials or backup data through an environment proxy.
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			RootCAs:    roots,
			NextProtos: []string{"http/1.1"},
		},
		TLSHandshakeTimeout:    30 * time.Second,
		IdleConnTimeout:        2 * time.Minute,
		MaxIdleConns:           1,
		MaxIdleConnsPerHost:    1,
		MaxResponseHeaderBytes: 64 << 10,
		DisableCompression:     true,
	}
	return &http.Client{
		Transport: transport,
		// Redirects must not forward the operation token to another endpoint.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("backint relay redirects are forbidden")
		},
		// Large backup/restore bodies are bounded by the operation context,
		// not by a short transport-wide response/body timeout.
	}, nil
}

# hanadb-init

Init image for KubeDB HanaDB pods.

The image builds and installs the relay-only SAP HANA Backint agent used by
`hanadb-restic-plugin`. It installs only the `hdbbackint` binary into the shared
agent mount. The Restic binary stays in the KubeStash backup or restore job
image. This repository owns HANA's Backint command/input/output handling and
the HTTPS client; the plugin owns the relay server, repository credentials,
Restic execution, and backup metadata lookup.

Restore acknowledgments require a matching job-side relay completion protocol:
the agent verifies the final byte count and SHA-256 checksum after the producer
has completed successfully. Truncated streams, missing completion trailers, and
producer errors return `#ERROR` instead of `#RESTORED`.

Relay traffic requires TLS 1.3 with server/IP verification against the
operation-specific CA in `BACKINT_RELAY_CA` (base64 PEM) and a nonempty
`BACKINT_RELAY_TOKEN`. The agent rejects plaintext URLs, missing trust or
authentication, and redirects; it does not use environment HTTP proxies or
disable certificate verification. These settings arrive in the private
parameter file, not command-line arguments. The relay's private key stays in
the Job.

Rebuild and deploy this agent together with `hanadb-restic-plugin`. Pause
affected schedules and let active operations finish before updating the pair,
then resume them: there is no compatibility fallback to plaintext peers. The
agent advertises `BACKINT_RELAY_TLS_V1` in detailed `-V` output, which new Jobs
check before changing HANA configuration.

Standalone direct-Restic mode has been removed. Repository, password, binary,
and process-priority settings in an agent parameter file are rejected; use the
job-owned relay instead. The required settings are `BACKINT_RELAY_URL`,
`BACKINT_RELAY_TOKEN`, and `BACKINT_RELAY_CA`. `RESTIC_COMMAND_TIMEOUT` remains
the compatible name for the optional positive relay-request timeout, not a
local Restic command. Old `BACKINT_META_ROOT`, `BACKINT_SPOOL_ROOT`, and `TMPDIR`
hints are accepted but ignored; the client does not store metadata or spool data.

## Runtime

The default entrypoint is `/init-script/run.sh`.

Default paths:

```text
BACKINT_ROOT=/hana/mounts/backint-agent
BACKINT_TARGET=/hana/mounts/backint-agent/bin/hdbbackint
HDBBACKINT_PATH=/usr/sap/<SID>/SYS/global/hdb/opt/hdbbackint
```

The script always creates the agent directory and installs:

```text
/hana/mounts/backint-agent/bin/hdbbackint
/hana/mounts/backint-agent/meta
/hana/mounts/backint-agent/spool
/hana/mounts/backint-agent/tmp
```

If the HANA opt directory is mounted and writable in the init container, the
script also creates:

```text
/usr/sap/<SID>/SYS/global/hdb/opt/hdbbackint -> /hana/mounts/backint-agent/bin/hdbbackint
```

If that path is not writable from the init container, the main HanaDB container
must recreate the symlink during startup.

The existing install directories and their ownership/permissions are preserved
for mounted-volume compatibility; the relay-only agent does not use the metadata
or spool directories.

## Build

```bash
make container
```

Build the agent binary locally:

```bash
make build
```

Local client/TLS/stream regression fixtures live in the sibling `kubedb.dev/tests`
repository under `hack/hanadb-backint`; `go test ./...` here is only a package
compilation check when no local test files are present.

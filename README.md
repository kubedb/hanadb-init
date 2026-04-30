# hanadb-init

Init image for KubeDB HanaDB pods.

The image builds and installs the SAP HANA Backint-compatible agent used by
`hanadb-restic-plugin`. It installs only the `hdbbackint` binary into the shared
agent mount. The Restic binary stays in the KubeStash backup or restore job
image.

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

## Build

```bash
make container
```

Build the agent binary locally:

```bash
make build
```

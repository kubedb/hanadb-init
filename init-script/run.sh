#!/usr/bin/env bash

set -euo pipefail

BACKINT_ROOT="${BACKINT_ROOT:-/hana/mounts/backint-agent}"
BACKINT_BIN_DIR="${BACKINT_BIN_DIR:-${BACKINT_ROOT}/bin}"
BACKINT_SOURCE="${BACKINT_SOURCE:-/tmp/agent/hdbbackint}"
BACKINT_TARGET="${BACKINT_TARGET:-${BACKINT_BIN_DIR}/hdbbackint}"
BACKINT_UID="${BACKINT_UID:-12000}"
BACKINT_GID="${BACKINT_GID:-79}"
HANA_SID="${HANA_SID:-${SID:-HXE}}"
HDBBACKINT_PATH="${HDBBACKINT_PATH:-/usr/sap/${HANA_SID}/SYS/global/hdb/opt/hdbbackint}"

log() {
    local level="$1"
    local message="$2"
    echo "$(date +"%Y/%m/%d %T") [${level}] ${message}"
}

if [[ ! -x "${BACKINT_SOURCE}" ]]; then
    log "ERROR" "Backint source binary is missing or not executable: ${BACKINT_SOURCE}"
    exit 1
fi

install -d -m 0775 "${BACKINT_BIN_DIR}" "${BACKINT_ROOT}/meta" "${BACKINT_ROOT}/spool" "${BACKINT_ROOT}/tmp"
install -m 0755 "${BACKINT_SOURCE}" "${BACKINT_TARGET}"

if [[ "$(id -u)" == "0" ]]; then
    chown -R "${BACKINT_UID}:${BACKINT_GID}" "${BACKINT_ROOT}"
fi

log "INFO" "Installed HanaDB Backint agent at ${BACKINT_TARGET}"

hdbbackint_dir="$(dirname "${HDBBACKINT_PATH}")"
if [[ -d "${hdbbackint_dir}" && -w "${hdbbackint_dir}" ]]; then
    ln -sfn "${BACKINT_TARGET}" "${HDBBACKINT_PATH}"
    log "INFO" "Linked ${HDBBACKINT_PATH} to ${BACKINT_TARGET}"
else
    log "INFO" "Skipped HANA opt-path symlink because ${hdbbackint_dir} is not writable in this container"
fi

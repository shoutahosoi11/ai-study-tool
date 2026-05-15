#!/usr/bin/env bash
set -euo pipefail

database_url="${1:-}"
migrations_dir="${2:-backend/db/migrations}"

if [ -z "${database_url}" ]; then
  echo "usage: $0 DATABASE_URL [MIGRATIONS_DIR]" >&2
  exit 2
fi

if [ ! -d "${migrations_dir}" ]; then
  echo "migrations directory not found: ${migrations_dir}" >&2
  exit 2
fi

checksum_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi
  shasum -a 256 "$1" | awk '{print $1}'
}

sql_quote() {
  printf "%s" "$1" | sed "s/'/''/g"
}

psql "${database_url}" -v ON_ERROR_STOP=1 -c "
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  checksum TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
"

find "${migrations_dir}" -maxdepth 1 -type f -name '*.sql' | sort | while IFS= read -r migration; do
  version="$(basename "${migration}")"
  checksum="$(checksum_file "${migration}")"
  quoted_version="$(sql_quote "${version}")"
  quoted_checksum="$(sql_quote "${checksum}")"
  applied_checksum="$(psql "${database_url}" -v ON_ERROR_STOP=1 -At -c "SELECT checksum FROM schema_migrations WHERE version = '${quoted_version}'")"

  if [ "${applied_checksum}" = "${checksum}" ]; then
    echo "migration already applied: ${version}"
    continue
  fi

  if [ -n "${applied_checksum}" ]; then
    echo "migration checksum mismatch: ${version}" >&2
    echo "recorded: ${applied_checksum}" >&2
    echo "current:  ${checksum}" >&2
    exit 1
  fi

  if [ "${BASELINE_MIGRATIONS:-false}" = "true" ]; then
    echo "baselining previously applied migration: ${version}"
    psql "${database_url}" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations (version, checksum) VALUES ('${quoted_version}', '${quoted_checksum}');"
    continue
  fi

  echo "applying migration: ${version}"
  temp_file="$(mktemp)"
  {
    echo "BEGIN;"
    cat "${migration}"
    echo
    echo "INSERT INTO schema_migrations (version, checksum) VALUES ('${quoted_version}', '${quoted_checksum}');"
    echo "COMMIT;"
  } > "${temp_file}"

  if ! psql "${database_url}" -v ON_ERROR_STOP=1 -f "${temp_file}"; then
    rm -f "${temp_file}"
    exit 1
  fi
  rm -f "${temp_file}"
done

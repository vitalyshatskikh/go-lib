#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE ROLE replicator WITH LOGIN REPLICATION PASSWORD '${REPLICATION_PASSWORD:-postgres}';
EOSQL

cat >> "$PGDATA/pg_hba.conf" <<-EOF
host replication replicator 0.0.0.0/0 md5
host all all 0.0.0.0/0 md5
EOF

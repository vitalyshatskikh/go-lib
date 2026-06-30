#!/bin/bash
set -e

PRIMARY_HOST="${REPLICA_OF:-postgres-primary}"
REPLICATION_USER="${REPLICATION_USER:-replicator}"
REPLICATION_PASSWORD="${REPLICATION_PASSWORD:-${POSTGRES_PASSWORD:-postgres}}"
DATA_DIR="/var/lib/postgresql/data"

if [ -z "$(ls -A "$DATA_DIR" 2>/dev/null)" ]; then
    echo "Data directory is empty. Initializing replica from ${PRIMARY_HOST}..."

    until pg_isready -h "$PRIMARY_HOST" -U postgres 2>/dev/null; do
        echo "Waiting for primary to be ready..."
        sleep 2
    done

    echo "Running pg_basebackup from ${PRIMARY_HOST}..."
    PGPASSWORD="$REPLICATION_PASSWORD" \
    pg_basebackup -h "$PRIMARY_HOST" \
        -U "$REPLICATION_USER" \
        -D "$DATA_DIR" \
        -P -v -R -X stream

    AUTO_CONF="${DATA_DIR}/postgresql.auto.conf"
    if [ -f "$AUTO_CONF" ] && grep -q "primary_conninfo" "$AUTO_CONF"; then
        sed -i "s/primary_conninfo = '\(.*\)'/primary_conninfo = '\1 password=${REPLICATION_PASSWORD}'/" "$AUTO_CONF"
    fi

    chown -R postgres:postgres "$DATA_DIR"
    chmod 700 "$DATA_DIR"

    echo "Replica initialized successfully."
fi

exec /usr/local/bin/docker-entrypoint.sh "$@"

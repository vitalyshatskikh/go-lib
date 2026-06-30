#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.yml"

PRIMARY_SERVICE="postgres-primary"
REPLICA_SERVICE="postgres-replica"
REPLICATION_USER="replicator"
REPLICATION_PASSWORD="${REPLICATION_PASSWORD:-postgres}"
DATA_DIR="/var/lib/postgresql/data"

# Detect Docker Compose variant (v1 plugin vs standalone v1 binary)
if docker compose version &>/dev/null; then
    DOCKER_COMPOSE="docker compose"
elif docker-compose --version &>/dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    echo "ERROR: Neither 'docker compose' nor 'docker-compose' found."
    exit 1
fi

echo "=== PostgreSQL Manual Switchover ==="
echo "Demoting:  ${PRIMARY_SERVICE}"
echo "Promoting: ${REPLICA_SERVICE}"
echo ""

# Step 1 — Stop the old primary (prevents split-brain)
echo "[1/4] Stopping old primary..."
$DOCKER_COMPOSE -f "$COMPOSE_FILE" stop "${PRIMARY_SERVICE}"

# Step 2 — Promote replica to new primary
echo "[2/4] Promoting replica to primary..."
$DOCKER_COMPOSE -f "$COMPOSE_FILE" exec -T "${REPLICA_SERVICE}" \
    psql -U postgres -tAc "SELECT pg_promote();"

echo "Waiting for new primary to be ready..."
until $DOCKER_COMPOSE -f "$COMPOSE_FILE" exec -T "${REPLICA_SERVICE}" \
    pg_isready -U postgres 2>/dev/null; do
    sleep 1
done

IS_RECOVERY=$($DOCKER_COMPOSE -f "$COMPOSE_FILE" exec -T "${REPLICA_SERVICE}" \
    psql -U postgres -tAc "SELECT pg_is_in_recovery();" | tr -d '[:space:]')
if [ "$IS_RECOVERY" != "f" ]; then
    echo "ERROR: Replica did not promote successfully."
    exit 1
fi
echo "New primary is accepting connections."

# Step 3 — Re-initialize old primary's data as a replica of the new primary
echo "[3/4] Re-initializing old primary as replica..."
$DOCKER_COMPOSE -f "$COMPOSE_FILE" run --rm --no-deps \
    "${PRIMARY_SERVICE}" \
    bash -c "
        rm -rf ${DATA_DIR}/* &&
        PGPASSWORD=${REPLICATION_PASSWORD} pg_basebackup \
            -h ${REPLICA_SERVICE} \
            -U ${REPLICATION_USER} \
            -D ${DATA_DIR} \
            -P -v -R -X stream &&
        sed -i \"s/primary_conninfo = '\\\\(.*\\\)'/primary_conninfo = '\\\1 password=${REPLICATION_PASSWORD}'/\" ${DATA_DIR}/postgresql.auto.conf
    "

# Step 4 — Start the old primary as a replica
echo "[4/4] Starting old primary as replica..."
$DOCKER_COMPOSE -f "$COMPOSE_FILE" start "${PRIMARY_SERVICE}"

echo ""
echo "=== Switchover Complete ==="
echo "New primary: ${REPLICA_SERVICE}"
echo "New replica: ${PRIMARY_SERVICE}"

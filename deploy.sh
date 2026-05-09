#!/usr/bin/env bash
set -euo pipefail

COMPOSE_DIR="/opt/new-api"
IMAGE_TAG="${1:?Usage: deploy.sh <image-tag>}"

cd "$COMPOSE_DIR"

echo "=== Deploying clearmind1/new-api:${IMAGE_TAG}-amd64 ==="

docker compose pull new-api
docker compose up -d new-api

echo "Waiting for health check..."
for i in $(seq 1 30); do
    STATUS=$(docker inspect --format='{{.State.Health.Status}}' new-api 2>/dev/null || echo "not-running")
    if [ "$STATUS" = "healthy" ]; then
        echo "Deploy OK: ${IMAGE_TAG}"
        exit 0
    fi
    sleep 2
done

echo "WARNING: Health check did not pass within 60 seconds. Current status: $STATUS"
echo "Container logs:"
docker compose logs --tail=50 new-api
exit 1
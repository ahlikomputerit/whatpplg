#!/bin/bash
set -euo pipefail

# ============================================================
# Publish WA Gateway Docker Image
# ============================================================
# Usage:
#   ./publish.sh                    # push to default (ghcr)
#   ./publish.sh docker.io/user/wa-gateway
#
# Prerequisites:
#   docker login ghcr.io -u USER --password-stdin
# ============================================================

IMAGE="${1:-ghcr.io/ahlikomputerit/whatpplg}"
TAG="${2:-latest}"
FULL="${IMAGE}:${TAG}"

echo "==> Building image: ${FULL}"
docker build -f Dockerfile.release -t "${FULL}" .

echo "==> Pushing image: ${FULL}"
docker push "${FULL}"

echo ""
echo "✅ Done!"
echo ""
echo "Di PC target, tinggal jalankan:"
echo ""
echo "  curl -sL https://raw.githubusercontent.com/ahlikomputerit/whatpplg/main/install.sh | bash"
echo ""
echo "Atau manual:"
echo ""
echo "  docker pull ${FULL}"
echo "  docker run -d \\"
echo "    --name wa-gateway \\"
echo "    -p 8080:8080 \\"
echo "    -v wa-data:/app/data \\"
echo "    -v \$(pwd)/config.yaml:/app/config.yaml \\"
echo "    ${FULL}"
echo ""

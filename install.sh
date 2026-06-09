#!/bin/bash
set -euo pipefail

# ============================================================
# WA Gateway Installer — one-line installation
#
# Usage:
#   curl -sL https://raw.githubusercontent.com/ahlikomputerit/whatpplg/main/install.sh | bash
#
# Or with options:
#   curl -sL https://...install.sh | bash -s -- --port 8080 --dir /opt/wa-gateway
# ============================================================

# ---- Defaults ----
IMAGE="${IMAGE:-ghcr.io/ahlikomputerit/whatpplg}"
TAG="${TAG:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/wa-gateway}"
PORT="${PORT:-8080}"
API_KEY="${API_KEY:-}"  # auto-generate if empty

# Parse args
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dir)    INSTALL_DIR="$2"; shift 2 ;;
    --port)   PORT="$2"; shift 2 ;;
    --api-key) API_KEY="$2"; shift 2 ;;
    --image)  IMAGE="$2"; shift 2 ;;
    --tag)    TAG="$2"; shift 2 ;;
    --help|-h)
      echo "Usage: curl -sL https://...install.sh | bash -s -- [options]"
      echo ""
      echo "Options:"
      echo "  --dir DIR       Install directory (default: ~/wa-gateway)"
      echo "  --port PORT     Port to expose (default: 8080)"
      echo "  --api-key KEY   API key (auto-generated if empty)"
      echo "  --image IMG     Docker image (default: ghcr.io/ahlikomputerit/whatpplg)"
      echo "  --tag TAG       Image tag (default: latest)"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

FULL_IMAGE="${IMAGE}:${TAG}"

# ---- Prerequisites ----
echo "🔍 Checking prerequisites..."

if ! command -v docker &>/dev/null; then
  echo "❌ Docker not found. Install Docker first:"
  echo "   https://docs.docker.com/engine/install/"
  exit 1
fi

# ---- Setup directory ----
echo "📁 Creating install directory: ${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}"
cd "${INSTALL_DIR}"

# ---- Generate config ----
if [[ -z "$API_KEY" ]]; then
  API_KEY="wa-$(openssl rand -hex 16 2>/dev/null || head -c 32 /dev/urandom | xxd -p -c 32)"
fi

echo "⚙️  Generating config.yaml..."

cat > config.yaml <<YAML
server:
  port: ${PORT}
  api_key: "${API_KEY}"

whatsapp:
  db_path: "/app/data/wa_session.db"
  preset: "moderate"
  config:
    enable_typo_injection: true
    enable_zero_width: true
    enable_punctuation_vary: true

sources:
  - name: "default"
    mode: "api"
    api_key: "${API_KEY}"

queue:
  type: "memory"
  max_size: 10000
YAML

# ---- Pull image ----
echo "🐳 Pulling image: ${FULL_IMAGE}"
docker pull "${FULL_IMAGE}"

# ---- Stop existing container ----
docker rm -f wa-gateway 2>/dev/null || true

# ---- Run container ----
echo "🚀 Starting WA Gateway..."
docker run -d \
  --name wa-gateway \
  --restart unless-stopped \
  -p "${PORT}:${PORT}" \
  -v wa-gateway-data:/app/data \
  -v "${INSTALL_DIR}/config.yaml:/app/config.yaml:ro" \
  -e TZ=Asia/Jakarta \
  "${FULL_IMAGE}"

# ---- Health check ----
echo "⏳ Waiting for service to start..."
sleep 3

if docker ps --format '{{.Names}}' | grep -q wa-gateway; then
  echo ""
  echo "✅ WA Gateway installed successfully!"
  echo ""
  echo "   API:       http://localhost:${PORT}/api/v1/send"
  echo "   Health:    http://localhost:${PORT}/api/v1/health"
  echo "   API Key:   ${API_KEY}"
  echo "   Config:    ${INSTALL_DIR}/config.yaml"
  echo "   Data:      wa-gateway-data (Docker volume)"
  echo ""
  echo "📖 First time? Scan QR code untuk login WhatsApp:"
  echo "   docker logs -f wa-gateway"
  echo ""
  echo "📝 Example send:"
  echo "   curl -X POST http://localhost:${PORT}/api/v1/send \\"
  echo "     -H \"Authorization: Bearer ${API_KEY}\" \\"
  echo "     -H \"Content-Type: application/json\" \\"
  echo "     -d '{\"to\":\"62812xxxx\",\"message\":\"Hello!\"}'"
  echo ""
else
  echo "❌ Failed to start. Check logs:"
  echo "   docker logs wa-gateway"
  exit 1
fi

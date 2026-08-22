#!/bin/bash
set -e

IMAGE_NAME=${1:-task156-subtitleqa}
DOCKER_PLATFORM=${2:-linux/amd64}

docker build --platform "$DOCKER_PLATFORM" -f benzhi.Dockerfile -t "$IMAGE_NAME" .

echo ""
echo "✅ Docker image '$IMAGE_NAME' built successfully for $DOCKER_PLATFORM!"
echo ""
echo "📋 Verify (must print 'smoke test passed'):"
echo "  • docker run --rm $IMAGE_NAME /app/subtitleqa --smoke-test"
echo ""

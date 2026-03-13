#!/bin/sh
set -e

echo "🚀 Installing Bruce..."

# Create directory structure
mkdir -p bruce/data
cd bruce

# Download docker-compose.yml
echo "📥 Downloading docker-compose.yml..."
curl -fsSL https://raw.githubusercontent.com/DemitriusQuadros/bruce/main/install/docker-compose.yml -o docker-compose.yml

# Download config.yml
echo "📥 Downloading config.yml..."
curl -fsSL https://raw.githubusercontent.com/DemitriusQuadros/bruce/main/install/config.yml -o config.yml

echo ""
echo "✅ Bruce installed successfully!"
echo ""
echo "📝 Next steps:"
echo "  1. Edit config.yml and add your API keys (Claude and/or Gemini)"
echo "  2. Run: docker compose up -d"
echo "  3. Open: http://localhost:8080"
echo ""
echo "📖 Documentation: https://github.com/DemitriusQuadros/bruce"

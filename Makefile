.PHONY: all dev dev-mock build-server run-server run-server-mock stop build-web serve-web \
        build-android build-android-release build-ios-release \
        run-android install-apk devices test-integration clean health logs \
        docker-build deploy deploy-restart deploy-logs deploy-status deploy-ssh

# Default - just build server
all: build-server

# GeoIP database URL (community mirror of MaxMind GeoLite2)
GEOIP_URL = https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-Country.mmdb
GEOIP_FILE = server/GeoLite2-Country.mmdb

# Download GeoIP database if missing
download-geoip:
	@if [ ! -f $(GEOIP_FILE) ]; then \
		echo "=== Downloading GeoLite2-Country.mmdb... ==="; \
		curl -sL -o $(GEOIP_FILE) $(GEOIP_URL); \
		echo "=== GeoIP database downloaded ==="; \
	else \
		echo "=== GeoIP database exists ==="; \
	fi

# Build Go server (auto-downloads GeoIP if missing)
build-server: download-geoip
	cd server && go build -o server ./cmd/server/...

# Dev: build server + run everything (requires SUBSCRIPTION_URL)
dev: build-server
	@echo "=== Starting Go server on :8080 ==="
	@cd server && SUBSCRIPTION_URL="${SUBSCRIPTION_URL}" nohup ./server > /tmp/vpn-server.log 2>&1 &
	@sleep 1
	@curl -s http://localhost:8080/health || (echo "Server failed to start. Check /tmp/vpn-server.log" && exit 1)
	@echo "=== Server running on http://localhost:8080 ==="
	@echo "=== Starting Flutter web ==="
	@SUBSCRIPTION_URL="${SUBSCRIPTION_URL}" cd client && flutter run -d chrome

# Dev with mock configs (no SUBSCRIPTION_URL needed, for UI testing)
dev-mock: build-server
	@echo "=== Starting Go server with MOCK_CONFIGS=true ==="
	@cd server && MOCK_CONFIGS=true nohup ./server > /tmp/vpn-server.log 2>&1 &
	@sleep 1
	@curl -s http://localhost:8080/health || (echo "Server failed to start" && exit 1)
	@curl -s http://localhost:8080/api/configs | head -c 200
	@echo ""
	@echo "=== Mock server running on http://localhost:8080 ==="
	@echo "=== Starting Flutter web ==="
	@cd client && flutter run -d chrome

# Run Go server only (auto-builds first)
run-server: build-server
	cd server && ./server

# Run Go server with mock configs (auto-builds first)
run-server-mock: build-server
	cd server && MOCK_CONFIGS=true ./server

# Serve built web (run build-web first)
serve-web:
	@cd client/build/web && python3 -m http.server 5000

# Build Flutter web
build-web:
	cd client && flutter build web

APK_PATH = client/build/app/outputs/flutter-apk/app-debug.apk
APK_RELEASE_PATH = client/build/app/outputs/flutter-apk/app-release.apk
IPA_PATH = client/build/ios/ipa/vpn_client.ipa

# Build Android APK (debug, for local dev)
#   make build-android                       # default: use debug menu
#   make build-android SERVER_URL=http://192.168.1.42:8080   # Wi-Fi адрес сервера
build-android:
	cd client && flutter build apk --debug $(if $(SERVER_URL),--dart-define=SERVER_URL=$(SERVER_URL),)
	@echo "=== APK: $(APK_PATH) ==="

# Build Android APK (release, requires signing config)
# For CI: set ANDROID_KEYSTORE_PATH, ANDROID_STORE_PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD
# For local: put keystore.jks in client/android/app/ and create key.properties
build-android-release:
	cd client && flutter build apk --release $(if $(SERVER_URL),--dart-define=SERVER_URL=$(SERVER_URL),)
	@echo "=== Release APK: $(APK_RELEASE_PATH) ==="

# Build iOS IPA (release, requires macOS + Apple Developer account)
# For CI: set IOS_TEAM_ID, IOS_P12_BASE64, IOS_P12_PASSWORD, IOS_PROVISIONING_BASE64
build-ios-release:
	cd client/ios && pod install
	cd client && flutter build ipa --release $(if $(SERVER_URL),--dart-define=SERVER_URL=$(SERVER_URL),)
	@echo "=== IPA: $(IPA_PATH) ==="

# Build + install via flutter (требуется подключённый телефон)
run-android: build-android
	cd client && flutter install --use-application-binary build/app/outputs/flutter-apk/app-debug.apk

# Build + install via adb (резервный способ, если flutter install не сработал)
install-apk: build-android
	adb install -r $(APK_PATH)
	@echo "=== Установлено ==="

# Список подключённых Android устройств
devices:
	adb devices -l

# Run integration tests (starts/stops server automatically)
test-integration:
	cd server && bash test_integration.sh

# Check server health
health:
	@curl -s http://localhost:8080/health || echo "Server not running"

# Stop all processes
stop:
	@pkill -f "server" 2>/dev/null || true
	@pkill -f "flutter" 2>/dev/null || true

# Clean up
clean:
	rm -f server/server
	cd client && flutter clean

# Remove GeoIP database (re-downloaded on next build)
clean-geoip:
	rm -f $(GEOIP_FILE)

# Show logs
logs:
	@tail -f /tmp/vpn-server.log

# =============================================================================
# Remote server deployment (Docker)
# =============================================================================
# Remote server connection — set these via env or .env.local (not committed)
#   export SSH_HOST=your-server.com SSH_PORT=22 SSH_USER=deploy DOMAIN=your.domain.com
SSH_HOST    ?=
SSH_PORT    ?= 22
SSH_USER    ?= deploy
SSH_KEY     ?= $(HOME)/.ssh/id_ed25519
DOMAIN      ?=
# Build Docker image locally
docker-build:
	docker build -t vpn-server server/

# Push source to remote via rsync and build/run via docker-compose
deploy:
	@echo "=== Deploying to $(SSH_HOST):$(SSH_PORT) ==="
	@rsync -avz --delete \
		-e "ssh -p $(SSH_PORT) -i $(SSH_KEY)" \
		--exclude '.env' \
		--exclude '.git' \
		--exclude '/server' \
		--exclude '/vpn-server' \
		--exclude '/vpn-test' \
		server/ \
		$(SSH_USER)@$(SSH_HOST):~/vpn-server/
	@echo "=== Building and starting Docker container ==="
	@ssh -p $(SSH_PORT) -i $(SSH_KEY) $(SSH_USER)@$(SSH_HOST) \
		"cd ~/vpn-server && \
		 if [ ! -f .env ]; then \
		   cp .env.example .env && \
		   echo '=== .env CREATED from .env.example — EDIT IT: ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) \"nano ~/vpn-server/.env\"' && \
		   exit 1; \
		 fi && \
		 DOMAIN_VAL=$$(grep '^DOMAIN=' .env | head -1 | cut -d= -f2) && \
		 export SUBDOMAIN=$${DOMAIN_VAL%.duckdns.org} && \
		 ([ -f GeoLite2-Country.mmdb ] || curl -sL -o GeoLite2-Country.mmdb \
		   https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-Country.mmdb) && \
		 docker compose build --pull && docker compose up -d"
	@echo "=== Waiting for server to initialize (up to 60s) ==="
	@for i in $$(seq 1 12); do \
		curl -sf --max-time 5 https://$(DOMAIN):8443/health > /dev/null 2>&1 || \
		curl -sf --max-time 5 http://$(SSH_HOST):8080/health > /dev/null 2>&1; \
		if [ $$? -eq 0 ]; then \
			echo "=== Server is healthy (https://$(DOMAIN):8443) ===" && exit 0; \
		fi; \
		sleep 5; \
	done; \
	echo "WARN: health check timeout"

# Restart container on remote (rebuilds if code changed)
deploy-restart:
	@ssh -p $(SSH_PORT) -i $(SSH_KEY) $(SSH_USER)@$(SSH_HOST) \
		"cd ~/vpn-server && docker compose up -d --build && echo '=== Restarted ==='"

# View logs from remote
deploy-logs:
	@ssh -p $(SSH_PORT) -i $(SSH_KEY) $(SSH_USER)@$(SSH_HOST) \
		"cd ~/vpn-server && docker compose logs -f"

# Check container status on remote
deploy-status:
	@ssh -p $(SSH_PORT) -i $(SSH_KEY) $(SSH_USER)@$(SSH_HOST) \
		"cd ~/vpn-server && docker compose ps && echo '---' && docker compose logs --tail=10"

# SSH to remote server
deploy-ssh:
	ssh -p $(SSH_PORT) -i $(SSH_KEY) $(SSH_USER)@$(SSH_HOST)

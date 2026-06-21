# План работ

> Обновлено: 2026-06-21
> Статусы: ⬜ Pending · 🔄 In Progress · ✅ Done · ❌ Cancelled

---

## ✅ Фаза 0 — Безопасность — ВЫПОЛНЕНО

- [x] **SCRUB: Удалить credentials из PLAN.md** — IP, порт SSH, username сервера
- [x] **CONFIG: Вынести `InsecureSkipVerify` в настройки** — добавлен флаг `SKIP_VERIFY_TLS` (default: true)
- [x] **CORS: Сделать `AllowOrigin` конфигурируемым** — переменная окружения `CORS_ORIGINS` (default: *)

---

## ✅ Фаза 0.5 — Убрать хардкод домена — ВЫПОЛНЕНО

- [x] **Flutter client** — URL только localhost по умолчанию
- [x] **Caddyfile** — домен через `{$DOMAIN}` (env)
- [x] **docker-compose** — Caddy + DuckDNS через env
- [x] **build-android.yml / deploy.yml / Makefile** — всё через vars/env

---

## ✅ Фаза 1 — Тесты — ВЫПОЛНЕНО

### Go unit-тесты
- [x] **parser** — 24 теста: парсинг VLESS/VMess/Trojan/SS, subscription (base64/JSON/plain), порты, id, raw_link
- [x] **resolver** — тесты с мок-резолвером
- [x] **geo** — 6 тестов: nil GeoDB, пустые конфиги, invalid path, nil receiver, invalid IP, non-RU filter
- [x] **config** — 11 тестов: parseDotEnv (normal, quoted, comments, whitespace, no-override, malformed, empty key), LoadConfig (defaults, env vars)
- [x] **tester** — 6 тестов: ParseUUID (valid, zeros, Fs, invalid length, random), Unhex
- [x] **pipeline** — 6 тестов: mock configs (non-RU, sorted, fastest DE), Run(), NoSubscriptionURL, filterReality

### Dart тесты
- [x] **VpnConfig** — 12+ тестов: fromJson, toJson, fromRawLink
- [x] **WidgetTest** — тесты отображения состояний HomeScreen

### Остаётся
- [ ] **Добавить `golangci-lint` в CI** — перед deploy workflow
- [ ] **Добавить `flutter analyze` в CI**
- [ ] **ApiClient** — unit-тесты с мок-ответами (mockito/mocktail)

---

## ⬜ Фаза 2 — Архитектура (Go server)

### Сервер (Go) — в основном выполнено
- [x] **Рефакторинг `cache.refresh()`** — выделен модуль `internal/pipeline/pipeline.go`
  - `fetcher.Fetch()` → `parser.ParseAll()` → `tester.TestAll()` → `geo.Filter()` → `reality.Filter()` → `sort.ByLatency()`
- [x] **Вынести REALITY filter** — в `pipeline.filterReality()`
- [x] **`parseDotEnv` поддержка `\r\n`** — добавлена нормализация
- [x] **Сортировка в loadMockConfigs** — локальная, без дублирования
- [ ] **Добавить `net.Resolver` с DoH** — опциональный DNS-over-HTTPS
- [ ] **Обновить/удалить устаревший `docker-compose.prod.yml`**

### Клиент (Flutter/Dart)
- [x] **Пустая `domain/`** — удалена (директории больше нет)
- [ ] **Выделить persistence из ApiClient**:
  - Создать `StorageService` (обёртка над `SharedPreferences`)
  - ApiClient принимает `StorageService` или URL через конструктор
- [ ] **Добавить `initialize()` в интерфейс `VpnService`** — или статический factory
- [ ] **Заменить `hashCode.toString()` на стабильный id** — `server:port`
- [ ] **Добавить `Result<VpnConfig>` или sealed class для ответов ApiClient**

---

## ⬜ Фаза 4 — Code Quality & Cleanup

### Разделение home_screen.dart
- [x] **Вынести `_ServerInfoCard`** → `presentation/widgets/server_info_card.dart`
- [x] **Вынести `_DebugSheet`** → `presentation/widgets/debug_sheet.dart`
- [ ] Вынести строки UI в константы/локализацию

### Остальное
- [ ] **Добавить exponential backoff** в `fetcher.go` (1s → 2s → 4s)
- [ ] **Добавить `mounted` check в `web_vpn_service.dart`**
- [ ] **Убрать мутацию `cfg.Country`** внутри `geo.FilterNonRussia()`
- [ ] **Обновить go.mod** — подтянуть актуальные версии зависимостей
- [ ] **Добавить pre-commit hooks** — `.githooks/pre-commit` с `go fmt` и `dart format`

---

## ⬜ Фаза 4 — Новые фичи

- [ ] **Выбор конкретного сервера из списка** — продвинутый режим
- [ ] **Поддержка REALITY в Flutter клиенте** — uTLS/Xray core
- [ ] **История подключений** — логи соединений, статистика
- [ ] **Push-уведомления о статусе сервера** — Firebase Cloud Messaging
- [ ] **Тёмная тема** — `ThemeMode.dark`
- [ ] **Авто-подключение при запуске**
- [ ] **WireGuard protocol support**
- [ ] **Multi-user support** — per-user config cache

---

## 🟡 Фаза 8 — Whitelist bypass (отложено, реализация не на один день)

### Как на самом деле работает ТСПУ с белыми списками
Двухуровневый фильтр:
1. **CIDR (IP-уровень)** — пакеты на не-whitelisted IP просто дропаются на уровне маршрутизатора
2. **SNI (прикладной уровень)** — если IP пропущен, проверяется SNI в TLS ClientHello

**REALITY обходит SNI** (маскируется под `yandex.ru`), но **НЕ обходит IP-фильтр**.

### Единственный работающий метод (2026): Chain (Relay)
```
Клиент → [Российский VPS с whitelisted IP] → [Сервер (EU)] → Интернет
                                     ↓
                              Российские сайты (DIRECT)
```

- **Российский VPS**: Timeweb / VDSina / Selectel (300-500 руб/мес)
- **Xray на релее**: inbound от клиента → outbound (VLESS+REALITY) на сервер в EU
- **Direct для RU**: Яндекс, ВК, Госуслуги и т.д. — напрямую с релея
- **Yandex Cloud — не работает**: AS Yandex.Cloud блокируется отдельно от AS Yandex LLC

### Резервный метод: Bootstrap конфиги в APK
- Вшить 2-3 публичных VLESS+REALITY конфига в `assets/bootstrap_configs.json`
- При первом запуске: если сервер недоступен → подключиться через bootstrap
- После подключения — обновить конфиги с API сервера
- Минус: конфиги могут умереть, надо обновлять

### Cloudflare — не работает (заблокирован ТСПУ с 2025)

### Что нужно будет сделать
- [ ] **Выбрать провайдера** для российского VPS (Timeweb / VDSina / Selectel)
- [ ] **Настроить Xray relay** на российском VPS (inbound от клиента → outbound на наш сервер)
- [ ] **Docker-образ для релея** или ansible-скрипт для быстрого развёртывания
- [ ] **Обновить TODO**: routing правила для DIRECT (российские сайты) через relay
- [ ] **CI**: добавить деплой конфига релея
- [ ] **Bootstrap конфиги** (резерв): вшить в APK публичные REALITY конфиги как fallback

---

## ⬜ Фаза 9 — Cloudflare WARP генерация конфигов

### Цель
Генерировать собственные WARP (WireGuard) конфиги через Cloudflare API, чтобы не зависеть от сторонних подписок или дополнять их.

### Как работает
Cloudflare WARP использует WireGuard. Регистрация устройства:
- `POST https://api.cloudflareclient.com/v0a<version>/reg`
- Возвращает приватный ключ, адрес, DNS
- Публичный ключ Cloudflare фиксированный

### Что нужно сделать
- [ ] **Сервер: WARP генератор** — новый пакет `internal/warp/` 
  - Регистрация нового устройства через Cloudflare API
  - Парсинг ответа в формате WireGuard config
  - Периодическая регенерация (конфиги живут N дней)
- [ ] **Сервер: добавить WARP конфиги в пул** — объединять с existing конфигами из подписок
- [ ] **Клиент: WireGuard поддержка** — для подключения к WARP конфигам (либо через Xray-core, либо напрямую)
- [ ] **Клиент: UI метка "WARP"** — отличать WARP конфиги от прокси

---

## 🟡 Фаза 10 — Внутриприложные обновления (self-hosted APK) — РЕАЛИЗОВАНО (ждёт деплоя)

### Проблема
GitHub может быть недоступен в РФ. Релизные APK на GitHub Releases не скачать из приложения.

### Решение
Хостить APK на нашем сервере, обновляться через приложение.

### Что сделано
- [x] **Сервер: эндпоинт `GET /api/update`** — отдаёт JSON с версией из `version.json`
- [x] **Сервер: раздача APK** — `GET /api/update/download` стримит APK файл
- [x] **CI: загрузка APK на сервер** — appleboy/scp-action + обновление `version.json` через SSH
- [x] **Клиент: `UpdateService`** — проверка версии, сравнение, скачивание с прогрессом
- [x] **Клиент: установка APK** — MethodChannel → FileProvider → `Intent.ACTION_VIEW`
- [x] **Клиент: UI диалог** — AlertDialog с changelog, прогрессом, кнопками "Update"/"Later"

### Файлы
- `server/internal/handler/update.go` — эндпоинты /api/update и /api/update/download
- `server/apk/version.json` — метаданные версии
- `server/docker-compose.yml` — mount `./apk:/app/apk:ro`
- `.github/workflows/build-android.yml` — SCP + SSH обновление version.json
- `client/lib/core/update/update_service.dart` — UpdateService (check, download, install)
- `client/lib/data/api/api_client.dart` — checkForUpdate()
- `client/lib/presentation/screens/home_screen.dart` — диалог обновления
- `client/android/app/src/main/kotlin/.../MainActivity.kt` — MethodChannel installApk
- `client/android/app/src/main/res/xml/file_paths.xml` — FileProvider paths
- `client/android/app/src/main/AndroidManifest.xml` — provider + REQUEST_INSTALL_PACKAGES

### Чтобы заработало
1. Передеплоить сервер (чтобы подхватился монтирование `apk/`)
2. Запушить тег `v*` — CI зальёт APK на сервер и обновит version.json
3. На телефоне откроется приложение → диалог с предложением обновления

---

---

## Приоритет выполнения

```
1. ✅ Фаза 0 — Безопасность
2. ✅ Фаза 0.5 — Убрать хардкод домена
3. ✅ Фаза 1 — Тесты (основные написаны)
4. ⬜ Линтеры в CI + Go и Dart тесты в CI
5. ⬜ Фаза 2 — Архитектура (клиент)
6. ⬜ Фаза 3 — Code Quality
7. ⬜ Фаза 4 — Новые фичи
```

### Быстрые победы (1-2 часа)
- `link.hashCode` → `server:port`
- `docker-compose.prod.yml` cleanup
- `mounted` check в `web_vpn_service.dart`

### Средний приоритет (2-8 часов)
- `golangci-lint` + `flutter analyze` в CI
- Разделение home_screen.dart
- Exponential backoff в fetcher
- ApiClient unit-тесты

### Большие работы (8+ часов)
- Persistence из ApiClient → StorageService
- `initialize()` рефакторинг VpnService
- Выбор сервера из списка (UI + API)
- WireGuard protocol
- REALITY поддержка в Flutter

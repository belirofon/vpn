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

## ⬜ Фаза 2 — Архитектура

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

## ⬜ Фаза 3 — Code Quality & Cleanup

### Разделение home_screen.dart
- [ ] Вынести `_ServerInfoCard` → `presentation/widgets/server_info_card.dart`
- [ ] Вынести `_DebugSheet` → `presentation/widgets/debug_sheet.dart`
- [ ] Вынести строки UI в константы/локализацию

### Остальное
- [ ] **Добавить exponential backoff** в `fetcher.go` (1s → 2s → 4s)
- [ ] **Добавить `mounted` check в `web_vpn_service.dart`**
- [ ] **Убрать мутацию `cfg.Country`** внутри `geo.FilterNonRussia()`
- [ ] **Обновить go.mod** — подтянуть актуальные версии зависимостей
- [ ] **Добавить pre-commit hooks** — `.githooks/pre-commit` с `go fmt` и `dart format`

---

## ✅ Фаза 4 — WARP & Admin Panel — ВЫПОЛНЕНО

- [x] **Сервер: Cloudflare WARP генерация** — модуль `internal/warp/warp.go` (регистрация устройства, ключи X25519, тест latency)
- [x] **Сервер: WARP admin API** — `GET/POST/DELETE /api/admin/warp` в `admin.go`
- [x] **Клиент: AdminWarpStatus DTO** — `admin_models.dart`
- [x] **Клиент: WARP методы ApiClient** — `adminGetWarp`, `adminGenerateWarp`, `adminDeleteWarp`
- [x] **Клиент: AdminViewModel WARP state** — загрузка, генерация, удаление с UI-фидбеком
- [x] **Клиент: WARP секция в админке** — карточка с деталями (endpoint, client_id, latency, protocol) + кнопки Generate/Delete
- [x] **HttpClient: добавлен delete метод** — в абстракцию и Dio имплементацию
- [x] **Документация** — README, TODO.md, TECH_DEBT.md обновлены

## ⬜ Фаза 5 — Новые фичи

- [ ] **Выбор конкретного сервера из списка** — продвинутый режим
- [ ] **Поддержка REALITY в Flutter клиенте** — uTLS/Xray core
- [ ] **История подключений** — логи соединений, статистика
- [ ] **Push-уведомления о статусе сервера** — Firebase Cloud Messaging
- [ ] **Тёмная тема** — `ThemeMode.dark`
- [ ] **Авто-подключение при запуске**
- [ ] **WireGuard protocol support**
- [ ] **Multi-user support** — per-user config cache

---

## Приоритет выполнения

```
1. ✅ Фаза 0 — Безопасность
2. ✅ Фаза 0.5 — Убрать хардкод домена
3. ✅ Фаза 1 — Тесты (основные написаны)
4. ✅ Линтеры в CI + Go и Dart тесты в CI
5. ✅ Фаза 2 — Архитектура (клиент)
6. ⬜ Фаза 3 — Code Quality
7. ✅ Фаза 4 — WARP & Admin Panel
8. ⬜ Фаза 5 — Новые фичи
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

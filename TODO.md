# План работ

> Основан на анализе проекта от 2026-06-19
> Статусы: ⬜ Pending · 🔄 In Progress · ✅ Done · ❌ Cancelled

---

## ✅ Фаза 0 — Безопасность (Critical) — ВЫПОЛНЕНО

- [x] **SCRUB: Удалить credentials из PLAN.md** — IP, порт SSH, username сервера
- [x] **CONFIG: Вынести `InsecureSkipVerify` в настройки** — добавлен флаг `SKIP_VERIFY_TLS` (default: true)
- [x] **CORS: Сделать `AllowOrigin` конфигурируемым** — переменная окружения `CORS_ORIGINS` (default: *)

---

## ✅ Фаза 0.5 — Убрать хардкод домена — ВЫПОЛНЕНО

- [x] **Flutter client** — `_defaultWebUrl` больше не содержит `belirofon-vpn.duckdns.org`, только localhost
- [x] **Caddyfile** — домен читается из `{$DOMAIN}` (env)
- [x] **docker-compose** — Caddy получает `DOMAIN`, DuckDNS парсит subdomain
- [x] **build-android.yml** — `SERVER_URL` через `vars.SERVER_URL`
- [x] **deploy.yml** — `DOMAIN` через `vars.DOMAIN`
- [x] **Makefile** — `SSH_HOST`, `SSH_PORT`, `SSH_USER`, `DOMAIN` больше не захордкожены

---

## ⬜ Фаза 1 — Тесты (High Priority)

Самый критичный пробел проекта. Без тестов нет уверенности в рефакторинге и новых фичах.

### Go unit-тесты (минимум)
- [ ] **parser** — `TEST(ParseConfigLink)` для vless, vmess, trojan, ss; `TEST(ParseSubscription)` для base64/JSON/plain
- [ ] **resolver** — `TEST(ResolveIP)` с мок-резолвером
- [ ] **geo** — `TEST(FilterNonRussia)` с мок-GeoDB
- [ ] **config** — `TEST(ParseDotEnv)` с разными форматами строк
- [ ] **tester** — `TEST(ParseUUID)` для корректности 16-байтного UUID

### Go integration
- [ ] **Добавить `golangci-lint` в CI** — перед deploy workflow
- [ ] **Добавить `go test` в CI** — `go test ./internal/...`

### Dart тесты
- [ ] **VpnConfig** — `TEST(fromJson)`, `TEST(toJson)`, `TEST(fromRawLink)`
- [ ] **ApiClient** — unit-тесты с мок-ответами (mockito/mocktail)
- [ ] **WidgetTest** — `TEST(HomeScreen)` — проверка отображения состояний
- [ ] **Добавить `flutter analyze` в CI**

---

## ⬜ Фаза 2 — Архитектура

### Сервер (Go)
- [ ] **Рефакторинг `cache.refresh()`** — выделить пайплайн:
  - `fetcher.Fetch()` → `parser.ParseAll()` → `tester.TestAll()` → `geo.Filter()` → `reality.Filter()` → `sort.ByLatency()`
  - Каждый шаг — отдельный публичный метод
- [ ] **Вынести сортировку** — общий `sort.SortByLatency()` для `refresh()` и `loadMockConfigs()`
- [ ] **Вынести REALITY filter** — отдельный пакет `internal/filter/reality.go`
- [ ] **Добавить `net.Resolver` с DoH** — опциональный DNS-over-HTTPS для резолва
- [ ] **Обновить/удалить устаревший `docker-compose.prod.yml`**

### Клиент (Flutter/Dart)
- [ ] **Решить судьбу `domain/`**:
  - Вариант A: удалить пустую директорию
  - Вариант B: имплементировать `GetBestConfigUseCase`, `ConnectToVpnUseCase`
- [ ] **Выделить persistence из ApiClient**:
  - Создать `StorageService` (обёртка над `SharedPreferences`)
  - ApiClient принимает `StorageService` или URL через конструктор
- [ ] **Добавить `initialize()` в интерфейс `VpnService`** — или сделать статический factory `VpnService.create()`
- [ ] **Заменить `hashCode.toString()` на стабильный id** — `server:port`
- [ ] **Добавить `Result<VpnConfig>` или sealed class для ответов ApiClient** — сохранить контекст ошибки

---

## ⬜ Фаза 3 — Code Quality & Cleanup

### Разделение home_screen.dart
- [ ] Вынести `_ServerInfoCard` → `presentation/widgets/server_info_card.dart`
- [ ] Вынести `_DebugSheet` → `presentation/widgets/debug_sheet.dart`
- [ ] Вынести строки UI в константы/локализацию

### Остальное
- [ ] **Исправить `parseDotEnv(\r\n)`** — добавить поддержку Windows line endings
- [ ] **Добавить exponential backoff** в `fetcher.go` (1s → 2s → 4s)
- [ ] **Добавить `mounted` check в `web_vpn_service.dart`** — sync с mobile версией
- [ ] **Убрать мутацию `cfg.Country`** внутри `geo.FilterNonRussia()` — возвращать отдельную структуру с результатом
- [ ] **Обновить go.mod** — подтянуть актуальные версии зависимостей
- [ ] **Добавить pre-commit hooks** — `.githooks/pre-commit` с `go fmt` и `dart format`

---

## ⬜ Фаза 4 — Нереализованные фичи (из PLAN.md Roadmap)

- [ ] **Выбор конкретного сервера из списка** — продвинутый режим, показать все конфиги и дать выбрать вручную
- [ ] **Поддержка REALITY в Flutter клиенте** — требует uTLS/Xray core (Go 1.24+)
- [ ] **История подключений** — логи соединений, статистика
- [ ] **Push-уведомления о статусе сервера** — через Firebase Cloud Messaging
- [ ] **Тёмная тема** — `ThemeMode.dark` по расписанию или системной настройке
- [ ] **Авто-подключение при запуске** — флаг `auto_connect` в настройках
- [ ] **WireGuard protocol support** — добавить парсинг и тестирование WireGuard конфигов

---

## Приоритет выполнения

```
1. ✅ Фаза 0 — Безопасность              (выполнено)
2. ✅ Фаза 0.5 — Убрать хардкод домена    (выполнено)
3. 🔴 Фаза 1 — Тесты                      (следующий шаг)
4. 🟡 Фаза 2 — Архитектура                (после тестов)
5. 🟡 Фаза 3 — Code Quality               (параллельно с Фазой 2)
6. 🟢 Фаза 4 — Новые фичи                 (после стабилизации)
```

### Быстрые победы (1-2 часа) — сделано ✅
- ~~SCRUB credentials из PLAN.md~~ ✅
- ~~README `/health` status fix~~ ✅
- ~~Убрать хардкод домена~~ ✅
- ~~CORS конфигурация~~ ✅
- `link.hashCode` → `server:port`
- `parseDotEnv` Windows `\r\n`
- Удалить пустую `domain/`
- `docker-compose.prod.yml` cleanup

### Средний приоритет (2-8 часов)
- Unit-тесты parser, resolver, geo
- `golangci-lint` + `flutter analyze` в CI
- Разделение home_screen.dart
- REALITY filter вынести из cache.refresh()
- Exponential backoff в fetcher

### Большие работы (8+ часов)
- Рефакторинг `cache.refresh()` в pipeline
- Persistence из ApiClient → StorageService
- `initialize()` рефакторинг VpnService
- WireGuard protocol
- Выбор сервера из списка (UI + API)

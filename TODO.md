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
- [x] **deploy.yml** — `DOMAIN` через `vars.DOMAIN`, SUBDOMAIN вычисляется
- [x] **Makefile** — `SSH_HOST`, `SSH_PORT`, `SSH_USER`, `DOMAIN` больше не захордкожены

---

## ✅ Фаза 1 — Тесты — ВЫПОЛНЕНО

### Go unit-тесты (48 тестов, 5 файлов)
- [x] **parser** (24 теста) — парсинг VLESS, VMess, Trojan, SS; подписки base64/JSON/plain; edge cases
- [x] **resolver** (4 теста) — localhost, loopback, invalid domain, empty host
- [x] **geo** (6 тестов) — nil DB, пустой список, все non-RU, invalid IP
- [x] **config** (4 теста) — parseDotEnv (normal, quoted, whitespace, malformed), LoadConfig defaults/env
- [x] **tester** (6 тестов) — ParseUUID (valid, zeros, Fs, invalid length, random), unhex

### Dart тесты (13 тестов, 2 файла)
- [x] **VpnConfig.fromJson / toJson / fromRawLink** (8 тестов)
- [x] **WidgetTest** (5 тестов) — states: disconnected, connecting error, debug menu, title

### CI
- [ ] **Добавить `go test` в CI** — перед deploy workflow
- [ ] **Добавить `flutter analyze` и `flutter test` в CI**
- [ ] **Добавить `golangci-lint` в CI**

---

## ⬜ Фаза 2 — Архитектура (Go server)

- [ ] **Рефакторинг `cache.refresh()`** — выделить пайплайн:
  - `fetcher.Fetch()` → `parser.ParseAll()` → `tester.TestAll()` → `geo.Filter()` → `reality.Filter()` → `sort.ByLatency()`
  - Каждый шаг — отдельный публичный метод
- [ ] **Вынести сортировку** — общий `sort.SortByLatency()` для `refresh()` и `loadMockConfigs()`
- [ ] **Вынести REALITY filter** — отдельный пакет `internal/filter/reality.go`
- [ ] **Добавить `net.Resolver` с DoH** — опциональный DNS-over-HTTPS для резолва
- [ ] **Обновить/удалить устаревший `docker-compose.prod.yml`**

---

## ⬜ Фаза 3 — Архитектура (Flutter client)

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

## ⬜ Фаза 4 — Code Quality & Cleanup

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

## ⬜ Фаза 5 — Админ-панель в клиенте

### Сервер (добавить эндпоинты)
- [ ] **POST /api/admin/login** — авторизация админа (email + пароль из .env, отдаёт JWT или token)
- [ ] **GET /api/admin/health** — расширенный health (статус сервера, время работы, кол-во конфигов)
- [ ] **GET /api/admin/endpoints** — список всех доступных эндпоинтов сервера
- [ ] **POST /api/admin/refresh-configs** — принудительный refresh конфигов (сейчас есть, но без auth)
- [ ] **PUT /api/admin/config** — обновить `SUBSCRIPTION_URL` и `REFRESH_INTERVAL` (сохранить в .env или в runtime)

### Авторизация
- [ ] **Добавить `ADMIN_EMAIL` и `ADMIN_PASSWORD` в `.env.example`** и в `config.go`
- [ ] **Middleware проверки токена** для /api/admin/* эндпоинтов

### Клиент — экран входа
- [ ] **Создать `presentation/screens/admin_login_screen.dart`** — форма email + пароль
- [ ] **Добавить кнопку входа на главном экране** (иконка/шестерёнка или Long-press, как DebugSheet)
- [ ] **Сохранять токен в SharedPreferences** после успешного входа

### Клиент — админ-панель
- [ ] **Создать `presentation/screens/admin_panel_screen.dart`** — главный экран админа
- [ ] **Карточка Health** — статус сервера, uptime, кол-во конфигов (GET /api/admin/health)
- [ ] **Карточка Endpoints** — список всех эндпоинтов сервера (GET /api/admin/endpoints)
- [ ] **Карточка Subscription** — просмотр и редактирование SUBSCRIPTION_URL
- [ ] **Карточка Refresh Interval** — просмотр и редактирование REFRESH_INTERVAL
- [ ] **Кнопка "Refresh Configs Now"** — POST /api/admin/refresh-configs
- [ ] **Кнопка "Logout"** — сброс токена, возврат на главную

### Разделение экранов
- [ ] **Вынести DebugSheet** из `home_screen.dart` → `presentation/widgets/debug_sheet.dart`
- [ ] **Вынести ServerInfoCard** из `home_screen.dart` → `presentation/widgets/server_info_card.dart`

---

## ⬜ Фаза 6 — Нереализованные фичи (из PLAN.md Roadmap)

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
1. ✅ Фаза 0 — Безопасность               (выполнено)
2. ✅ Фаза 0.5 — Хардкод домена            (выполнено)
3. ✅ Фаза 1 — Тесты                       (выполнено)
4. 🔴 Фаза 5 — Админ-панель               (следующий шаг)
5. 🟡 Фаза 2 — Архитектура Go             (после админки)
6. 🟡 Фаза 3 — Архитектура Flutter        (после админки)
7. 🟡 Фаза 4 — Code Quality               (параллельно)
8. 🟢 Фаза 6 — Новые фичи                 (после стабилизации)
```

### Быстрые победы (1-2 часа)
- `link.hashCode` → `server:port`
- `parseDotEnv` Windows `\r\n`
- Удалить пустую `domain/`
- `docker-compose.prod.yml` cleanup
- DebugSheet вынести из `home_screen.dart`

### Средний приоритет (2-8 часов)
- Админ-панель: серверные эндпоинты (login, health, endpoints, config)
- Админ-панель: UI клиента (логин, панель, карточки)
- `golangci-lint` + `flutter analyze` в CI
- REALITY filter вынести из cache.refresh()
- Exponential backoff в fetcher

### Большие работы (8+ часов)
- Рефакторинг `cache.refresh()` в pipeline
- Persistence из ApiClient → StorageService
- `initialize()` рефакторинг VpnService
- WireGuard protocol
- Выбор сервера из списка (UI + API)

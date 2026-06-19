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
- [x] **WidgetTest** (5 тестов) — states: disconnected, connecting, error, debug menu, title

### CI
- [x] **Добавить `go test` в CI** — добавлен в деплой workflow
- [x] **Добавить `flutter analyze` и `flutter test` в CI** — добавлены
- [ ] **Добавить `golangci-lint` в CI**

---

## ⬜ Фаза 2 — Архитектура (Go server)

- [ ] **Рефакторинг `cache.refresh()`** — выделить пайплайн:
  - `fetcher.Fetch()` → `parser.ParseAll()` → `tester.TestAll()` → `geo.Filter()` → `sort.ByLatency()`
  - Каждый шаг — отдельный публичный метод
- [ ] **Вынести сортировку** — общий `sort.SortByLatency()` для `refresh()` и `loadMockConfigs()`
- [x] **REALITY filter** — удалён из `cache.go` (фильтрация больше не нужна; REALITY конфиги тестируются и отдаются в API)
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
- [x] **Вынести `_ServerInfoCard`** → `presentation/widgets/server_info_card.dart`
- [x] **Вынести `_DebugSheet`** → `presentation/widgets/debug_sheet.dart`
- [ ] Вынести строки UI в константы/локализацию

### Остальное
- [ ] **Исправить `parseDotEnv(\r\n)`** — добавить поддержку Windows line endings
- [ ] **Добавить exponential backoff** в `fetcher.go` (1s → 2s → 4s)
- [ ] **Добавить `mounted` check в `web_vpn_service.dart`** — sync с mobile версией
- [ ] **Убрать мутацию `cfg.Country`** внутри `geo.FilterNonRussia()` — возвращать отдельную структуру с результатом
- [ ] **Обновить go.mod** — подтянуть актуальные версии зависимостей
- [ ] **Добавить pre-commit hooks** — `.githooks/pre-commit` с `go fmt` и `dart format`

---

## ✅ Фаза 5 — Админ-панель в клиенте — ВЫПОЛНЕНО

### Сервер (добавить эндпоинты)
- [x] **POST /api/admin/login** — авторизация админа (email + пароль из .env, отдаёт JWT или token)
- [x] **GET /api/admin/health** — расширенный health (статус сервера, время работы, кол-во конфигов)
- [x] **GET /api/admin/endpoints** — список всех доступных эндпоинтов сервера
- [x] **POST /api/admin/refresh-configs** — принудительный refresh конфигов (сейчас есть, но без auth)
- [x] **PUT /api/admin/config** — обновить `SUBSCRIPTION_URL` и `REFRESH_INTERVAL` (runtime)

### Авторизация
- [x] **Добавить `ADMIN_EMAIL` и `ADMIN_PASSWORD` в `.env.example`** и в `config.go`
- [x] **Middleware проверки токена** для /api/admin/* эндпоинтов

### Клиент — экран входа
- [x] **Создать `presentation/screens/admin_login_screen.dart`** — форма email + пароль
- [x] **Добавить кнопку входа на главном экране** (иконка админа в AppBar)
- [x] **Сохранять токен в SharedPreferences** после успешного входа

### Клиент — админ-панель
- [x] **Создать `presentation/screens/admin_panel_screen.dart`** — главный экран админа
- [x] **Карточка Health** — статус сервера, uptime, кол-во конфигов (GET /api/admin/health)
- [x] **Карточка Endpoints** — список всех эндпоинтов сервера (GET /api/admin/endpoints)
- [x] **Карточка Subscription** — просмотр и редактирование SUBSCRIPTION_URL
- [x] **Карточка Refresh Interval** — просмотр и редактирование REFRESH_INTERVAL
- [x] **Кнопка "Refresh Configs Now"** — POST /api/admin/refresh-configs
- [x] **Кнопка "Logout"** — сброс токена, возврат на главную

### Разделение экранов
- [x] **Вынести DebugSheet** из `home_screen.dart` → `presentation/widgets/debug_sheet.dart`
- [x] **Вынести ServerInfoCard** из `home_screen.dart` → `presentation/widgets/server_info_card.dart`

---

## ✅ Фаза 6 — Выбор сервера из списка — ВЫПОЛНЕНО

### UI
- [x] **ServerInfoCard переработан** — Column layout: флаг страны + название, пинг (цветной бейдж), имя конфига, теги протокола
- [x] **Prev/Next навигация** — кнопки `< >` для переключения между топ-10 конфигами
- [x] **Счётчик позиции** — `3 / 10` между кнопками
- [x] **Цветовая индикация пинга** — зелёный (<50ms), оранжевый (<100ms), красный (>=100ms)

### Данные
- [x] **Fetch `/api/configs`** — загрузка топ-10 при инициализации HomeScreen
- [x] **Connect использует выбранный конфиг** — вместо `getBestConfig()` используется `_configs[_currentIndex]`
- [x] **Карточка видна до подключения** — можно листать и выбирать, не нажимая CONNECT

---

## ⬜ Фаза 7 — Нереализованные фичи (из PLAN.md Roadmap)

- [x] **Сервер: REALITY фильтр удалён** — REALITY конфиги тестируются (TCP+latency) и доступны в API
- [x] **Dart модель VpnConfig** — добавлены REALITY поля (sni, fp, pbk, sid, flow)
- [x] **REALITY поддержка** — миграция на flutter_v2ray_client (Xray v26.4.17)
- [ ] **uTLS для полного протокольного теста REALITY** — требует Go 1.24+, заблокировано
- [ ] **История подключений** — логи соединений, статистика
- [ ] **Push-уведомления о статусе сервера** — через Firebase Cloud Messaging
- [ ] **Тёмная тема** — `ThemeMode.dark` по расписанию или системной настройке
- [ ] **Авто-подключение при запуске** — флаг `auto_connect` в настройках
- [ ] **WireGuard protocol support** — добавить парсинг и тестирование WireGuard конфигов

---

## ⬜ Фаза 8 — Whitelist bypass для первого запроса к серверу

### Проблема
При включённых белых списках у оператора РФ первый запрос к серверу (`belirofon-vpn.duckdns.org:8443`) может не пройти. Нужен механизм доставки конфигов через адреса, гарантированно работающие при белых списках.

### Список гарантированно работающих адресов
https://raw.githubusercontent.com/hxehex/russia-mobile-internet-whitelist/refs/heads/main/whitelist.txt

### Варианты решения
- [ ] **Вариант A: Relay на whitelist IP** — арендовать VPS с IP из whitelist. Настроить простой TCP/HTTP relay, который проксирует запросы к основному серверу.
- [ ] **Вариант B: Cloudflare как прокси** — если Cloudflare в whitelist, пропустить наш домен через Cloudflare (DNS прокси).
- [ ] **Вариант C: Multi-IP fallback в клиенте** — вшить в клиент список known-good IP/доменов. Если основной сервер недоступен, перебирать fallback.

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

## ⬜ Фаза 10 — Внутриприложные обновления (self-hosted APK)

### Проблема
GitHub может быть недоступен в РФ. Релизные APK на GitHub Releases не скачать из приложения.

### Решение
Хостить APK на нашем сервере, обновляться через приложение.

### Что нужно сделать
- [ ] **Сервер: эндпоинт `/api/update`** — отдаёт JSON с версией и ссылкой на APK
- [ ] **Сервер: раздача APK** — Caddy раздаёт статический файл `client.apk`
- [ ] **CI: загрузка APK на сервер** — после сборки заливать APK на сервер через SCP
- [ ] **Клиент: `UpdateService`** — проверка версии при запуске, скачивание APK
- [ ] **Клиент: установка APK** — запуск системного установщика (requestInstallPackages)
- [ ] **Клиент: UI уведомления** — SnackBar/диалог "Доступно обновление vX.Y.Z"

---

---

## Приоритет выполнения

```
1. ✅ Фаза 0 — Безопасность                (выполнено)
2. ✅ Фаза 0.5 — Хардкод домена            (выполнено)
3. ✅ Фаза 1 — Тесты                       (выполнено)
4. ✅ Фаза 5 — Админ-панель               (выполнено)
5. ✅ Фаза 6 — Выбор сервера из списка     (выполнено)
6. ✅ Фаза R1 — REALITY сервер             (выполнено)
7. ✅ Фаза R2 — REALITY Flutter            (миграция на flutter_v2ray_client)
8. 🟡 Фаза 8 — Whitelist bypass           (первый запрос к серверу)
9. 🟡 Фаза 9 — Cloudflare WARP            (генерация своих конфигов)
10. 🟡 Фаза 10 — Self-hosted updates       (обновления через приложение)
11. 🟡 Фаза 2 — Архитектура Go            (средний приоритет)
12. 🟡 Фаза 3 — Архитектура Flutter       (после Go)
13. 🟡 Фаза 4 — Code Quality              (параллельно)
14. 🟢 Фаза 7 — Остальные фичи            (после стабилизации)
```

### Быстрые победы (1-2 часа)
- `link.hashCode` → `server:port`
- `parseDotEnv` Windows `\r\n`
- Удалить пустую `domain/`
- `docker-compose.prod.yml` cleanup

### Средний приоритет (2-8 часов)
- Рефакторинг `cache.refresh()` в pipeline
- Exponential backoff в fetcher
- `golangci-lint` в CI
- Whitelist bypass: relay через известный IP (Фаза 8)
- WARP генератор конфигов на сервере (Фаза 9)

### Большие работы (8+ часов)
- Persistence из ApiClient → StorageService
- `initialize()` рефакторинг VpnService
- Self-hosted APK updates (Фаза 10)
- WireGuard protocol
- WARP интеграция в клиент (WireGuard подключение)
- Whitelist bypass: multi-IP fallback в клиенте

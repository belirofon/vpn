# Технический долг

> Дата: 2026-06-19
> Формат: `[Приоритет] Категория — Описание`
> ✅ = Исправлено

Приоритеты: 🔴 Critical · 🟡 High · 🟢 Low

---

## 🔴 Critical

### Безопасность

- ✅ **`PLAN.md` содержит credentials сервера** — IP (`162.248.227.46`), SSH порт (`1337`), имя пользователя (`pilot`). Удалены.
- ✅ **InsecureSkipVerify: true** — в `tester/tester.go` и `vless.go` для TLS-соединений. Вынесен в конфиг `SKIP_VERIFY_TLS`.
- ✅ **Хардкод домена в клиенте и инфраструктуре** — `belirofon-vpn.duckdns.org` был захордкожен в `api_client.dart`, `Caddyfile`, `docker-compose.yml`, `build-android.yml`, `deploy.yml`, `Makefile`. Заменён на env-переменные.

### Тесты

- ✅ **Нет unit-тестов Go** — добавлено 48 тестов (parser, resolver, geo, config, tester).
- ✅ **Нет `dart analyze` в CI** — `flutter analyze` добавлен в деплой workflow.
- **Нет `golangci-lint` в CI** — только Go unit-тесты в CI.
- ✅ **widget_test.dart пустой** — 13 Dart тестов (8 model + 5 widget).

---

## 🟡 High

### Архитектура

- **Пустая директория `client/lib/domain/`** — артефакт незавершённого Clean Architecture рефакторинга. Удалить или имплементировать UseCases/Entities.
- **ApiClient смешивает HTTP и persistence** — в `api_client.dart` HTTP-клиент (`Dio`) хранит URL, читает `SharedPreferences`, управляет `_webUrl`. Необходимо вывести persistence в отдельный сервис/репозиторий.
- **`cache.refresh()` — монолит 250+ строк** — в `cache.go` метод `refresh()` выполняет fetch → parse → test → geo → reality filter → sort. Нарушает SRP. Выделить в pipeline отдельных шагов.

### Клиент

- ✅ **`home_screen.dart` — 445 строк, 3 виджета** — `_ServerInfoCard` и `_DebugSheet` вынесены в `widgets/`.
- **Нет `initialize()` в интерфейсе `VpnService`** — `MobileVpnService` добавляет метод не реализуя абстракцию. Клиентский код вынужден делать `as MobileVpnService`.
- **`link.hashCode.toString()` как id** — в `VpnConfig.fromRawLink()`. Коллизии возможны. Нужен стабильный id на основе server:port.

### Сервер

- **REALITY filter в `refresh()`** — фильтрация конфигов с `tls="reality"` захардкожена в методе `refresh()` в `cache.go`. Должна быть отдельным шагом пайплайна.
- **`loadMockConfigs()` дублирует сортировку** — логика сортировки по `latency_ms` повторяется в `loadMockConfigs()` и `refresh()`. Вынести в общий метод.
- **`parseDotEnv` не поддерживает `\r\n`** — в `config.go`. На Windows строки с `\r\n` не будут распаршены.

### UI/UX

- **Магические строки в UI** — "Server unavailable.\nCheck that the server is running." и другие строки в `home_screen.dart`. Вынести в константы.
- **Error handling в ApiClient теряет контекст** — все методы возвращают `null/[]` при ошибке вместо `Result<T>` или кастомного исключения.
- **Нет кеширования списка конфигов** — список топ-10 загружается с сервера при каждом открытии HomeScreen. Можно добавить локальное кеширование.

---

## 🟢 Low

### Code Quality

- **geo.go: `cfg.Country = country` без синхронизации** — мутация поля конфига внутри `FilterNonRussia()`.
- **`pingServer()` слишком много логики** — в `tester.go:62-128` функция обрабатывает TLS/WS выбор и делегирует на 3 разных протокола. Разбить.
- **docker-compose.prod.yml устарел** — использует `version: "3.9"` (deprecated) и не синхронизирован с основным `docker-compose.yml`.
- **build-ios.yml нерабочий в GitHub Actions** — требует self-hosted macOS runner.
- **Makefile содержал credentials сервера** — `SSH_HOST`, `SSH_PORT`, `SSH_USER`, `DOMAIN` были захордкожены. Заменены на пустые `?=` для явной передачи.
- **Тесты используют MockApiClient + NoopHttpClientAdapter** — хрупкая связка для обхода Dio-таймеров. При обновлении тестовой инфраструктуры пересмотреть.

### Документация

- ✅ **README.md не соответствует API** — `/health` возвращает `status: "ready"`, а не `status: "ok"`. Исправлен тест, приведён в соответствие.
- ✅ **PLAN.md содержал credentials** — удалены (IP, SSH порт, username, публичный ключ).

### Инфраструктура

- ✅ **CORS: `Access-Control-Allow-Origin: *`** — теперь конфигурируется через `CORS_ORIGINS`.
- ✅ **CORS: отсутствовал `PUT`** — добавлен в `Access-Control-Allow-Methods`.
- **Нет pre-commit hooks** — `go fmt` и `dart format` не форматируются автоматически.
- **DuckDNS токен в `.env`** — хорошо что в `.gitignore`, но стоит рассмотреть секрет-менеджер.
- **retry без exponential backoff** — `fetcher.go` — всегда 1s между попытками, независимо от номера попытки.
- **Caddyfile и docker-compose** — домен теперь читается из `$DOMAIN` (env).

---

## Сводка

| Категория | 🔴 Critical | 🟡 High | 🟢 Low | ✅ Исправлено | Всего |
|-----------|-------------|---------|--------|---------------|-------|
| Безопасность | 0 | 0 | 0 | 3 | 3 |
| Архитектура | 0 | 3 | 0 | 0 | 3 |
| Тесты | 0 | 0 | 1 | 3 | 3 |
| Сервер | 0 | 3 | 1 | 0 | 4 |
| Клиент | 0 | 2 | 1 | 1 | 4 |
| UI/UX | 0 | 3 | 0 | 0 | 3 |
| Инфраструктура | 0 | 0 | 5 | 2 | 5 |
| Документация | 0 | 0 | 0 | 2 | 2 |
| **Итого** | **0** | **11** | **8** | **11** | **27** |

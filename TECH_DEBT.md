# Технический долг

> Дата: 2026-06-21
> Формат: `[Приоритет] Категория — Описание`
> ✅ = Исправлено · 🔄 = В процессе

Приоритеты: 🔴 Critical · 🟡 High · 🟢 Low

---

## 🔴 Critical

### Безопасность

- ✅ **`PLAN.md` содержит credentials сервера** — IP, SSH порт, username. Удалены.
- ✅ **InsecureSkipVerify: true** — вынесен в конфиг `SKIP_VERIFY_TLS`.
- ✅ **Хардкод домена** — заменён на env-переменные во всех компонентах.

### Тесты

- ✅ **Нет unit-тестов Go** — написаны тесты для: parser (24), config (11), geo (6), tester (6), pipeline (6), resolver. Все проходят.
- ✅ **widget_test.dart пустой** — дополнен тестами.

### Остаётся критическим

- **Нет линтера в CI** — ни `golangci-lint` для Go, ни `dart analyze` для Flutter.
- **`link.hashCode.toString()` как id** — в `VpnConfig.fromRawLink()` на клиенте. Коллизии возможны. Нужен стабильный id на основе server:port.

---

## 🟡 High

### Архитектура

- ✅ **`cache.refresh()` — монолит 250+ строк** — выделен отдельный модуль `pipeline.Pipeline` с шагами: fetch → parse → test → geo → reality → sort.
- ✅ **REALITY filter в `refresh()`** — вынесен в `pipeline.filterReality()`.
- ✅ **`loadMockConfigs()` дублирует сортировку** — перемещён в `pipeline.loadMockConfigs()`, сортировка локальная.
- ✅ **`parseDotEnv` не поддерживает `\r\n`** — добавлена нормализация `\r\n` → `\n`.
- **ApiClient смешивает HTTP и persistence** — в `api_client.dart` HTTP-клиент (`Dio`) хранит URL, читает `SharedPreferences`, управляет `_webUrl`. Необходимо вывести persistence в отдельный сервис/репозиторий.

### Клиент

- **`home_screen.dart` — 445 строк, 3 виджета** — `HomeScreen` + `_ServerInfoCard` + `_DebugSheet` в одном файле. Разнести по отдельным файлам.
- **Нет `initialize()` в интерфейсе `VpnService`** — `MobileVpnService` добавляет метод не реализуя абстракцию.
- **`link.hashCode.toString()` как id** — коллизии возможны. Нужен стабильный id на основе server:port.

### UI/UX

- **Магические строки в UI** — строки в `home_screen.dart` не вынесены в константы.
- **Error handling в ApiClient теряет контекст** — все методы возвращают `null/[]` при ошибке вместо `Result<T>`.

---

## 🟢 Low

### Code Quality

- **geo.go: `cfg.Country = country` без синхронизации** — мутация поля конфига внутри `FilterNonRussia()`.
- **`pingServer()` слишком много логики** — в `tester.go` функция обрабатывает TLS/WS выбор и делегирует на 3 разных протокола.
- **docker-compose.prod.yml устарел** — использует `version: "3.9"` (deprecated), не синхронизирован с `docker-compose.yml`.
- **build-ios.yml нерабочий в GitHub Actions** — требует self-hosted macOS runner.

### Инфраструктура

- **Нет pre-commit hooks** — `go fmt` и `dart format` не форматируются автоматически.
- **DuckDNS токен в `.env`** — хорошо что в `.gitignore`, но стоит рассмотреть секрет-менеджер.
- **retry без exponential backoff** — `fetcher.go` — всегда 1s между попытками.

---

## Сводка

| Категория | 🔴 Critical | 🟡 High | 🟢 Low | ✅ Исправлено | 🔄 В процессе | Всего |
|-----------|-------------|---------|--------|---------------|---------------|-------|
| Безопасность | 0 | 0 | 0 | 3 | 0 | 3 |
| Архитектура | 0 | 1 | 0 | 4 | 0 | 5 |
| Тесты | 2 | 0 | 0 | 1 | 0 | 3 |
| Сервер | 0 | 0 | 2 | 3 | 0 | 5 |
| Клиент | 1 | 3 | 0 | 0 | 0 | 4 |
| UI/UX | 0 | 2 | 0 | 0 | 0 | 2 |
| Инфраструктура | 0 | 0 | 3 | 0 | 0 | 3 |
| Документация | 0 | 0 | 0 | 2 | 0 | 2 |
| **Итого** | **3** | **6** | **5** | **13** | **0** | **27** |

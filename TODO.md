# TODO - PR Reviewer Assignment Service

## Требования

- **База данных**: PostgreSQL
- **Объём данных**: до 20 команд, до 200 пользователей
- **Производительность**: RPS — 5, SLI времени ответа — 300 мс, SLI успешности — 99.9%
- **Правила**: пользователь с `isActive = false` не назначается на ревью
- **Идемпотентность**: операция merge должна быть идемпотентной
- **Развёртывание**: docker-compose up, миграции применяются автоматически
- **Порт**: 8080

## 1. Структура проекта

- [ ] Создать `internal/domain/` - доменные модели (User, Team, PullRequest)
- [ ] Создать `internal/storage/` - репозиторий для работы с PostgreSQL
- [ ] Создать `internal/service/` - бизнес-логика (назначение ревьюеров, переназначение)
- [ ] Создать `internal/handler/` - HTTP handlers, реализующие ServerInterface
- [ ] Создать `internal/config/` - конфигурация приложения (БД, порт)
- [ ] Создать `migrations/` - SQL миграции для PostgreSQL

## 2. Доменные модели (`internal/domain/`)

- [ ] Модель `User` (user_id, username, team_name, is_active)
- [ ] Модель `Team` (team_name, members)
- [ ] Модель `PullRequest` (id, name, author_id, status, reviewers, timestamps)

## 3. База данных и миграции

- [ ] Создать схему БД: таблицы `users`, `teams`, `pull_requests`, `pr_reviewers`
- [ ] Миграция: создание таблиц с индексами
- [ ] Индексы для оптимизации запросов (user_id, team_name, pr_id, reviewer_id)
- [ ] Уникальные ограничения (team_name, pr_id)
- [ ] Внешние ключи для целостности данных
- [ ] Настроить автоматическое применение миграций при старте

## 4. Хранилище (`internal/storage/`)

- [ ] Подключение к PostgreSQL (использовать `database/sql` или `pgx`)
- [ ] Репозиторий `TeamRepository`: CreateTeam, GetTeam, TeamExists
- [ ] Репозиторий `UserRepository`: CreateOrUpdateUser, GetUser, UpdateUserIsActive, GetActiveUsersByTeam
- [ ] Репозиторий `PRRepository`: CreatePR, GetPR, UpdatePRStatus, GetPRsByReviewer
- [ ] Репозиторий `ReviewerRepository`: AssignReviewers, ReassignReviewer, GetReviewersByPR
- [ ] Транзакции для атомарных операций
- [ ] Обработка ошибок БД (duplicate key, not found)

## 5. Сервисный слой (`internal/service/`)

- [ ] `TeamService` - управление командами и пользователями
  - [ ] CreateOrUpdateTeam - создание/обновление команды с валидацией уникальности
  - [ ] GetTeam - получение команды с участниками
- [ ] `UserService` - управление пользователями
  - [ ] SetUserIsActive - изменение флага активности
- [ ] `PRService` - создание PR, назначение ревьюеров, переназначение
  - [ ] CreatePR - создание PR с автоназначением ревьюеров (до 2 активных из команды автора, исключая автора)
  - [ ] MergePR - идемпотентная операция пометки PR как MERGED
  - [ ] ReassignReviewer - переназначение ревьюера (только для OPEN PR)
  - [ ] GetPRsByReviewer - получение списка PR для ревьювера
- [ ] Логика выбора ревьюеров: случайный выбор из активных участников команды автора (исключая автора, is_active = true)
- [ ] Логика переназначения: выбор из активных участников команды заменяемого ревьювера (is_active = true)
- [ ] Валидация: нельзя менять ревьюеров у MERGED PR
- [ ] Обработка edge cases: если доступных кандидатов меньше двух, назначается доступное количество (0/1)

## 6. HTTP Handlers (`internal/handler/`)

- [ ] Реализовать все методы `ServerInterface`:
  - [ ] `PostTeamAdd` - создание/обновление команды (201/400 TEAM_EXISTS)
  - [ ] `GetTeamGet` - получение команды (200/404 NOT_FOUND)
  - [ ] `PostUsersSetIsActive` - изменение активности пользователя (200/404 NOT_FOUND)
  - [ ] `PostPullRequestCreate` - создание PR с автоназначением ревьюеров (201/404/409 PR_EXISTS)
  - [ ] `PostPullRequestMerge` - пометка PR как MERGED, идемпотентная (200/404)
  - [ ] `PostPullRequestReassign` - переназначение ревьюера (200/404/409 PR_MERGED/NOT_ASSIGNED/NO_CANDIDATE)
  - [ ] `GetUsersGetReview` - список PR пользователя-ревьювера (200/404)
- [ ] Обработка ошибок с правильными HTTP статусами и кодами из OpenAPI
- [ ] Парсинг JSON запросов и формирование JSON ответов
- [ ] Валидация входных данных (проверка существования пользователей, команд)

## 7. Точка входа (`cmd/server/main.go`)

- [ ] Загрузка конфигурации (переменные окружения или файл)
- [ ] Подключение к PostgreSQL с retry логикой
- [ ] Применение миграций при старте (автоматически)
- [ ] Инициализация репозиториев
- [ ] Инициализация сервисов
- [ ] Инициализация handlers
- [ ] Настройка HTTP сервера с chi router
- [ ] Запуск сервера на порту 8080
- [ ] Graceful shutdown

## 8. Docker и развёртывание

- [ ] Создать `Dockerfile` для Go приложения
- [ ] Создать `docker-compose.yml` с сервисами:
  - [ ] `app` - основной сервис (порт 8080)
  - [ ] `postgres` - база данных PostgreSQL
  - [ ] Настроить переменные окружения для подключения к БД
  - [ ] Настроить healthcheck для сервисов
- [ ] Настроить автоматическое применение миграций при старте (через init container или в коде)
- [ ] `.dockerignore` для исключения ненужных файлов
- [ ] Проверить, что `docker-compose up` поднимает всё корректно

## 9. Конфигурация

- [ ] Переменные окружения:
  - [ ] `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
  - [ ] `SERVER_PORT` (по умолчанию 8080)
- [ ] Структура конфигурации в `internal/config/`

## 10. Утилиты и вспомогательные функции

- [ ] Утилита для случайного выбора из списка (для назначения ревьюеров)
- [ ] Функции для работы с временными метками (createdAt, mergedAt)
- [ ] Валидация входных данных
- [ ] Обработка edge cases (нет доступных кандидатов, пустые команды)

## Структура проекта

```
pr-review-assigner/
├── cmd/server/main.go
├── internal/
│   ├── api/api.gen.go 
│   ├── domain/
│   │   ├── user.go
│   │   ├── team.go
│   │   └── pullrequest.go
│   ├── config/
│   │   └── config.go
│   ├── storage/
│   │   ├── repository.go
│   │   ├── team_repository.go
│   │   ├── user_repository.go
│   │   └── pr_repository.go
│   ├── service/
│   │   ├── team_service.go
│   │   ├── user_service.go
│   │   └── pr_service.go
│   └── handler/
│       └── server.go
├── migrations/
│   └── 001_init.sql
├── Dockerfile
├── docker-compose.yml
├── .dockerignore
├── openapi.yml
└── go.mod
```

## Зависимости для добавления в go.mod

- [ ] `github.com/lib/pq` или `github.com/jackc` - драйвер PostgreSQL
- [ ] `github.com/golang-migrate/migrate` - для миграций БД 


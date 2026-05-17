# Практическое занятие №7 — Dockerfile и сборка контейнеров

**Студент:** Выборнов Олег Андреевич
**Группа:** ЭФМО-02-25
**Дисциплина:** Технологии программирования

## Цель работы

Упаковать сервисы в Docker-образы через multi-stage сборку для минимизации размера и обеспечения воспроизводимости. Запускать связанные сервисы через docker compose с общей сетью.

## Архитектура

```
                  HTTPS (8443)
   curl/браузер ─────────────► nginx ──► auth   (8081)
                                    │
                                    └──► tasks  (8082)
                                              │
                                              ▼
                                        PostgreSQL (5432)
```

Все четыре контейнера общаются внутри сети `app-network` по DNS-именам (`auth`, `tasks`, `postgres`, `nginx`). Наружу проброшен только порт 8443 nginx.

## Структура проекта

```
pz7/
├── deploy/
│   ├── Dockerfile.auth          # multi-stage: golang:1.25-alpine → alpine:3.20
│   ├── Dockerfile.tasks         # multi-stage: golang:1.25-alpine → alpine:3.20
│   ├── docker-compose.yml       # 4 сервиса, общая сеть, healthcheck postgres
│   ├── nginx.conf               # TLS-терминация, проксирование auth+tasks
│   └── tls/                     # самоподписанный сертификат (в .gitignore)
├── migrations/
│   └── 01_create_tasks_table.sql
├── services/
│   ├── auth/                    # HTTP-сервис авторизации с cookies
│   └── tasks/                   # CRUD задач + CSRF/XSS защита
├── shared/
│   ├── middleware/              # RequestID, Logging, CSRF, SecurityHeaders
│   └── httpx/
├── images/                      # скриншоты проверок
├── .dockerignore                # исключает мусор из контекста сборки
├── go.mod
└── README.md
```

## Multi-stage Dockerfile

### deploy/Dockerfile.tasks

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/tasks ./services/tasks/cmd/tasks

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/tasks /app/tasks
EXPOSE 8082
ENTRYPOINT ["/app/tasks"]
```

**Что и зачем:**

- `golang:1.25-alpine AS builder` — полный Go-toolchain нужен только для компиляции.
- `CGO_ENABLED=0` — статическая сборка, бинарь не зависит от системных библиотек, работает в любом alpine.
- Второй `FROM alpine:3.20` — стартует «с чистого листа», из первой стадии копируется ТОЛЬКО готовый бинарь.
- `ca-certificates` — нужны, чтобы из контейнера можно было ходить по HTTPS наружу.
- Финальный образ не содержит исходного кода, Go-компилятора, кеша модулей.

`Dockerfile.auth` устроен аналогично, отличается только путём `./services/auth/cmd/auth`.

## .dockerignore

В корне проекта:

```
.git
.gitignore
README.md
*.md
.vscode/
.idea/
images/
*.log
*.tmp
cookies.txt
login.json
body.json
xss.json
deploy/tls/key.pem
deploy/tls/cert.pem
```

`.dockerignore` исключает файлы из контекста сборки — то, что Docker передаёт демону при `docker build`. Это:
- ускоряет сборку (меньше данных передаётся),
- уменьшает размер слоёв (если бы что-то из этого попало в COPY),
- защищает от случайного попадания приватного ключа в образ.

## docker-compose.yml — взаимодействие сервисов

```yaml
services:
  postgres:
    image: postgres:15-alpine
    container_name: pz7-postgres
    environment:
      POSTGRES_USER:     tasks_user
      POSTGRES_PASSWORD: tasks_pass
      POSTGRES_DB:       tasks_db
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U tasks_user -d tasks_db"]
      interval: 3s
      retries: 10
    networks: [app-network]

  auth:
    build:
      context: ..
      dockerfile: deploy/Dockerfile.auth
    container_name: pz7-auth
    environment:
      AUTH_PORT: "8081"
    networks: [app-network]

  tasks:
    build:
      context: ..
      dockerfile: deploy/Dockerfile.tasks
    container_name: pz7-tasks
    environment:
      TASKS_PORT:    "8082"
      AUTH_BASE_URL: "http://auth:8081"
      DB_HOST:       postgres
      DB_USER:       tasks_user
      DB_PASSWORD:   tasks_pass
      DB_NAME:       tasks_db
    depends_on:
      postgres:
        condition: service_healthy
      auth:
        condition: service_started
    networks: [app-network]

  nginx:
    image: nginx:1.27-alpine
    container_name: pz7-nginx
    ports:
      - "8443:8443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./tls:/etc/nginx/tls:ro
    networks: [app-network]
```

**Ключевые моменты:**

- `AUTH_BASE_URL: "http://auth:8081"` — tasks обращается к auth по имени сервиса, не по `localhost`. Docker-сеть резолвит имя в IP контейнера.
- `depends_on: condition: service_healthy` — tasks стартует только когда postgres реально готов принимать соединения, а не просто запустился.
- `:ro` на volume — конфиг nginx и сертификаты примонтированы только на чтение.

## Переменные окружения

| Сервис | Переменная | Значение | Описание |
|---|---|---|---|
| auth | `AUTH_PORT` | 8081 | HTTP-порт сервиса |
| tasks | `TASKS_PORT` | 8082 | HTTP-порт сервиса |
| tasks | `AUTH_BASE_URL` | http://auth:8081 | Адрес auth внутри docker-сети |
| tasks | `DB_HOST` | postgres | Хост БД (имя сервиса в compose) |
| tasks | `DB_PORT` | 5432 | Порт БД |
| tasks | `DB_USER` | tasks_user | Пользователь БД |
| tasks | `DB_PASSWORD` | tasks_pass | Пароль БД |
| tasks | `DB_NAME` | tasks_db | Имя БД |

Конфигурация передаётся через окружение, секретов в Dockerfile нет — образ можно безопасно публиковать.

## Команды сборки и запуска

### Сборка вручную

```powershell
# из корня проекта
docker build -t pz7-auth:latest  -f deploy/Dockerfile.auth  .
docker build -t pz7-tasks:latest -f deploy/Dockerfile.tasks .
```

### Запуск всего стека

```powershell
cd deploy
docker compose up -d --build
docker compose ps
```

### Просмотр логов

```powershell
docker compose logs -f tasks
docker compose logs -f auth
```

### Остановка

```powershell
docker compose down

# с удалением volume (БД будет пересоздана)
docker compose down -v
```

## Проверки

### 1. Статус контейнеров

```powershell
docker compose ps
```

Все четыре сервиса в `Up`, postgres имеет статус `healthy`.

![Статус контейнеров](images/01_containers.png)

### 2. Размеры образов

```powershell
docker images | Select-String "deploy-tasks|deploy-auth|nginx.*1.27-alpine|postgres.*15-alpine"
```

| Образ | Размер | Комментарий |
|---|---|---|
| `deploy-auth:latest` | 27.1 MB | alpine + статический Go-бинарь |
| `deploy-tasks:latest` | 30 MB | alpine + бинарь + ca-certificates |
| `nginx:1.27-alpine` | 74.5 MB | официальный образ nginx |
| `postgres:15-alpine` | 392 MB | официальный образ БД |

Размер наших образов (27-30 MB) — результат multi-stage сборки. Без неё образ на основе `golang:1.25-alpine` весил бы около 350 MB, потому что тащил бы с собой весь Go-toolchain.

![Размеры образов](images/02_images.png)

### 3. Внутренняя сеть Docker

```powershell
docker exec pz7-tasks wget -qO- --header="Authorization: Bearer demo-token" http://auth:8081/v1/auth/verify
```

Из контейнера `tasks` обращаемся к контейнеру `auth` по DNS-имени `auth` — Docker-сеть резолвит его в IP внутреннего адреса.

Ответ: `{"valid":true,"subject":"student"}`

![Внутренняя сеть Docker](images/03_internal_network.png)

### 4. End-to-end: HTTPS логин

```powershell
Set-Content -Path login.json -Value '{"username":"student","password":"student"}'

curl.exe -k -i -c cookies.txt -X POST https://localhost:8443/v1/auth/login `
  -H "Content-Type: application/json" `
  --data-binary "@login.json"
```

`200 OK`, две cookies (session+csrf), security headers. Запрос прошёл через nginx → auth.

![Логин через HTTPS](images/04_login.png)

### 5. Создание задачи через весь стек

```powershell
$csrf = (Get-Content cookies.txt | Select-String "csrf_token").ToString().Split("`t")[-1]
Set-Content -Path body.json -Value '{"title":"docker test","description":"from pz7"}'

curl.exe -k -i -b cookies.txt -X POST https://localhost:8443/v1/tasks `
  -H "Content-Type: application/json" `
  -H "X-CSRF-Token: $csrf" `
  --data-binary "@body.json"
```

`201 Created`. Запрос прошёл: nginx → tasks → проверка auth (по docker-сети) → запись в postgres → ответ обратно.

![Создание задачи через стек](images/05_create.png)

### 6. Логи сервиса

```powershell
docker compose logs tasks --tail 20
```

Сервис пишет в stdout — это правильно для контейнеров: Docker сам собирает логи через драйвер. Никаких файлов в /var/log писать не нужно.

![Логи tasks](images/06_logs.png)

## Контрольные вопросы

**1. Зачем multi-stage сборка?**

Чтобы в финальном образе не оставался Go-компилятор, исходный код и кеш модулей. В первой стадии (builder) собирается бинарь, во второй — копируется только он. Образ становится в 10+ раз меньше, в нём нечего ломать, поверхность атаки минимальна.

**2. Почему два сервиса в одной docker-сети могут обращаться друг к другу по имени?**

Docker запускает встроенный DNS-сервер (`127.0.0.11`) внутри каждой пользовательской сети. Имена сервисов из `docker-compose.yml` регистрируются как DNS-записи. Когда tasks делает запрос на `http://auth:8081`, ОС в контейнере резолвит `auth` через этот DNS и получает IP контейнера auth.

**3. Зачем `.dockerignore`?**

При `docker build` весь контекст (каталог сборки) отправляется демону Docker, даже файлы, которые потом не попадут в образ. `.dockerignore` исключает ненужные файлы из этой передачи: ускоряет сборку, защищает от случайной утечки приватных файлов (ключи, пароли) через COPY, уменьшает размер слоёв если бы кто-то скопировал .git или images.

**4. Чем `condition: service_healthy` отличается от обычного `depends_on`?**

Обычный `depends_on` ждёт только запуска контейнера — но запуск ≠ готовность. PostgreSQL может стартовать за 2 секунды, но принимать соединения только через 8. Без `service_healthy` сервис tasks падал бы при попытке подключиться к БД на старте. С healthcheck Docker реально проверяет `pg_isready` каждые 3 секунды и поднимает зависимые сервисы только после `healthy`.

**5. Почему секреты передаются через переменные окружения, а не зашиваются в Dockerfile?**

Если зашить пароль БД в Dockerfile, он попадёт в каждый слой образа — извлекается через `docker history` или просмотр слоёв. Образ становится непереносимым: один и тот же бинарь не получится использовать с разными БД. Через окружение секрет инжектится в момент запуска и в образе не оседает. Для production используют secrets manager или Docker Swarm secrets, в учебной работе — переменные в compose.

**6. Почему сервис должен писать логи в stdout, а не в файл?**

Docker собирает stdout контейнера через драйвер логирования (json-file по умолчанию, можно настроить syslog, fluentd, journald). Если писать в файл внутри контейнера — он останется внутри, при перезапуске потеряется (если volume не настроен), сложнее агрегировать с других инстансов. stdout — стандарт для контейнерных приложений (12-factor app).
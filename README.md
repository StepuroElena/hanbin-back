# hanbin-back

Go-бэкенд для [Hanbin Drama Tracker](../hanbin-front).

---

## Содержание

- [Структура проекта](#структура-проекта)
- [Требования](#требования)
- [Первый запуск](#первый-запуск)
- [Запуск после установки](#запуск-после-установки)
- [API для фронтенда](#api-для-фронтенда)
- [Коды ошибок](#коды-ошибок)
- [Примеры запросов из JS](#примеры-запросов-из-js)

---

## Структура проекта

```
hanbin-back/
├── cmd/api/
│   └── main.go                              # точка входа, сборка зависимостей
├── internal/
│   ├── domain/user/
│   │   ├── profile.entity.go                # модель Profile (бизнес-правила)
│   │   ├── profile.entity_test.go           # тесты модели
│   │   └── profile.repository.go            # интерфейс репозитория
│   ├── repository/user/
│   │   └── profile.postgres.go              # работа с PostgreSQL
│   ├── service/user/
│   │   └── profile.service.go               # use-case'ы + DTO
│   ├── handler/user/
│   │   └── profile.handler.go               # HTTP-хэндлеры
│   └── middleware/
│       └── cors.go                          # CORS для фронтенда
├── migrations/
│   ├── 001_create_profiles.up.sql           # создать таблицу
│   └── 001_create_profiles.down.sql         # удалить таблицу
├── .env                                     # переменные окружения
├── Makefile                                 # команды для запуска
└── go.mod
```

---

## Требования

- [Go](https://go.dev/dl/) 1.22+
- [PostgreSQL](https://formulae.brew.sh/formula/postgresql@16) 16+
- [Homebrew](https://brew.sh/) (для Mac)

Установка если ещё не установлено:
```bash
brew install go
brew install postgresql@16
brew services start postgresql@16
```

---

## Первый запуск

Выполни один раз при первоначальной настройке:

```bash
# 1. Создать базу данных
createdb hanbin

# 2. Подтянуть зависимости Go
go mod tidy

# 3. Применить миграции (создать таблицы)
make migrate-up
```

---

## Запуск после установки

```bash
make run
```

Сервер запустится на `http://localhost:8080`.

Чтобы остановить — нажми `Ctrl + C` в терминале.

---

## API для фронтенда

Базовый URL: `http://localhost:8080`

Все запросы и ответы в формате `application/json`.

---

### Получить профиль пользователя

```
GET /api/v1/profiles/{id}
```

**Пример запроса:**
```
GET /api/v1/profiles/1
```

**Ответ `200 OK`:**
```json
{
  "id": 1,
  "name": "Hanbin",
  "email": "hanbin@example.com",
  "created_at": "2026-03-07T12:51:50Z",
  "updated_at": "2026-03-07T12:51:50Z"
}
```

---

### Создать профиль пользователя

```
POST /api/v1/profiles
```

**Тело запроса:**
```json
{
  "name": "Hanbin",
  "email": "hanbin@example.com"
}
```

**Ответ `201 Created`:**
```json
{
  "id": 1,
  "name": "Hanbin",
  "email": "hanbin@example.com",
  "created_at": "2026-03-07T12:51:50Z",
  "updated_at": "2026-03-07T12:51:50Z"
}
```

---

### Обновить профиль пользователя

```
PATCH /api/v1/profiles/{id}
```

Оба поля опциональны — передавай только то, что нужно изменить.

**Тело запроса:**
```json
{
  "name": "Новое имя"
}
```

**Ответ `200 OK`** — обновлённый профиль в том же формате.

---

### Удалить профиль пользователя

```
DELETE /api/v1/profiles/{id}
```

**Ответ `204 No Content`** — тело пустое.

---

## Коды ошибок

Все ошибки возвращаются в формате:
```json
{
  "error": "описание ошибки"
}
```

| Код | Когда возникает |
|-----|----------------|
| `400 Bad Request` | Невалидный ID, пустое имя/email, неверный формат email, превышен лимит 255 символов |
| `404 Not Found` | Профиль с таким ID не существует |
| `409 Conflict` | Email уже занят другим пользователем |
| `500 Internal Server Error` | Внутренняя ошибка сервера |

---

## Примеры запросов из JS

Готовые функции для использования на фронте:

```javascript
const API_URL = 'http://localhost:8080/api/v1';

// Получить профиль по ID
async function getProfile(id) {
  const res = await fetch(`${API_URL}/profiles/${id}`);
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error);
  }
  return res.json();
}

// Создать профиль
async function createProfile(name, email) {
  const res = await fetch(`${API_URL}/profiles`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, email }),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error);
  }
  return res.json();
}

// Обновить профиль
async function updateProfile(id, fields) {
  // fields = { name: '...' } или { email: '...' } или оба
  const res = await fetch(`${API_URL}/profiles/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(fields),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error);
  }
  return res.json();
}

// Удалить профиль
async function deleteProfile(id) {
  const res = await fetch(`${API_URL}/profiles/${id}`, {
    method: 'DELETE',
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error);
  }
}

// Пример использования:
const profile = await getProfile(1);
console.log(profile.name); // "Hanbin"
```

---

## Разрешённые origins для CORS

По умолчанию разрешены запросы с:
- `http://localhost:3000`
- `http://localhost:5500`
- `http://127.0.0.1:5500`

Если фронт открывается на другом порту — добавь его в `.env`:
```
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5500,http://127.0.0.1:5500,http://localhost:ТВОЙ_ПОРТ
```

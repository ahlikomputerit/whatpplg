.PHONY: build up down logs dev dev-down clean restart

# ---- Production ----

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

restart:
	docker compose restart

# ---- Development (hot reload) ----

dev:
	docker compose -f docker-compose.dev.yml up

dev-down:
	docker compose -f docker-compose.dev.yml down

dev-logs:
	docker compose -f docker-compose.dev.yml logs -f

# ---- With Redis ----

up-redis:
	docker compose --profile with-redis up -d

# ---- With PostgreSQL ----

up-pg:
	docker compose --profile with-postgres up -d

# ---- Clean ----

clean:
	docker compose down -v --remove-orphans

clean-all: clean
	docker system prune -f

# ---- Build & Push Image ----

image:
	docker build -t wa-gateway:latest .

# ---- Utility ----

exec:
	docker compose exec gateway sh

ps:
	docker compose ps

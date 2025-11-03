# Makefile

GRAFANA_PORT := 3000
PROMETHEUS_PORT := 9090
FRONTEND_FILE := frontend/index.html

DOCKER_COMPOSE := docker compose
OPEN_BROWSER := xdg-open

.PHONY: all up down open

all: up open

up:
	$(DOCKER_COMPOSE) up -d

down:
	$(DOCKER_COMPOSE) down

open:
	$(OPEN_BROWSER) $(FRONTEND_FILE)
	$(OPEN_BROWSER) http://localhost:$(GRAFANA_PORT)
	$(OPEN_BROWSER) http://localhost:$(PROMETHEUS_PORT)

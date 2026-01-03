SHELL=/bin/bash
SERVICE=modem-stats

.PHONY: up
up: docker
	/n/config/compose up -d ${SERVICE}

.PHONY: docker
docker:
	docker build . -f Dockerfile -t rcbilson/${SERVICE}

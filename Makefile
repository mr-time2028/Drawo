# PRODUCTION
up:
	docker-compose up
up_build:
	docker-compose up --build
down:
	docker-compose down

# DEVELOPMENT
dev_up:
	docker-compose -f docker-compose-dev.yml up
dev_up_build:
	docker-compose -f docker-compose-dev.yml up --build
dev_down:
	docker-compose -f docker-compose-dev.yml down

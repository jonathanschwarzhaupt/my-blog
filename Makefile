include .envrc

.PHONY: help
help: ## print this help message
	@echo 'Usage:'
	@echo
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_\/-]+:.*##/ {printf "  %-30s %s\n", $$1, substr($$0, index($$0, "##")+3)}' $(MAKEFILE_LIST) | sort

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

run/blog: addr ?= :8080
run/blog-admin: addr ?= :4001

.PHONY: run/blog
run/blog: css/build ## run the blog binary (override port: make run/blog addr=:8081)
	go run ./cmd/blog -db-dsn=${BLOG_DB_DSN} -addr=${addr}

.PHONY: run/blog-admin
run/blog-admin: css/build ## run the blog-admin binary (override port: make run/blog-admin addr=:4002)
	go run ./cmd/blog-admin -db-dsn=${BLOG_DB_DSN} -addr=${addr}

.PHONY: css/before_build
css/before_build:
	@cp ui/static/css/main.css /tmp/main.css.before-audit 2>/dev/null || true

.PHONY: css/build
css/build: ## compile Tailwind CSS to ui/static/css/main.css
	mise exec -- tailwindcss -i ./ui/css/input.css -o ./ui/static/css/main.css --minify

.PHONY: css/watch
css/watch: ## rebuild Tailwind CSS on file changes, for local development
	mise exec -- tailwindcss -i ./ui/css/input.css -o ./ui/static/css/main.css --watch

.PHONY: db/migrations/up
db/migrations/up: confirm ## apply all up migrations
	go run ./cmd/migrate -db-dsn=${BLOG_DB_DSN} up

.PHONY: db/migrations/down
db/migrations/down: confirm ## revert the most recently applied migration (goose down reverts one step, not all)
	go run ./cmd/migrate -db-dsn=${BLOG_DB_DSN} down

.PHONY: db/migrations/status
db/migrations/status: ## show migration status
	go run ./cmd/migrate -db-dsn=${BLOG_DB_DSN} status

.PHONY: sqlc/generate
sqlc/generate: ## regenerate internal/database from sql/queries + sql/schema
	go tool sqlc generate

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

.PHONY: audit
audit: css/before_build css/build ## run quality control checks
	@diff -q ui/static/css/main.css /tmp/main.css.before-audit >/dev/null 2>&1 && rm -f /tmp/main.css.before-audit || (rm -f /tmp/main.css.before-audit && echo "ui/static/css/main.css was stale and has been rebuilt — review and include it in your commit" && exit 1)
	go mod tidy -diff
	go mod verify
	go vet ./...
	go tool staticcheck ./...
	go test -race -vet=off ./...

.PHONY: tidy
tidy: ## tidy modfiles and format .go files
	go mod tidy
	go fix ./...
	go fmt ./...

# ==================================================================================== #
# BUILD
# ==================================================================================== #

# version comes from runtime/debug.ReadBuildInfo (internal/vcs.Version()), not ldflags -X
linker_flags = '-s'

.PHONY: build/blog
build/blog: css/build ## build the blog binary
	go build -ldflags=${linker_flags} -o=./bin/blog ./cmd/blog

.PHONY: build/blog-admin
build/blog-admin: css/build ## build the blog-admin binary
	go build -ldflags=${linker_flags} -o=./bin/blog-admin ./cmd/blog-admin

.PHONY: build/migrate
build/migrate: ## build the migrate binary (used as the Kubernetes init container image)
	go build -ldflags=${linker_flags} -o=./bin/migrate ./cmd/migrate

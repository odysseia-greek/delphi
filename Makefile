.PHONY: all

.PHONY: tidy tidy-go
tidy: tidy-go

tidy-go:
	@echo "==> Running 'go mod tidy' in all modules..."
	@mods=$$(find . -name go.mod -print0 | xargs -0 -n1 dirname | sort -u); \
	if [[ -z "$$mods" ]]; then \
		echo "No go.mod files found."; \
		exit 0; \
	fi; \
	for d in $$mods; do \
		echo "==> go mod tidy in $$d"; \
		( cd "$$d" && go mod tidy && go fmt ./... ); \
	done

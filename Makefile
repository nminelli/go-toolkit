.PHONY: help release

# Default target
help:
	@echo "Usage:"
	@echo "  make release LIB=<lib_name> VERSION=<version>"
	@echo ""
	@echo "Parameters:"
	@echo "  LIB      - Library name (webapp, httprouter)"
	@echo "  VERSION  - Semantic version (e.g., 1.0.0)"
	@echo ""
	@echo "Example:"
	@echo "  make release LIB=webapp VERSION=1.3.0"

# Release a library with a specific version
release:
	@# Validate LIB parameter is provided
	@if [ -z "$(LIB)" ]; then \
		echo "Error: LIB parameter is required"; \
		echo "Usage: make release LIB=<lib_name> VERSION=<version>"; \
		exit 1; \
	fi

	@# Validate VERSION parameter is provided
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION parameter is required"; \
		echo "Usage: make release LIB=<lib_name> VERSION=<version>"; \
		exit 1; \
	fi

	@# Validate semver format (X.Y.Z)
	@if ! echo "$(VERSION)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "Error: VERSION must be in semver format (X.Y.Z)"; \
		echo "Example: 1.0.0, 2.1.3"; \
		exit 1; \
	fi

	@# Validate library exists
	@if [ ! -f "$(LIB)/version.go" ]; then \
		echo "Error: Library '$(LIB)' does not exist or has no version.go"; \
		echo "Available libraries:"; \
		ls -d */version.go 2>/dev/null | cut -d'/' -f1 || echo "  (none found)"; \
		exit 1; \
	fi

	@# Validate we're on main branch
	@CURRENT_BRANCH=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$CURRENT_BRANCH" != "main" ]; then \
		echo "Error: Must be on 'main' branch to release"; \
		echo "Current branch: $$CURRENT_BRANCH"; \
		exit 1; \
	fi

	@# Update version.go
	@echo "Updating $(LIB)/version.go to version $(VERSION)..."
	@sed -i 's/var version = "[^"]*"/var version = "$(VERSION)"/' $(LIB)/version.go

	@# Commit the change
	@echo "Committing version bump..."
	@git add $(LIB)/version.go
	@git commit -m "chore($(LIB)): bump version to $(VERSION)"

	@# Push the commit
	@echo "Pushing to origin..."
	@git push origin main

	@# Create and push tag
	@TAG_NAME="$(LIB)/v$(VERSION)"; \
	echo "Creating tag $$TAG_NAME..."; \
	git tag -a "$$TAG_NAME" -m "Release $(LIB) v$(VERSION)"; \
	git push origin "$$TAG_NAME"

	@# Create GitHub release
	@TAG_NAME="$(LIB)/v$(VERSION)"; \
	echo "Creating GitHub release..."; \
	gh release create "$$TAG_NAME" --title "$(TAG_NAME)" --notes "Release $(LIB) version $(VERSION)"

	@echo ""
	@echo "✅ Successfully released $(LIB) v$(VERSION)"

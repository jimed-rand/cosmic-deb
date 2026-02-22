BINARY     := cosmic-deb
GO         := go
GOFLAGS    := -trimpath
LDFLAGS    := -s -w
OUTDIR     := cosmic-packages
WORKDIR    := cosmic-work
REPOS      := built-in
TAG        :=
JOBS       := $(shell nproc)
DESTDIR    :=
PREFIX     := /usr/local
BINDIR     := $(DESTDIR)$(PREFIX)/bin
SCRIPTDIR  := $(DESTDIR)$(PREFIX)/share/cosmic-deb/scripts

TAG_ARG    := $(if $(TAG),-tag $(TAG),)

.PHONY: all build clean install uninstall run run-tui run-skip-deps run-only update-repos fmt vet tidy help

all: build

banner:
	@echo "________________________________________________________________________________"
	@echo "  ____ ___  ____  __  __ ___ ____   ____  _____ ____ "
	@echo " / ___/ _ \\/ ___||  \\/  |_ _/ ___| |  _ \\| ____| __ )"
	@echo "| |  | | | \\___ \\| |\\/| || | |     | | | |  _| |  _ \\"
	@echo "| |__| |_| |___) | |  | || | |___  | |_| | |___| |_)"
	@echo " \\____\\___/|____/|_|  |_|___\\____| |____/|_____|____/ "
	@echo "________________________________________________________________________________"
	@echo ""

build: banner
	@echo ">> Building $(BINARY)..."
	@$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) .
	@echo ">> Build complete: $(BINARY)"

clean: banner
	@echo ">> Cleaning workspace..."
	@rm -f $(BINARY)
	@rm -rf $(OUTDIR)
	@echo ">> Done."

install: build
	@echo ">> Installing to $(PREFIX)..."
	@install -d $(BINDIR)
	@install -m 0755 $(BINARY) $(BINDIR)/$(BINARY)
	@install -d $(SCRIPTDIR)
	@install -m 0755 scripts/install-local.sh    $(SCRIPTDIR)/install-local.sh
	@install -m 0755 scripts/install-release.sh  $(SCRIPTDIR)/install-release.sh
	@install -m 0755 scripts/uninstall.sh        $(SCRIPTDIR)/uninstall.sh
	@echo ">> Installation complete."

uninstall:
	@echo ">> Uninstalling..."
	@rm -f $(BINDIR)/$(BINARY)
	@rm -rf $(DESTDIR)$(PREFIX)/share/cosmic-deb
	@echo ">> Done."

run: build
	@echo ">> Starting $(BINARY)..."
	@sudo ./$(BINARY) $(TAG_ARG) -repos $(REPOS) -outdir $(OUTDIR) -workdir $(WORKDIR) -jobs $(JOBS)

run-tui: build
	@echo ">> Starting $(BINARY) in TUI mode..."
	@sudo ./$(BINARY) $(TAG_ARG) -repos $(REPOS) -tui

run-skip-deps: build
	@echo ">> Starting $(BINARY) (skipping dependencies)..."
	@sudo ./$(BINARY) $(TAG_ARG) -repos $(REPOS) -outdir $(OUTDIR) -workdir $(WORKDIR) -jobs $(JOBS) -skip-deps

run-only: build
	@if [ -z "$(COMPONENT)" ]; then \
		echo "ERROR: Specify a component with COMPONENT=<n>, e.g. make run-only COMPONENT=cosmic-term"; \
		exit 1; \
	fi
	@echo ">> Starting $(BINARY) for $(COMPONENT)..."
	@sudo ./$(BINARY) $(TAG_ARG) -repos $(REPOS) -outdir $(OUTDIR) -workdir $(WORKDIR) -jobs $(JOBS) -only $(COMPONENT)

update-repos: build
	@echo ">> Fetching latest epoch tags from upstream..."
	@./$(BINARY) -repos $(REPOS) -update-repos

fmt:
	@echo ">> Formatting source code..."
	@$(GO) fmt ./...

vet:
	@echo ">> Running static analysis..."
	@$(GO) vet ./...

tidy:
	@echo ">> Tidying Go modules..."
	@$(GO) mod tidy

help: banner
	@echo "COSMIC-DEB: Debian Package Builder for COSMIC Desktop"
	@echo ""
	@echo "Primary Commands:"
	@echo "  build              Compile the cosmic-deb binary"
	@echo "  clean              Remove compiled binary and output packages"
	@echo "  install            Install binary and scripts to $(PREFIX)"
	@echo "  uninstall          Remove installed binary and scripts"
	@echo ""
	@echo "Execution Commands:"
	@echo "  run                Build and package all COSMIC components"
	@echo "  run-tui            Launch interactive TUI wizard and monitor"
	@echo "  run-skip-deps      Build all components, skipping dependency installation"
	@echo "  run-only           Build a specific component (requires COMPONENT=<n>)"
	@echo "  update-repos       Fetch latest epoch tags and update repos.json"
	@echo ""
	@echo "Maintenance Commands:"
	@echo "  fmt                Format Go source files"
	@echo "  vet                Run Go static analysis"
	@echo "  tidy               Tidy the Go module dependencies"
	@echo ""
	@echo "Variables:"
	@echo "  TAG=<tag>          Override all repo tags, e.g. TAG=epoch-1.0.7 (optional)"
	@echo "  REPOS=$(REPOS)      Repos config file"
	@echo "  OUTDIR=$(OUTDIR)     Output directory"
	@echo "  WORKDIR=$(WORKDIR)    Working directory"
	@echo "  JOBS=$(JOBS)            Parallel jobs"
	@echo "  COMPONENT=<n>      Component name for run-only"
	@echo ""

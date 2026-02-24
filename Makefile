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

.PHONY: all build clean install uninstall run run-tui run-skip-deps run-only run-branch update-repos fmt vet tidy help

all: build

build:
	@echo ">> Building $(BINARY)..."
	@$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) .
	@echo ">> Build complete."

clean:
	@echo ">> Cleaning workspace..."
	@rm -f $(BINARY)
	@rm -rf $(OUTDIR)
	@echo ">> Workspace cleaned."

install: build
	@echo ">> Installing to $(PREFIX)..."
	@install -d $(BINDIR)
	@install -m 0755 $(BINARY) $(BINDIR)/$(BINARY)
	@install -d $(SCRIPTDIR)
	@install -m 0755 scripts/install-local.sh   $(SCRIPTDIR)/install-local.sh
	@install -m 0755 scripts/install-release.sh $(SCRIPTDIR)/install-release.sh
	@install -m 0755 scripts/uninstall.sh       $(SCRIPTDIR)/uninstall.sh
	@echo ">> Installed."

uninstall:
	@echo ">> Uninstalling from $(PREFIX)..."
	@rm -f $(BINDIR)/$(BINARY)
	@rm -rf $(DESTDIR)$(PREFIX)/share/cosmic-deb
	@echo ">> Uninstalled."

run: build
	@echo ">> Starting $(BINARY)..."
	@./$(BINARY) $(TAG_ARG) -repos $(REPOS) -outdir $(OUTDIR) -workdir $(WORKDIR) -jobs $(JOBS)

run-tui: build
	@echo ">> Launching TUI interface..."
	@./$(BINARY) $(TAG_ARG) -repos $(REPOS) -tui

run-branch: build
	@echo ">> Starting $(BINARY) with main branch HEAD..."
	@./$(BINARY) -repos $(REPOS) -outdir $(OUTDIR) -workdir $(WORKDIR) -jobs $(JOBS) -use-branch

run-skip-deps: build
	@echo ">> Running without dependency installation..."
	@./$(BINARY) $(TAG_ARG) -repos $(REPOS) -outdir $(OUTDIR) -workdir $(WORKDIR) -jobs $(JOBS) -skip-deps

run-only: build
	@if [ -z "$(COMPONENT)" ]; then \
		echo "Error: Specify COMPONENT=<name>"; \
		exit 1; \
	fi
	@echo ">> Packaging $(COMPONENT)..."
	@./$(BINARY) $(TAG_ARG) -repos $(REPOS) -outdir $(OUTDIR) -workdir $(WORKDIR) -jobs $(JOBS) -only $(COMPONENT)

update-repos: build
	@echo ">> Refreshing repository epoch tags..."
	@./$(BINARY) -repos $(REPOS) -update-repos
	@echo ">> Metadata updated."

fmt:
	@echo ">> Formatting source..."
	@$(GO) fmt ./...

vet:
	@echo ">> Vetting source..."
	@$(GO) vet ./...

tidy:
	@echo ">> Tidying module dependencies..."
	@$(GO) mod tidy

help:
	@echo "Usage: make <target> [VARIABLES]"
	@echo ""
	@echo "Targets:"
	@echo "  build              Compile the orchestrator"
	@echo "  run                Full build pipeline (interactive source selection)"
	@echo "  run-tui            Launch TUI configuration wizard"
	@echo "  run-branch         Build from main branch HEAD"
	@echo "  run-only           Build single component (COMPONENT=name)"
	@echo "  update-repos       Fetch latest epoch tags from upstream"
	@echo "  install            Install binary and scripts to system paths"
	@echo "  uninstall          Remove system installation"
	@echo "  clean              Remove binary and output directory"
	@echo ""
	@echo "Variables:"
	@echo "  TAG=epoch-x.x.x    Override all repository tags"
	@echo "  OUTDIR=path        Output directory for .deb files"
	@echo "  WORKDIR=path       Build staging directory"
	@echo "  JOBS=n             Parallel compilation jobs"
	@echo "  COMPONENT=name     Component name for run-only"

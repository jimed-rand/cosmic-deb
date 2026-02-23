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

.PHONY: all build clean install uninstall run run-tui run-skip-deps run-only update-repos fmt vet tidy help banner

all: build

banner:
	@echo "--------------------------------------------------------------------------------"
	@echo "  ____ ___  ____  __  __ ___ ____   ____  _____ ____ "
	@echo " / ___/ _ \/ ___||  \/  |_ _/ ___| |  _ \| ____| __ )"
	@echo "| |  | | | \___ \| |\/| || | |     | | | |  _| |  _ \ "
	@echo "| |__| |_| |___) | |  | || | |___  | |_| | |___| |_)"
	@echo " \____\___/|____/|_|  |_|___\____| |____/|_____|____/ "
	@echo "--------------------------------------------------------------------------------"
	@echo ""

build: banner
	@echo ">> Building $(BINARY)..."
	@$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) .
	@echo ">> Build complete."

clean: banner
	@echo ">> Cleaning workspace..."
	@rm -f $(BINARY)
	@rm -rf $(OUTDIR)
	@echo ">> Workspace cleaned."

install: build
	@echo ">> Installing to $(PREFIX)..."
	@install -d $(BINDIR)
	@install -m 0755 $(BINARY) $(BINDIR)/$(BINARY)
	@install -d $(SCRIPTDIR)
	@install -m 0755 scripts/install-local.sh    $(SCRIPTDIR)/install-local.sh
	@install -m 0755 scripts/install-release.sh  $(SCRIPTDIR)/install-release.sh
	@install -m 0755 scripts/uninstall.sh        $(SCRIPTDIR)/uninstall.sh
	@echo ">> Binary and scripts installed."

uninstall: banner
	@echo ">> Uninstalling from $(PREFIX)..."
	@rm -f $(BINDIR)/$(BINARY)
	@rm -rf $(DESTDIR)$(PREFIX)/share/cosmic-deb
	@echo ">> Files removed."

run: build
	@echo ">> Starting $(BINARY)..."
	@sudo ./$(BINARY) $(TAG_ARG) -repos $(REPOS) -outdir $(OUTDIR) -workdir $(WORKDIR) -jobs $(JOBS)

run-tui: build
	@echo ">> Launching TUI interface..."
	@sudo ./$(BINARY) $(TAG_ARG) -repos $(REPOS) -tui

run-skip-deps: build
	@echo ">> Running without dependency installation..."
	@sudo ./$(BINARY) $(TAG_ARG) -repos $(REPOS) -outdir $(OUTDIR) -workdir $(WORKDIR) -jobs $(JOBS) -skip-deps

run-only: build
	@if [ -z "$(COMPONENT)" ]; then \
		echo "Error: Specify COMPONENT=<name>"; \
		exit 1; \
	fi
	@echo ">> Packaging $(COMPONENT)..."
	@sudo ./$(BINARY) $(TAG_ARG) -repos $(REPOS) -outdir $(OUTDIR) -workdir $(WORKDIR) -jobs $(JOBS) -only $(COMPONENT)

update-repos: build
	@echo ">> Refreshing repository epoch tags..."
	@./$(BINARY) -repos $(REPOS) -update-repos
	@echo ">> Metadata updated."

fmt: banner
	@echo ">> Standardising source formatting..."
	@$(GO) fmt ./...

vet: banner
	@echo ">> Analyzing code for common mistakes..."
	@$(GO) vet ./...

tidy: banner
	@echo ">> Tidying module dependencies..."
	@$(GO) mod tidy

help: banner
	@echo "Usage:"
	@echo "  make <target> [VARIABLES]"
	@echo ""
	@echo "Targets:"
	@echo "  build              Compile the orchestrator"
	@echo "  run                Execute the full build pipeline"
	@echo "  run-tui            Start with interactive configuration"
	@echo "  run-only           Build specific component (COMPONENT=name)"
	@echo "  update-repos       Fetch latest release tags from upstream"
	@echo "  install            Deploy binary to system paths"
	@echo "  uninstall          Remove system deployment"
	@echo "  clean              Reset workspace to initial state"
	@echo ""
	@echo "Variables:"
	@echo "  TAG=epoch-x.x.x    Override all repository tags"
	@echo "  OUTDIR=path        Change package output path"
	@echo "  WORKDIR=path       Change build staging path"
	@echo "  JOBS=n             Concurrent compilation units"

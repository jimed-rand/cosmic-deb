# cosmic-deb

Build tool for packaging the [COSMIC Desktop Environment](https://system76.com/cosmic) as native Debian/Ubuntu `.deb` packages. Upstream packaging sources are maintained by [hepp3n on Codeberg](https://codeberg.org/hepp3n).

## Requirements

- Debian bookworm (12) or later, or Ubuntu jammy (22.04) or later
- `apt-get`, `git`, `curl`, `fakeroot`, `dpkg-dev`
- Go 1.24 or later (to build the orchestrator itself)

## Build the orchestrator

```sh
make build
```

## Usage

### Interactive (CLI)

```sh
./cosmic-deb
```

You will be prompted to select a source mode (epoch tag or main branch HEAD). Build dependencies are checked first; only missing packages will be installed (using `sudo` if not root). The maintainer field in generated packages is automatically set to credit **hepp3n** as the upstream packaging author.

Packages are named with the distro codename to prevent version collisions across releases, e.g. `cosmic-comp_1.0.0~noble_amd64.deb`.

After all components are built, source and staging directories are cleaned up automatically. You will then be offered the option to install the packages locally. Container environments (Docker, LXC, etc.) are detected and the install offer is skipped automatically.

### Interactive (TUI)

```sh
./cosmic-deb -tui
```

A full-screen wizard guides you through source mode selection, output paths, parallelism, and dependency handling. A live build monitor is shown during compilation.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-tui` | false | Launch TUI wizard instead of CLI prompts |
| `-repos` | `built-in` | Path to repos JSON config, or `built-in` |
| `-tag` | _(none)_ | Override epoch tag for all repos |
| `-use-branch` | false | Build from main branch HEAD |
| `-workdir` | `cosmic-work` | Directory for source and staging files |
| `-outdir` | `cosmic-packages` | Directory for output `.deb` files |
| `-jobs` | _(nproc)_ | Parallel compilation jobs |
| `-skip-deps` | false | Skip dependency installation check |
| `-only` | _(none)_ | Build a single named component |
| `-update-repos` | false | Fetch latest epoch tags and write `repos.json` |
| `-gen-config` | false | Write built-in config to `repos.json` |
| `-dev-finder` | false | Regenerate `pkg/repos/finder.go` from current config |

### Makefile targets

```sh
make run                    # Full pipeline, interactive source selection
make run-tui                # Full pipeline with TUI wizard
make run-branch             # Build from main branch HEAD
make run-only COMPONENT=cosmic-term
make run-skip-deps          # Skip dep check (assumes deps already installed)
make update-repos           # Fetch latest epoch tags
make install                # Install binary and scripts to /usr/local
make uninstall              # Remove installation
make clean                  # Remove binary and output directory
```

## Source mode

**Epoch tags** correspond to versioned COSMIC releases (e.g. `epoch-1.0.0`). Each repository may carry its own tag, or you may apply a single tag globally with `-tag`.

**Branch HEAD** builds the latest untagged commit from each repository's default branch. This may be unstable.

## Build procedure

1. Dependency check — missing APT packages are installed (sudo only if required). Rust toolchain and `just` are ensured.
2. Build order — all components are sorted A–Z before any compilation begins, preventing partial-build confusion.
3. For each component: source is downloaded (tarball first, git clone fallback), vendored if a justfile vendor target is present, compiled, and build output is validated before staging.
4. Packaging — a `DEBIAN/control` is generated, runtime dependencies are declared, and `fakeroot dpkg-deb` assembles the `.deb`. Package filenames include the distro codename.
5. Meta package — `cosmic-desktop` is assembled as a convenience dependency on all built components.
6. Cleanup — source trees and staging directories are removed after all packages are assembled.
7. Local install offer — if not running inside a container, you are asked whether to install the packages immediately.

## Installation scripts

| Script | Purpose |
|--------|---------|
| `scripts/install-local.sh [dir]` | Install locally built packages from a directory |
| `scripts/install-release.sh [tag]` | Download and install a GitHub release |
| `scripts/uninstall.sh` | Remove all COSMIC packages |

## Package structure

```
cosmic-deb/
├── main.go                    # Orchestrator entry point
├── exec.go                    # Shell exec helpers
├── go.mod / go.sum
├── Makefile
├── README.md
├── pkg/
│   ├── build/
│   │   ├── compile.go         # Compilation, vendor, staging install
│   │   ├── deps.go            # Dependency installation and Rust toolchain
│   │   ├── source.go          # Source download and git clone
│   │   └── version.go         # Version detection
│   ├── debian/
│   │   └── package.go         # .deb assembly and meta package
│   ├── distro/
│   │   ├── deps.go            # Per-distro build dependency resolution
│   │   └── detect.go          # Distro detection, support check, container detection
│   ├── repos/
│   │   ├── finder.go          # Built-in repo list (hepp3n/Codeberg)
│   │   ├── loader.go          # Config loading, epoch tag queries, update
│   │   └── types.go           # Repo entry and config types
│   └── tui/
│       ├── monitor.go         # Live build monitor
│       └── wizard.go          # Configuration wizard
└── scripts/
    ├── install-local.sh
    ├── install-release.sh
    └── uninstall.sh
```

## Credits

Upstream Debian packaging maintained by **hepp3n** — [codeberg.org/hepp3n](https://codeberg.org/hepp3n).

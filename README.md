# cosmic-deb

COSMIC Debian Builder or `cosmic-deb`, is a builder tool for creating Debian/Ubuntu packages for COSMIC desktop.

## Abstract

The `cosmic-deb` initiative represents a sophisticated build orchestration framework, meticulously engineered in the Go programming language to provide a seamless and highly automated pathway for the acquisition, compilation, and subsequent packaging of the COSMIC™ Desktop Environment into standardised Debian binary archives (`.deb`). In the contemporary landscape of open-source software development, the transition of modern, Rust-based source implementations into the rigorous packaging hierarchies required by the Debian and Ubuntu ecosystems involves a multifaceted interplay of complex dependency resolution, toolchain synchronisation, and meticulous metadata management. This framework serves as a critical systemic bridge, abstracting these inherent complexities through a robust architectural design that prioritises systemic build integrity, distribution-specific compatibility, and operational efficiency, thereby allowing users and developers to bypass the traditional manual barriers associated with high-level environment synthesis.

Central to the operational efficacy of this orchestration utility is its strategic and seamless integration with the verified community-led repository forks, specifically those curated by the developer hepp3n on the Codeberg platform, which provide the essential `debian/` administrative structures necessary for professional-grade packaging. These repositories are distinguished by their inclusion of detailed metadata—encompassing control files, build rules, and properly formatted changelogs—that define the exact build-time and runtime requirements for each modular component of the COSMIC™ suite, ranging from core session managers to high-level system configurations. The logic embedded within `cosmic-deb` dynamically identifies the appropriate build systems—be it Just, Make, or Cargo—and executes release-grade optimisations while simultaneously automating the procurement of intricate dependencies, utilising high-performance technologies such as the `mold` linker to significantly reduce the temporal latency typically associated with large-scale compilation tasks.

Furthermore, the `cosmic-deb` project is underpinned by a profound commitment to the philosophical principles of open-source collaboration and proactive intellectual property stewardship, ensuring that the synthesis process remains entirely transparent and respectful of the original authors' work. While the orchestrator itself is distributed under the GNU General Public License v2.0, it operates in total alignment with the licensing regimes of the upstream COSMIC™ components developed by System76 and the packaging efforts of the wider community, preserving all original metadata and legal markers throughout the build lifecycle. Ultimately, this framework is envisioned as a scholarly and technical endeavour to democratise access to the next generation of memory-safe desktop environments, providing a robust, academically grounded, and fully automated methodology that empowers the global Linux community to experience the future of desktop computing on their preferred APT-based distributions while adhering to the highest standards of software engineering and collaborative development.

## Systemic Requirements and Distribution Compatibility

The efficacy of this framework is contingent upon its deployment within an APT-based Linux distribution. Current support is prioritised for modern Debian releases and Ubuntu Long Term Support (LTS) or development cycles, ensuring a stable yet forward-looking environment for the COSMIC™ ecosystem.

| Distribution | Supported Release Iterations |
|:---|:---|
| **Debian GNU/Linux** | Version 12 (Bookworm) and subsequent stable/testing branches (Trixie, Forky, Sid). |
| **Ubuntu Linux** | LTS releases (22.04 Jammy, 24.04 Noble, 26.04 Resolute) and development branch (devel). |

*Note: Package availability is resolved at runtime per detected codename; only LTS and devel releases are supported.*

## Distro-Aware Dependency Resolution

Build dependencies are resolved dynamically at runtime based on the detected distribution and release codename. This ensures that packages installed are appropriate for the host system and avoids conflicts caused by package naming differences or availability gaps across releases.

The following table summarises per-release dependency behaviour, verified against packages.debian.org and packages.ubuntu.com:

| Package | Debian Bookworm | Debian Trixie/Forky/Sid | Ubuntu Jammy | Ubuntu Noble/Resolute |
|:---|:---:|:---:|:---:|:---:|
| `libdisplay-info-dev` | No | Yes | No | Yes |
| `rust-all` | Yes | Yes | Yes | Yes |
| `dh-cargo` | Yes | Yes | Yes | Yes |
| `just` (via apt) | No | Yes | No | No |

When `just` is not available through APT (Debian Bookworm, all Ubuntu releases), `cosmic-deb` automatically installs it via `cargo install just` after the toolchain setup phase. The `libdisplay-info-dev` package is conditionally added to the build dependency set for both global and per-component (`cosmic-comp`, `cosmic-settings`) resolution only on releases where it is available in the official repositories. All other conditional packages (`rust-all`, `dh-cargo`) are available across all supported Debian and Ubuntu releases and are included unconditionally per distro family.

## Methodological Compilation

Prior to the execution of the build process, the host system must be equipped with the Go toolchain (version 1.21 or later). The compilation of the `cosmic-deb` binary is achieved through a standardised `Makefile` procedure, which encapsulates the necessary build directives:

```bash
make build
```

## Source Mode Selection

`cosmic-deb` supports two distinct source acquisition strategies, selectable at runtime via both the interactive CLI prompt and the TUI wizard:

**Default Branch (Recommended)** — Clones or downloads the current HEAD of each repository's default branch (typically `master` for hepp3n's Codeberg forks). This mode tracks the latest packaging state including the `debian/` directory metadata, and is the recommended approach since hepp3n manages versioning through `debian/changelog` on the primary branch rather than through git tags. The version number is extracted dynamically from the `debian/changelog` file at build time.

**Epoch Tag** — Checks out a specific, versioned release tag (e.g., `epoch-1.0.0`) from each repository. This is the reproducible option and is appropriate when a known-good baseline is required. Available tags are discovered dynamically from the upstream repositories via `git ls-remote` without any hardcoded version references.

The source mode can also be forced non-interactively via the `-use-branch` flag:

```bash
./cosmic-deb -use-branch
```

## Configuration and Metadata Integration

A distinguishing feature of this utility is the encapsulation of repository metadata directly within the binary via `finder.go`. This design decision minimises external file dependencies, thereby enhancing the portability and integrity of the build process. The metadata points exclusively to verified Codeberg forks maintained by hepp3n, which contain the essential `debian/` directory structures (including `control`, `rules`, and `changelog` files) required for native Debianisation. No version information is hardcoded; all tags and versions are resolved dynamically at runtime from the upstream repositories.

To synchronise the embedded configuration with the latest upstream epoch tags, the following command should be executed periodically:

```bash
./cosmic-deb -update-repos
```

## Source Distribution Rationale

The decision to utilise hepp3n's Codeberg forks over the primary System76 GitHub repositories is rooted in the necessity for standardised Debian packaging logic. These forks integrate critical metadata that governs:

*   **Dependency Resolution**: Detailed Build-Depends and runtime Depends within `debian/control`.
*   **Build Directives**: Customised build instructions in `debian/rules` that utilise the `just` automation tool for vendoring and compilation.
*   **Version Tracking**: Properly formatted Debian changelogs to ensure seamless upgrades across various APT repositories.

Where a `debian/` directory is identified, `cosmic-deb` invokes the `dpkg-buildpackage` suite for a native packaging experience. In instances where such metadata is absent, the tool reverts to a manual compilation and packaging fallback to ensure continuity.

## Operational Procedure

### Comprehensive Environment Synthesis

The `cosmic-deb` orchestrator is designed to operate within a standard user environment, necessitating elevated privileges only for the installation of systemic dependencies. By executing the utility as a non-privileged user, the resulting compilation artifacts and source repositories remain owned by the current user, thereby ensuring a cleaner and more secure build lifecycle. The orchestrator will proactively prompt for a `sudo` password when systemic modifications are required.

Execution of the orchestrator is initiated as follows:

```bash
./cosmic-deb
```

Upon launch, the interactive prompt will ask you to select a source mode. Enter `b` to build from the main branch HEAD, or select a numbered epoch tag from the list:

```
Select source mode:
  [b] Latest (main branch HEAD)
  [0] epoch-1.0.7
  [*] Use per-repo tags from repos config
Select option (b / index / Enter for per-repo tags):
```

### Advanced Execution Parameters

The behaviour of `cosmic-deb` can be modified through several command-line flags, allowing for granular control over the build environment.

| Parameter | Default Value | Technical Description |
|:---|:---|:---|
| `-repos` | `built-in` | Specifies an alternative path for repository configuration. |
| `-update-repos` | `false` | Triggers a metadata update of epoch tags followed by immediate termination. |
| `-tag` | *(null)* | Enforces a specific version tag across all managed repositories. |
| `-use-branch` | `false` | Build from main branch HEAD instead of epoch tags. |
| `-workdir` | `cosmic-work` | Designates the directory for source retrieval and compilation. |
| `-outdir` | `cosmic-packages` | Output location for the resulting `.deb` binary packages. |
| `-jobs` | *CPU Count* | Determines the level of parallelism during the compilation phase. |
| `-skip-deps` | `false` | Bypasses the automated installation of requisite build dependencies. |
| `-only` | *(null)* | Isolates the build process to a singular named component. |
| `-tui` | `false` | Initialises an interactive Terminal User Interface (TUI) wizard. |

## Deployment and Removal

### Installation Strategy

Once the compilation has concluded successfully, the generated packages should be installed using the `dpkg` utility, followed by an APT resolution pass to ensure all runtime dependencies are met:

```bash
sudo dpkg -i cosmic-packages/cosmic-desktop_*.deb
sudo apt-get install -f
```

For a more streamlined experience, a local installation script is provided:

```bash
sudo bash scripts/install-local.sh cosmic-packages
```

### System Restoration

To purge the COSMIC™ components from the host system, the provided uninstallation script should be utilised:

```bash
sudo bash scripts/uninstall.sh
```

## Technical Architecture and Dependency Mapping

The framework maintains a comprehensive mapping of both build-time and runtime requirements, derived through rigorous analysis of upstream metadata. The following table highlights critical dependencies for primary components:

| Component | Essential Runtime Dependencies |
|:---|:---|
| `cosmic-session` | Core COSMIC packages, `gnome-keyring`, `xwayland`, `fonts-open-sans`. |
| `cosmic-comp` | `libegl1`, `libwayland-server0`, and associated Wayland drivers. |
| `cosmic-greeter` | `greetd`, `adduser`, `cosmic-comp`, `dbus`. |
| `cosmic-settings` | `accountsservice`, `iso-codes`, `network-manager-gnome`. |
| `cosmic-launcher` | `pop-launcher` and its associated plugins. |
| `cosmic-player` | `gstreamer1.0-plugins-base`, `gstreamer1.0-plugins-good`. |
| `pop-launcher` | `qalc` (calculator integration), `fd-find`. |

## Procedural Sophistication

The orchestration logic prioritises high-performance retrieval via source archives, with a secondary fallback to shallow Git cloning mechanisms. Build system identification is automated, recognising `Justfiles`, `Makefiles`, and `Cargo.toml` configurations to apply appropriate release-grade optimisations.

The build environment is further enhanced by the integration of the `mold` linker and `nasm` assembler where applicable, significantly reducing compilation latency. Rust stability is ensured through automated `rustup` configurations, maintaining a consistent toolchain version across the entirety of the COSMIC™ suite.

## Legal and Ethical Framework

### Licensing Provisions

`cosmic-deb` is distributed under the terms of the **GNU General Public License v2.0 (GPL-2.0)**. The comprehensive text of this license is available within the `LICENSE` file of this repository.

It is imperative to note that the upstream COSMIC™ components and the associated Debian packaging sources retrieved during the execution phase are governed by their respective licenses (predominantly **GPL-3.0**). This framework is designed to operate in total alignment with these legal requirements, ensuring that original metadata and license markers are preserved throughout the packaging lifecycle.

### Intellectual Property Statement

This project represents an independent academic and technical endeavour aimed at broadening the accessibility of the COSMIC™ Desktop Environment.

*   **Honouring Original Authorship**: We hold the innovative work of the **System76 / Pop!_OS team** in the highest esteem. Their pioneering efforts in developing a modern, memory-safe desktop environment are the fundamental motivation for this project.
*   **Acknowledgement of Fork Maintainers**: We express our profound gratitude to **hepp3n (Piotr)** for his meticulous maintenance of the Debian packaging forks. His contributions are the cornerstone upon which this automated orchestrator is built.
*   **Community Objectives**: `cosmic-deb` is envisioned as a contribution to the global Linux community, providing a robust mechanism for enthusiasts and developers to experience and test the next generation of desktop computing on their preferred APT-based platforms.

## Acknowledgements

*   **System76 / Pop!_OS Team**: The primary architects and visionaries of the COSMIC™ Desktop Environment.
*   **hepp3n (Piotr)**: For the dedicated provision of Debian/Ubuntu-compatible packaging infrastructure.
*   **James Ed Randson (jimed-rand)**: Lead developer and maintainer of the `cosmic-deb` framework.
*   **The Global Open Source Collective**: For the continuous advancement of the technologies that make this utility possible.

---
*COSMIC™ is a registered trademark of System76. This project is developed independently and does not imply endorsement by, or affiliation with, System76.*

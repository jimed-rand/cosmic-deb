# COSMIC-DEB: An Automated Build Orchestration Framework for the COSMIC Desktop Environment

## Abstract

`cosmic-deb` represents a sophisticated build orchestration framework engineered in the Go programming language, specifically designed to facilitate the automated retrieval, compilation, and subsequent packaging of the COSMIC™ Desktop Environment into Debian-compliant binary archives (`.deb`). Recognising the complexities inherent in multi-component desktop environments, this utility serves as a critical bridge between the upstream Rust-based source implementations and the rigorous packaging standards required for the Debian and Ubuntu ecosystems. By leveraging the comprehensive `debian/` metadata maintained within [hepp3n's Codeberg repositories](https://codeberg.org/hepp3n), `cosmic-deb` ensures that each component—ranging from core session managers to modular applications—is synthesised with full adherence to build-time and runtime dependency specifications.

## Systemic Requirements and Distribution Compatibility

The efficacy of this framework is contingent upon its deployment within an APT-based Linux distribution. Current support is prioritised for modern Debian releases and Ubuntu Long Term Support (LTS) or development cycles, ensuring a stable yet forward-looking environment for the COSMIC™ ecosystem.

| Distribution | Supported Release Iterations |
|:---|:---|
| **Debian GNU/Linux** | Version 12 (Bookworm) and subsequent stable/testing branches. |
| **Ubuntu Linux** | Current LTS releases (e.g., 22.04 Jammy, 24.04 Noble) and development branches (e.g., Plucky Puffin). |

*Note: Non-LTS intermediate releases are deprecated in favour of more robust development cycles and stable long-term support foundations.*

## Methodological Compilation

Prior to the execution of the build process, the host system must be equipped with the Go toolchain (version 1.21 or later). The compilation of the `cosmic-deb` binary is achieved through a standardised `Makefile` procedure, which encapsulates the necessary build directives:

```bash
make build
```

## Configuration and Metadata Integration

A distinguishing feature of this utility is the encapsulation of repository metadata directly within the binary via `finder.go`. This design decision minimises external file dependencies, thereby enhancing the portability and integrity of the build process. The metadata points exclusively to verified Codeberg forks maintained by hepp3n, which contain the essential `debian/` directory structures (including `control`, `rules`, and `changelog` files) required for native Debianisation.

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

As the build process involves the installation of systemic dependencies and the generation of privileged package files, root access is mandatory. Execution of the orchestrator is initiated as follows:

```bash
sudo ./cosmic-deb
```

### Advanced Execution Parameters

The behaviour of `cosmic-deb` can be modified through several command-line flags, allowing for granular control over the build environment.

| Parameter | Default Value | Technical Description |
|:---|:---|:---|
| `-repos` | `built-in` | Specifies an alternative path for repository configuration. |
| `-update-repos` | `false` | Triggers a metadata update of epoch tags followed by immediate termination. |
| `-tag` | *(null)* | Enforces a specific version tag across all managed repositories. |
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

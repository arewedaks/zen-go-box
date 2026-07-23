<div align="center">
  <h1>🛡️ ZenGoBox</h1>
  <p><strong>A Modern, Golang-based Transparent Proxy Manager for Rooted Android</strong></p>

  <p>
    <img src="https://img.shields.io/badge/Language-Go-00ADD8?style=for-the-badge&logo=go" alt="Golang" />
    <img src="https://img.shields.io/badge/Platform-Android_Root-3DDC84?style=for-the-badge&logo=android" alt="Android" />
    <img src="https://img.shields.io/badge/Status-Active-success?style=for-the-badge" alt="Status" />
  </p>
</div>

---

## ⚡ Overview

**ZenGoBox** (formerly *Box for Magisk*) is a powerful, lightweight, and fully portable transparent proxy daemon manager written entirely in Go. Designed exclusively for rooted Android devices (Magisk, KernelSU, APatch), it orchestrates network traffic rules (iptables) and manages proxy cores seamlessly without the bloat of traditional bash scripts.

## ✨ Key Features

- **🚀 Written in Go:** Extremely fast, compiled binary with low memory footprint and zero shell-script overhead.
- **🔄 Multi-Core Support:** Seamlessly integrates and orchestrates popular proxy cores.
- **📱 Root Environment Agnostic:** Automatically detects and adapts to Magisk, KernelSU, or APatch.
- **🧩 100% Portable:** The binary self-discovers its working directory, meaning it can run from *anywhere* in the filesystem.
- **📦 Auto-Extraction Setup:** Configurations and templates are embedded inside the binary and extracted automatically upon setup.
- **🤖 Smart Geo Updater:** Built-in smart downloader for `geoip.dat` & `geosite.dat` that targets specific cores accurately.
- **🎛️ Cobra CLI:** Enjoy a modern terminal experience with `bash`, `zsh`, and `fish` auto-completion support.

## 🧩 Supported Cores

- [Mihomo (Clash)](https://github.com/MetaCubeX/mihomo) *(Default)*
- [Sing-box](https://github.com/SagerNet/sing-box)
- [Xray-core](https://github.com/XTLS/Xray-core)
- [v2ray-core (v2fly)](https://github.com/v2fly/v2ray-core)
- [Hysteria](https://github.com/apernet/hysteria)

## 🛠️ Getting Started

### 1. Installation
Because ZenGoBox is **100% portable**, you can place the binary in *any* directory. It will automatically detect its location and build the proxy environment right there!

For example, let's set it up in a custom folder:
```bash
# Create a folder and push the binary
mkdir -p /data/local/tmp/myproxy
cp zengobox /data/local/tmp/myproxy/
su -c "chmod 755 /data/local/tmp/myproxy/zengobox"
```
*(If you are building a Magisk Module, placing it in `/data/adb/zengobox/bin/zengobox` is recommended, and the system will intelligently use `/data/adb/zengobox` as the root).*

### 2. Initialization & Setup
Run the setup command from your chosen directory. It will automatically extract templates, download Geo databases, fetch the proxy kernel, and install the Web Dashboard in one go!
```bash
# Navigate to your folder
cd /data/local/tmp/myproxy/

# Setup for Clash (Default)
su -c "./zengobox setup clash"

# Setup for Sing-box
su -c "./zengobox setup sing-box"

# Setup for Xray
su -c "./zengobox setup xray"

# Setup for v2fly
su -c "./zengobox setup v2fly"

# Setup for Hysteria
su -c "./zengobox setup hysteria"

# Or setup ALL cores simultaneously
su -c "./zengobox setup all"
```
*Note: You can edit the main configuration at `zengobox.yaml` in your folder after setup.*

## 💻 CLI Commands

ZenGoBox acts as a unified command-line tool. Here are the core commands you'll use:

| Command | Description |
| :--- | :--- |
| `zengobox start` | Starts the proxy service and applies transparent routing rules. |
| `zengobox stop` | Stops the proxy service and cleans up iptables rules. |
| `zengobox restart` | Restarts the proxy service instantly. |
| `zengobox toggle` | Toggles the proxy state (used by Magisk Action button). |
| `zengobox status` | Checks the running status and PID of the proxy core. |
| `zengobox setup [core]` | Extracts templates, downloads GeoIP, Kernel, and Web Dashboard. |
| `zengobox config check` | Validates the syntax of your `zengobox.yaml`. |
| `zengobox log` | Displays a real-time tail of the daemon logs. |
| `zengobox update kernel` | Downloads the latest proxy core binary (e.g., Mihomo/Sing-box). |
| `zengobox update geo` | Updates `geoip.dat` and `geosite.dat` routing databases. |
| `zengobox update sub` | Updates your provider subscription links. |
| `zengobox update dash` | Updates the Web UI Dashboard (Zashboard). |
| `zengobox update all` | Runs all update commands (kernel, geo, sub, dash) sequentially. |
| `zengobox daemon` | Background watcher process (handles boot and network changes). |
| `zengobox version` | Displays the binary version, architecture, and root environment. |
| `zengobox completion` | Generates shell autocomplete scripts (Bash/Zsh/Fish). |

## 🏗️ Building from Source

To compile the binary yourself, simply run the `make` command:
```bash
# Clone the repository
git clone https://github.com/arewedaks/zen-go-box.git
cd zen-go-box

# Build for Android ARM64
make build-arm64
```

## 📜 License
This project is open-source and heavily inspired by the original `box_for_magisk` concept, evolved into a full-fledged Go application.

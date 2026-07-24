<div align="center">
  <h1>ZenGoBox</h1>
  <p><strong>An Ultra-Fast & Portable Transparent Proxy Manager for Rooted Android</strong></p>

  <p>
    <img src="https://img.shields.io/badge/Language-Go-00ADD8?style=for-the-badge&logo=go" alt="Golang" />
    <img src="https://img.shields.io/badge/Platform-Android_Root-3DDC84?style=for-the-badge&logo=android" alt="Android" />
    <img src="https://img.shields.io/badge/Status-Active-success?style=for-the-badge" alt="Status" />
  </p>
</div>

---

## Overview

**ZenGoBox** is a powerful, lightweight, and fully portable transparent proxy daemon manager. Designed exclusively for rooted Android devices (Magisk, KernelSU, APatch), it orchestrates network traffic rules (iptables) and manages proxy cores seamlessly without the bloat of traditional bash scripts.

## Key Features

- **Ultra-Fast & Efficient:** High-performance compiled binary with low memory footprint and zero shell-script overhead.
- **Multi-Core Support:** Seamlessly integrates and orchestrates popular proxy cores.
- **Root Environment Agnostic:** Automatically detects and adapts to Magisk, KernelSU, or APatch.
- **100% Portable:** The binary self-discovers its working directory, meaning it can run from *anywhere* in the filesystem.
- **Auto-Extraction Setup:** Configurations and templates are embedded inside the binary and extracted automatically upon setup.
- **Non-Destructive Config Management:** ZenGoBox intelligently intercepts your original `config.yaml` and safely builds a temporary `run.yaml` in memory, ensuring your original configurations and comments are never permanently altered or destroyed.
- **Smart Geo Updater:** Built-in smart downloader for `geoip.dat` & `geosite.dat` that targets specific cores accurately.
- **Modern CLI:** Enjoy a modern terminal experience with `bash`, `zsh`, and `fish` auto-completion support.

## Supported Cores

- [Mihomo (Clash)](https://github.com/MetaCubeX/mihomo) *(Default)*
- [Sing-box](https://github.com/SagerNet/sing-box)
- [Xray-core](https://github.com/XTLS/Xray-core)
- [v2ray-core (v2fly)](https://github.com/v2fly/v2ray-core)
- [Hysteria](https://github.com/apernet/hysteria)

## Getting Started

### Method 1: Magisk / KernelSU Module (Recommended)
The easiest way to use ZenGoBox is by installing it as a systemless module:
1. Download the latest `ZenGoBox-module-vX.X.X.zip` from the [Releases](https://github.com/arewedaks/zen-go-box/releases) page.
2. Flash the ZIP file via **Magisk Manager**, **KernelSU**, or **APatch**.
3. Reboot your device.
4. Open the Magisk / KernelSU app, go to the Modules tab, and tap the **Action/WebUI** button on the ZenGoBox module.
5. The **ZenGoBox WebUI** will open automatically at `http://127.0.0.1:9999`. From there, you can install proxy cores, manage configurations, and start/stop the service seamlessly!

### Method 2: Manual Portable Setup (For Advanced Users)
Because ZenGoBox is **100% portable**, you can place the binary in *any* directory. It will automatically detect its location and build the proxy environment right there!

```bash
# 1. Create a folder and push the binary
mkdir -p /data/local/tmp/myproxy
cp zengobox /data/local/tmp/myproxy/
su -c "chmod 755 /data/local/tmp/myproxy/zengobox"

# 2. Run the setup command to extract templates & download dependencies
# Setup for Clash (Default)
su -c "/data/local/tmp/myproxy/zengobox setup clash"

# Or setup ALL cores simultaneously
su -c "/data/local/tmp/myproxy/zengobox setup all"
```
*Note: Once setup is complete, you can edit the main configuration at `zengobox.yaml` inside your folder or access the WebUI at `http://127.0.0.1:9999`.*

## CLI Commands

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
| `zengobox update dash` | Updates the Web UI Dashboard. |
| `zengobox update all` | Runs all update commands (kernel, geo, sub, dash) sequentially. |
| `zengobox daemon` | Background watcher process (handles boot and network changes). |
| `zengobox version` | Displays the binary version, architecture, and root environment. |
| `zengobox completion` | Generates shell autocomplete scripts (Bash/Zsh/Fish). |



## License
This project is open-source software.

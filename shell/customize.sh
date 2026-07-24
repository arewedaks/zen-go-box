SKIPUNZIP=1
ui_print "- Extracting ZenGoBox module files..."

# Setup directories
BOX_DIR="/data/adb/zengobox"
BIN_DIR="${BOX_DIR}/bin"

mkdir -p $BIN_DIR

# Detect architecture and extract appropriate zengobox binary
ARCH=$(getprop ro.product.cpu.abi)
if [ "$ARCH" = "arm64-v8a" ]; then
    ui_print "- Detected ARM64 architecture"
    unzip -o "$ZIPFILE" 'zengobox-arm64' -d $BIN_DIR >&2
    mv ${BIN_DIR}/zengobox-arm64 ${BIN_DIR}/zengobox
elif [ "$ARCH" = "armeabi-v7a" ] || [ "$ARCH" = "armeabi" ]; then
    ui_print "- Detected ARMv7 (32-bit) architecture"
    unzip -o "$ZIPFILE" 'zengobox-armv7' -d $BIN_DIR >&2
    mv ${BIN_DIR}/zengobox-armv7 ${BIN_DIR}/zengobox
else
    ui_print "! Unsupported architecture: $ARCH"
    abort "! ZenGoBox only supports arm64 and armv7"
fi

chmod 755 ${BIN_DIR}/zengobox

# Extract Magisk scripts
unzip -o "$ZIPFILE" 'uninstall.sh' -d $MODPATH >&2
unzip -o "$ZIPFILE" 'action.sh' -d $MODPATH >&2
unzip -o "$ZIPFILE" 'service.sh' -d $MODPATH >&2
unzip -o "$ZIPFILE" 'module.prop' -d $MODPATH >&2
unzip -o "$ZIPFILE" 'webroot/*' -d $MODPATH >&2

set_perm_recursive $MODPATH 0 0 0755 0755

ui_print "- ZenGoBox installed successfully!"
ui_print "- Run 'su -c /data/adb/zengobox/bin/zengobox setup clash' after reboot"

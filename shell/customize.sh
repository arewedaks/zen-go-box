SKIPUNZIP=1
ui_print "- Extracting ZenGoBox module files..."

# Setup directories
BOX_DIR="/data/adb/zengobox"
BIN_DIR="${BOX_DIR}/bin"

mkdir -p $BIN_DIR

# Extract zengobox binary
unzip -o "$ZIPFILE" 'zengobox' -d $BIN_DIR >&2
chmod 755 ${BIN_DIR}/zengobox

# Extract Magisk scripts
unzip -o "$ZIPFILE" 'uninstall.sh' -d $MODPATH >&2
unzip -o "$ZIPFILE" 'action.sh' -d $MODPATH >&2
unzip -o "$ZIPFILE" 'service.sh' -d $MODPATH >&2
unzip -o "$ZIPFILE" 'module.prop' -d $MODPATH >&2

set_perm_recursive $MODPATH 0 0 0755 0755

ui_print "- ZenGoBox installed successfully!"
ui_print "- Please run 'su -c zengobox setup clash' from terminal after reboot"

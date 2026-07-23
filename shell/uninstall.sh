#!/system/bin/sh

zennode_data_dir="/data/adb/zennode"

# Hentikan proxy service & bersihkan netfilter sebelum uninstall
if [ -f "${zennode_data_dir}/bin/zennode" ]; then
    "${zennode_data_dir}/bin/zennode" stop >/dev/null 2>&1
fi

rm_data() {
  if [ -d "${zennode_data_dir}" ]; then
    rm -rf "${zennode_data_dir}"
  fi

  if [ -f "/data/adb/ksu/service.d/zennode_service.sh" ]; then
    rm -rf "/data/adb/ksu/service.d/zennode_service.sh"
  fi

  if [ -f "/data/adb/service.d/zennode_service.sh" ]; then
    rm -rf "/data/adb/service.d/zennode_service.sh"
  fi
}

rm_data

#!/system/bin/sh

zengobox_data_dir="/data/adb/zengobox"

# Hentikan proxy service & bersihkan netfilter sebelum uninstall
if [ -f "${zengobox_data_dir}/bin/zengobox" ]; then
    "${zengobox_data_dir}/bin/zengobox" stop >/dev/null 2>&1
fi

rm_data() {
  if [ -d "${zengobox_data_dir}" ]; then
    rm -rf "${zengobox_data_dir}"
  fi
}

rm_data

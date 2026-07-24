#!/system/bin/sh

(
    until [ "$(getprop init.svc.bootanim)" = "stopped" ]; do
        sleep 10
    done

    # Kill existing zengobox daemon to prevent duplicates
    killall -9 zengobox >/dev/null 2>&1

    # Jalankan zengobox daemon core secara background
    if [ -f "/data/adb/zengobox/bin/zengobox" ]; then
        chmod 755 /data/adb/zengobox/bin/zengobox
        /data/adb/zengobox/bin/zengobox daemon >/dev/null 2>&1 &
    else
        echo "File /data/adb/zengobox/bin/zengobox not found" > "/data/adb/zengobox/run/zengobox_service.log"
    fi
) &

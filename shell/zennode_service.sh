#!/system/bin/sh

(
    until [ "$(getprop init.svc.bootanim)" = "stopped" ]; do
        sleep 10
    done

    # Jalankan zennode daemon core secara background
    if [ -f "/data/adb/zennode/bin/zennode" ]; then
        chmod 755 /data/adb/zennode/bin/zennode
        /data/adb/zennode/bin/zennode daemon >/dev/null 2>&1 &
    else
        echo "File /data/adb/zennode/bin/zennode not found" > "/data/adb/zennode/run/zennode_service.log"
    fi
) &

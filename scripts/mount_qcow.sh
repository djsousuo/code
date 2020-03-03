#!/bin/sh
MNT_POINT="./mount_qcow"
QEMU_NBD="/usr/bin/qemu-nbd"
NBD_DEV="/dev/nbd0"

QCOW_FILE=$1
if [[ -z $QCOW_FILE ]]; then
        echo "Usage: $0 <qcow file> <mount point>"
        exit 0
fi

modprobe nbd max_part=16
if [[ ! -b $NBD_DEV ]]; then
        echo "Error: Network Block Device module not loaded or compiled into kernel"
        exit 0
fi

if [[ ! -x $QEMU_NBD ]]; then
        echo "Error: Can't find qemu-nbd at: $QEMU_NBD"
        exit 0
fi

$QEMU_NBD -c $NBD_DEV $QCOW_FILE
if [[ $? -ne 0 ]]; then
        echo "Error: Couldn't connect nbd"
        exit 0
fi

n=0
cmd=$(fdisk -l $NBD_DEV | awk '/^\/dev/ {print $1}' | sed -e 's/\/dev\///g')
for i in $cmd; do
        echo -n "Mounting /dev/$i at $MNT_POINT/$i: "
        mkdir -p $MNT_POINT/$i
        mount /dev/$i $MNT_POINT/$i 2>/dev/null
        if [[ $? -ne 0 ]]; then
                echo "failed!"
                rmdir $MNT_POINT/$i
        else
                echo "done"
                n=$(($n+1))
        fi
done
echo "$n volumes mounted. unmount and do $QEMU_NBD -d $NBD_DEV when done"

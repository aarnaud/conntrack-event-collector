# conntrack-monitor

## Building it

1. Install Go
2. Install libnetfilter-conntrack-dev and libnfnetlink-dev
3. Compile

```bash
go build .
```

## cross-compile for openwrt

```bash
export OPENWRT_PATH=$HOME/openwrt/
export STAGING_DIR=${OPENWRT_PATH}/staging_dir/--TARGET--
export CC=${OPENWRT_PATH}/staging_dir/--TOOLCHAIN--/bin/XXXXX-openwrt-linux-gcc
export CC=${OPENWRT_PATH}/staging_dir/--TOOLCHAIN--/bin/XXXXX-openwrt-linux-g++
GOOS=linux GOARCH=arm CGO_ENABLED=1 GOARCH=arm go build
```

exemple:
```bash
export OPENWRT_PATH=$HOME/openwrt/
export STAGING_DIR=${OPENWRT_PATH}/staging_dir/target-arm_cortex-a9+vfpv3_uClibc-0.9.33.2_eabi
export CC=${OPENWRT_PATH}/staging_dir/toolchain-arm_cortex-a9+vfpv3_gcc-4.8-linaro_uClibc-0.9.33.2_eabi/bin/arm-openwrt-linux-gcc
export CXX=${OPENWRT_PATH}/staging_dir/toolchain-arm_cortex-a9+vfpv3_gcc-4.8-linaro_uClibc-0.9.33.2_eabi/bin/arm-openwrt-linux-g++
GOOS=linux GOARCH=arm CGO_ENABLED=1 GOARCH=arm go build
```
# --------------------- BASE LAYER ------------------------
FROM golang:1.12 AS base-layer
LABEL maintainer="guesslin@glasnostic.com"

# Install dependency
# pkg-config for cgo pkg-config
# curl for fetch codes
# bsdtar is from https://github.com/coreos/bugs/issues/1095
RUN apt-get update -qq && \
    apt-get install -y pkg-config curl cmake patch git unzip clang llvm libelf-dev kmod linux-base libssl-dev bison flex libncurses-dev bc zlib1g-dev libpcap-dev && \
    apt-get install -y --no-install-recommends bsdtar

# Prepare kernel
ARG kernel_version=5.1
RUN cd /usr/src && \
    wget -c https://cdn.kernel.org/pub/linux/kernel/v5.x/linux-${kernel_version}.tar.xz && \
    bsdtar xJf linux-${kernel_version}.tar.xz && \
    cd linux-${kernel_version} && \
    make defconfig && \
    make modules_prepare

# -------------------- BUILDER -----------------------
FROM base-layer AS builder

WORKDIR /go
ARG nff_version=fix-pci-v0.8.0
RUN git clone -b $nff_version --single-branch --depth=1 https://github.com/glasnostic/nff-go/
WORKDIR /go/nff-go
RUN export PATH="$PATH:$GOPATH"/bin
# needed by new dpdk building for lua5.3.pc
RUN apt-get install -y liblua5.3-dev libmnl-dev libibverbs-dev libnuma-dev
# nff-go is using git submodule to fetch dpdk and pktgen-dpdk code
RUN git submodule init
RUN	git submodule update
# Disable mlx driver for the time being
# "make all" will do "make nff-go-base dpdk test examples", and we don't need the test and examples
ARG NFF_GO_NO_MLX_DRIVERS=1
# Disable mlx driver for the time being
RUN sed -i 26d dpdk/Makefile
# "make all" will do "make nff-go-base dpdk test examples", and we don't need the test and examples
RUN make RTE_KERNELDIR=/usr/src/linux-5.1 -C nff-go-base
RUN make RTE_KERNELDIR=/usr/src/linux-5.1 -C dpdk

# build folder
RUN mkdir -p /go/router
WORKDIR /go/router
# copy codes
COPY . .
ENV CGO_CFLAGS_ALLOW='(-mno-fsgsbase|-mno-f16c|-include|rte_config.h)'
RUN make build

# --------------------- FINAL LAYER -----------------------
FROM golang:1.12 AS final-layer
LABEL maintainer="guesslin@glasnostic.com"

RUN apt-get update && \
    apt-get install -y libpcap0.8 ethtool iptables && \
    apt-get autoclean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /go/bin/
COPY --from=builder /go/router/router /go/bin/
COPY ./docker-entrypoint.sh /go/bin/

ENTRYPOINT ["/go/bin/docker-entrypoint.sh"]

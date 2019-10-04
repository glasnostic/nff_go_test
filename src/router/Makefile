CURRENT_DIR    = $(shell pwd)
NFFGO_ROOT     = $(abspath $(CURRENT_DIR)/../nff-go)
BUILD_TAGS = -tags nff

ifneq (,$(wildcard $(NFFGO_ROOT)/mk/include.mk))
include $(NFFGO_ROOT)/mk/include.mk
# Copy LDFLAGS from the source file: low.go
LDFLAGS = $(shell cat $(NFFGO_ROOT)/low/low.go | grep LDFLAGS | sed 's/^\#cgo LDFLAGS: //')
LDFLAGS := -L$(RTE_SDK)/$(RTE_TARGET)/lib $(LDFLAGS)
# Filter the MLX stuff
LDFLAGS := $(filter-out -lmlx4, $(LDFLAGS))
LDFLAGS := $(filter-out -lmlx5, $(LDFLAGS))
LDFLAGS := $(filter-out -lrte_pmd_mlx4, $(LDFLAGS))
LDFLAGS := $(filter-out -lrte_pmd_mlx5, $(LDFLAGS))
endif

.PHONY: cleanup-cgo-flags
cleanup-cgo-flags:
	sed -i "/^\#cgo/d" $(NFFGO_ROOT)/low/low.go

# XXX: Because we have multiple sub-packages using cgo, we cannot just:
#   1. overwrite the environment CGO_LDFLAGS
#   2. Set #cgo LDFLAGS in main package or in nff/runner_linux.go
# Unfortunatly intel-go/nff-go uses Makefile, which overwrite the environment
# CGO_FLAGS to run their application, which we cannot do as described above, we
# have to inject the CFLAGS and LDFLAGS to their source code: nff-go/low/log.go
.PHONY: prepare-cgo-flags
prepare-cgo-flags: cleanup-cgo-flags
	sed -i "s#\(\#include\ \"low\.h\"\)#\#cgo CFLAGS: $(CFLAGS)\n\#cgo LDFLAGS: $(LDFLAGS)\n\1#g" $(PROJECT_ROOT)/low/low.go

.PHONY: build
build: prepare-cgo-flags
	go generate -v $(BUILD_TAGS) ./...
	go build -o router $(BUILD_TAGS)
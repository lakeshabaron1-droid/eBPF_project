set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log()   { echo -e "${GREEN}[✓]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
err()   { echo -e "${RED}[✗]${NC} $1"; }
info()  { echo -e "${CYAN}[i]${NC} $1"; }

if [[ $EUID -ne 0 ]]; then
    err "This script must be run as root (sudo)."
    exit 1
fi

echo ""
echo "------------------------------------------------------------"
echo "  eBPF Gateway  VM Setup for Kali Linux"
echo "------------------------------------------------------------"
echo ""

info "Updating package lists..."
apt-get update -qq


info "Installing build essentials, clang, llvm..."
apt-get install -y -qq \
    build-essential \
    clang \
    llvm \
    pkg-config \
    > /dev/null 2>&1
log "Build tools installed"


info "Installing eBPF development libraries..."
apt-get install -y -qq \
    libbpf-dev \
    libelf-dev \
    dwarves \
    > /dev/null 2>&1
log "eBPF libraries installed"


KVER=$(uname -r)
info "Installing kernel headers for ${KVER}..."
apt-get install -y -qq \
    linux-headers-${KVER} \
    > /dev/null 2>&1 || warn "Kernel headers package not found — may already be installed"
log "Kernel headers ready"

info "Installing bpftool..."
apt-get install -y -qq bpftool > /dev/null 2>&1 || \
    apt-get install -y -qq linux-tools-common > /dev/null 2>&1 || \
    warn "bpftool not found in repos — you may need to build from source"
log "bpftool installed"


if command -v go &> /dev/null; then
    GO_VER=$(go version | awk '{print $3}')
    log "Go already installed: ${GO_VER}"
else
    info "Installing Go..."
    apt-get install -y -qq golang-go > /dev/null 2>&1
    log "Go installed: $(go version | awk '{print $3}')"
fi


if command -v node &> /dev/null; then
    NODE_VER=$(node --version)
    log "Node.js already installed: ${NODE_VER}"
else
    info "Installing Node.js 20.x..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - > /dev/null 2>&1
    apt-get install -y -qq nodejs > /dev/null 2>&1
    log "Node.js installed: $(node --version)"
fi


echo ""
info "Verifying eBPF / BTF support..."

if [[ -f /sys/kernel/btf/vmlinux ]]; then
    log "BTF support: /sys/kernel/btf/vmlinux exists"
else
    err "BTF support: /sys/kernel/btf/vmlinux NOT FOUND"
    warn "Your kernel may not have CONFIG_DEBUG_INFO_BTF=y"
    warn "Check: grep CONFIG_DEBUG_INFO_BTF /boot/config-$(uname -r)"
fi

# Check CONFIG_DEBUG_INFO_BTF
BOOT_CONFIG="/boot/config-$(uname -r)"
if [[ -f "${BOOT_CONFIG}" ]]; then
    BTF_CONFIG=$(grep -c "CONFIG_DEBUG_INFO_BTF=y" "${BOOT_CONFIG}" || true)
    if [[ "${BTF_CONFIG}" -ge 1 ]]; then
        log "Kernel config: CONFIG_DEBUG_INFO_BTF=y"
    else
        warn "Kernel config: CONFIG_DEBUG_INFO_BTF not set to y"
    fi
else
    warn "Boot config not found at ${BOOT_CONFIG}"
fi


echo "------------------------------------------------------------"
echo "  Setup Summary"
echo "------------------------------------------------------------"
echo ""
info "Kernel:    $(uname -r)"
info "Clang:     $(clang --version | head -1)"
info "Go:        $(go version | awk '{print $3}')"
info "Node:      $(node --version 2>/dev/null || echo 'not installed')"
info "bpftool:   $(bpftool version 2>/dev/null | head -1 || echo 'not found')"
echo ""

log "Setup complete! Next steps:"
echo "  1. Run: make vmlinux        (generate vmlinux.h)"
echo "  2. Run: make generate       (compile eBPF → Go bindings)"
echo "  3. Run: make build          (build gateway binary)"
echo "  4. Run: make dashboard      (install dashboard deps)"
echo ""

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "${SCRIPT_DIR}")"
OUTPUT="${PROJECT_ROOT}/bpf/headers/vmlinux.h"

# Check BTF availability
if [[ ! -f /sys/kernel/btf/vmlinux ]]; then
    echo "ERROR: /sys/kernel/btf/vmlinux not found."
    echo "Your kernel does not have BTF enabled."
    echo "Check: grep CONFIG_DEBUG_INFO_BTF /boot/config-$(uname -r)"
    exit 1
fi

# Check bpftool
if ! command -v bpftool &> /dev/null; then
    echo "ERROR: bpftool not found. Install with:"
    echo "  sudo apt install bpftool"
    exit 1
fi

# Generate
echo "Generating vmlinux.h from /sys/kernel/btf/vmlinux..."
mkdir -p "$(dirname "${OUTPUT}")"
bpftool btf dump file /sys/kernel/btf/vmlinux format c > "${OUTPUT}"

LINES=$(wc -l < "${OUTPUT}")
echo "Generated: ${OUTPUT} (${LINES} lines)"

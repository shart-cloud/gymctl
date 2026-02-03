#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check Docker socket access
check_docker() {
    if [ -S /var/run/docker.sock ]; then
        if docker version > /dev/null 2>&1; then
            print_info "Docker socket accessible"
        else
            print_warn "Docker socket mounted but not accessible. Some exercises may not work."
            print_warn "Try running with: docker run --group-add $(stat -c '%g' /var/run/docker.sock) ..."
        fi
    else
        print_warn "Docker socket not mounted. Docker exercises will not work."
        print_warn "To enable Docker exercises, run with: -v /var/run/docker.sock:/var/run/docker.sock"
    fi
}

# Initialize gym directory if needed
init_gym() {
    if [ ! -f "$HOME/.gym/progress.yaml" ]; then
        print_info "First run detected. Initializing gym environment..."
        mkdir -p "$HOME/.gym/workdir"
        echo "version: 1" > "$HOME/.gym/progress.yaml"
        echo "exercises: {}" >> "$HOME/.gym/progress.yaml"
    fi
}

# Main entrypoint logic
main() {
    print_info "GymCTL Container Environment"
    echo "============================="

    # Initialize environment
    init_gym
    check_docker

    echo ""
    print_info "Available commands:"
    echo "  gymctl list              - List all exercises"
    echo "  gymctl start <exercise>  - Start an exercise"
    echo "  gymctl check             - Check your solution"
    echo "  gymctl hint              - Get a hint"
    echo "  gymctl stop              - Stop current exercise"
    echo "  gymctl reset             - Reset current exercise"
    echo ""

    # If arguments provided, run gymctl with them
    if [ $# -gt 0 ]; then
        if [ "$1" = "bash" ] || [ "$1" = "sh" ]; then
            exec "$@"
        else
            exec gymctl "$@"
        fi
    else
        # Interactive mode
        print_info "Starting interactive shell. Type 'exit' to quit."
        exec /bin/bash
    fi
}

# Handle signals
trap 'print_info "Shutting down..."; exit 0' SIGTERM SIGINT

# Run main function
main "$@"
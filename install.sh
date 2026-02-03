#!/bin/bash
# Installation script for gymctl

set -e

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
TASKS_DIR="${TASKS_DIR:-$HOME/.gym/tasks}"
REPO_URL="https://github.com/shart/container-course"

echo "Installing gymctl..."

# Create directories
mkdir -p "$INSTALL_DIR"
mkdir -p "$(dirname "$TASKS_DIR")"

# Build or download gymctl binary
if [ -f "./main.go" ]; then
    echo "Building gymctl from source..."
    go build -o "$INSTALL_DIR/gymctl" .
else
    echo "Downloading gymctl binary..."
    # In a real setup, download from GitHub releases
    # curl -L https://github.com/shart/container-course/releases/latest/download/gymctl -o "$INSTALL_DIR/gymctl"
    echo "Error: Binary download not yet implemented. Please build from source."
    exit 1
fi

# Make it executable
chmod +x "$INSTALL_DIR/gymctl"

# Download exercise files
if [ ! -d "$TASKS_DIR" ]; then
    echo "Downloading exercise files to $TASKS_DIR..."
    git clone "$REPO_URL" /tmp/container-course
    mv /tmp/container-course/containers/gymctl/tasks "$TASKS_DIR"
    rm -rf /tmp/container-course
else
    echo "Exercise files already exist at $TASKS_DIR"
fi

# Add to PATH if needed
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "Add the following to your shell configuration file (.bashrc, .zshrc, etc.):"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi

echo ""
echo "Installation complete!"
echo ""
echo "Exercises installed at: $TASKS_DIR"
echo "Binary installed at: $INSTALL_DIR/gymctl"
echo ""
echo "To get started:"
echo "  gymctl list                    # List available exercises"
echo "  gymctl start <exercise-name>   # Start an exercise"
echo "  gymctl check                   # Check your solution"
Output demo-quickstart.gif
Require ./gymctl

Set FontSize 14
Set Width 1200
Set Height 600
Set Shell "bash"
Set TypingSpeed 50ms

Type "# Jerry's Gym - Quick Start Demo"
Enter
Sleep 1s

Type "# List available exercises"
Enter
Type "./gymctl list"
Sleep 500ms
Enter
Sleep 3s

Type "# Filter by track"
Enter
Type "./gymctl list --track docker-fundamentals"
Sleep 500ms
Enter
Sleep 2s

Type "# Start an exercise"
Enter
Type "./gymctl start jerry-broken-layers"
Sleep 500ms
Enter
Sleep 4s

Type "# Check solution (will fail initially)"
Enter
Type "./gymctl check"
Sleep 500ms
Enter
Sleep 3s

Type "# Get a hint"
Enter
Type "./gymctl hint"
Sleep 500ms
Enter
Sleep 3s

Type "# Show overall progress"
Enter
Type "./gymctl status"
Sleep 500ms
Enter
Sleep 3s

Type "clear"
Enter
Sleep 500ms
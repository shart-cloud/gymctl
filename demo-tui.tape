Output demo-tui.gif
Require ./gymctl

Set FontSize 14
Set Width 1200
Set Height 600
Set Shell "bash"
Set TypingSpeed 50ms

Type "# Jerry's Gym - Interactive TUI Demo"
Enter
Sleep 1s

Type "# Launch interactive exercise browser"
Enter
Type "./gymctl shell"
Sleep 500ms
Enter
Sleep 2s

# Simulate navigation in TUI
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Up
Sleep 500ms

# Select an exercise
Enter
Sleep 2s

# Exit TUI
Escape
Sleep 1s

Type "clear"
Enter
Sleep 500ms
Output demo-workflow.gif
Require ./gymctl

Set FontSize 14
Set Width 1200
Set Height 600
Set Shell "bash"
Set TypingSpeed 50ms

Type "# Jerry's Gym - Complete Workflow Demo"
Enter
Sleep 1s

Type "# Check prerequisites"
Enter
Type "./gymctl diagnose"
Sleep 500ms
Enter
Sleep 3s

Type "# Browse exercises by difficulty"
Enter
Type "./gymctl list --difficulty beginner"
Sleep 500ms
Enter
Sleep 2s

Type "# Get exercise info before starting"
Enter
Type "./gymctl info jerry-broken-service"
Sleep 500ms
Enter
Sleep 3s

Type "# Start the exercise"
Enter
Type "./gymctl start jerry-broken-service"
Sleep 500ms
Enter
Sleep 4s

Type "# Use watch mode for continuous checking"
Enter
Type "./gymctl watch &"
Sleep 500ms
Enter
Sleep 2s

Type "# View current status"
Enter
Type "./gymctl status"
Sleep 500ms
Enter
Sleep 3s

Type "clear"
Enter
Sleep 500ms
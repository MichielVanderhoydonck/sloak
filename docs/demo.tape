# Output configuration
Output docs/demo.gif

# Terminal appearance
Set FontSize 15
Set Width 850
Set Height 480
Set Padding 18
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set TypingSpeed 40ms

# Setup path and clear screen
Hide
Type "export PATH=$PWD:$PATH"
Enter
Type "clear"
Enter
Show

# 1. Error Budget Calculation
Type "sloak calculate errorbudget --slo 99.9 --window 30d"
Sleep 500ms
Enter
Sleep 3.5s

# 2. Burn Rate Triage
Type "sloak calculate burnrate --slo 99.9 --window 30d --elapsed 7d --consumed 30m"
Sleep 500ms
Enter
Sleep 4.5s

# 3. MWMBR Alerting Table
Type "sloak generate alert-table --slo 99.9"
Sleep 500ms
Enter
Sleep 5s

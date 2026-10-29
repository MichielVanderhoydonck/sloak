# Output configuration
Output docs/demo.gif

# Terminal appearance
Set FontSize 14
Set Width 920
Set Height 520
Set Padding 18
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set TypingSpeed 35ms

# Setup path and clear screen
Hide
Type "export PATH=$PWD:$PATH"
Enter
Type "clear"
Enter
Show

# 1. Time & Request-Based Error Budget
Type "sloak calculate errorbudget --slo 99.9 --window 30d --rps 100"
Sleep 400ms
Enter
Sleep 3s

# 2. Incident Burn Rate Triage (Request-Based)
Type "sloak calculate burnrate --slo 99.9 --window 30d --elapsed 7d --rps 100 --failed-requests 180000"
Sleep 400ms
Enter
Sleep 4s

Hide
Type "clear"
Enter
Show

# 3. Multi-Tier Dependency Topology & Bottleneck Detection
Type `sloak calculate dependency --components "api=99.99,db1=99.9|db2=99.9,auth=99.9" --slo 99.9`
Sleep 400ms
Enter
Sleep 3.5s

# 4. Google SRE Multi-Window Alerting Matrix
Type "sloak generate alert-table --slo 99.9 --window 30d"
Sleep 400ms
Enter
Sleep 3.5s

Hide
Type "clear"
Enter
Show

# 5. Turnkey Prometheus Operator Alert & Recording Rules
Type `sloak render --template prometheus-operator --slo 99.9 --set metric_name=api_errors --set errors_query='http_requests_total{code=~"5.."}' --set total_query='http_requests_total' | head -n 20`
Sleep 400ms
Enter
Sleep 4.5s

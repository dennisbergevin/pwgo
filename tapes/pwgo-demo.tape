Output ./assets/pwgo-demo.gif

Set WindowBar Colorful
Set WindowBarSize 70
Set Theme "nord"
Set Shell "bash"
Set FontSize 20
Set Width 1400
Set Height 1000


Type "pwgo"
Enter
Wait+Screen /items/
Sleep 1s
Type "/"
Sleep 1s
Type "Persistence"
Sleep 400ms
Enter
Sleep 1s
Space
Sleep 1s
Type "Shift+L"
Sleep 1s
Type "Shift+L"
Sleep 1s
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Space
Sleep 1s
Type "Shift+L"
Sleep 1s

Hide
Ctrl+c
Sleep 1s
Type "clear"
Enter

Show

Type "pwgo --help"
Sleep 1s
Enter
Sleep 3s

Hide
Ctrl+c
Sleep 1s
Type "clear"
Enter

Show

Type "pwgo --config playwright.dev.config.ts"
Sleep 1s
Enter
Wait+Screen /items/
Sleep 1s
Type "Shift+L"
Sleep 1s
Type "Shift+L"
Sleep 1s

Hide
Ctrl+c
Sleep 1s
Type "clear"
Enter

Show

Type "pwgo --project webkit"
Sleep 1s
Enter
Sleep 1s
Type "Shift+L"
Sleep 1s
Type "Shift+L"
Sleep 1s

Hide
Ctrl+c
Sleep 1s
Type "clear"
Enter

Show

Type "pwgo --last-failed"
Sleep 1s
Enter
Sleep 1s
Type "Shift+L"
Sleep 1s
Type "Shift+L"
Sleep 1s

Hide
Ctrl+c
Sleep 1s
Type "clear"
Enter

Show 

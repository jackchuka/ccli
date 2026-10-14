# Regenerate with: vhs assets/demo.tape (from the repository root)
# The demo runs against a fabricated Claude Code installation built by assets/demo/setup.sh.

Output assets/demo.gif

Set Shell "bash"
Set FontSize 22
Set Width 1440
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

Hide
Type `export D=/tmp/ccli-demo && bash assets/demo/setup.sh "$D" && go build -o "$D/bin/ccli" . && export HOME="$D/home" PATH="$D/bin:$PATH" PS1='\[\e[1;35m\]$ \[\e[0m\]' && unset ANTHROPIC_MODEL && cd "$HOME/code/acme-api" && clear`
Enter
Sleep 4s
Show

Type "ccli info"
Sleep 400ms
Enter
Sleep 4s

Type "clear"
Enter
Type "ccli mcp list"
Sleep 400ms
Enter
Sleep 2s
Type "ccli skills list"
Sleep 400ms
Enter
Sleep 4s

Type "clear"
Enter
Type "ccli memory list"
Sleep 400ms
Enter
Sleep 4s

Type "clear"
Enter
Type "ccli projects list"
Sleep 400ms
Enter
Sleep 2s
Type "ccli projects clean --older-than 30d --dry-run"
Sleep 400ms
Enter
Sleep 4s

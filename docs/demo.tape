# vhs tape for the clipsh README demo.
#
# Run:
#   vhs docs/demo.tape
#
# Produces docs/demo.gif — a scripted, deterministic tour of the common
# invocations. No real SSH target is required; every command runs with
# --dry-run so the output is the "would upload" plan instead of an
# actual network round-trip.

Output docs/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 960
Set Height 540
Set Padding 16
Set Theme "Dracula"
Set TypingSpeed 40ms

# Seed the text clipboard for step 2.
Hide
Type `printf 'Hello from the clipboard' | pbcopy`
Enter
Show

# 1. Version
Type "clipsh --version"
Sleep 400ms
Enter
Sleep 900ms

# 2. Dry-run with text clipboard
Type "clipsh -n user@dev.example.com"
Sleep 400ms
Enter
Sleep 1200ms

# 3. Template with placeholders, ad-hoc hook
Type "clipsh -n -r '/uploads/{hostname}-{random}.{ext}' --hook tmux:main user@dev"
Sleep 400ms
Enter
Sleep 1400ms

# 4. Explicit file argument — filename + extension are preserved
Type "clipsh -n user@dev.example.com ./README.md"
Sleep 400ms
Enter
Sleep 1400ms

Sleep 800ms

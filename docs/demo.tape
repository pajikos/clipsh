# vhs tape for the clipsh README demo.
#
# Run:
#   vhs docs/demo.tape
#
# Produces docs/demo.gif. To rebuild, you need:
#   - a reachable SSH alias named dev.local (any host that accepts a
#     Cmd+C-style upload is fine; the default /tmp path will be used)
#   - vhs + ttyd installed
#
# The tape isolates XDG_CONFIG_HOME so your real ~/.config/clipsh/config.toml
# (profile, hook, template) has no effect on the recording.

Output docs/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 960
Set Height 540
Set Padding 16
Set Theme "Dracula"
Set TypingSpeed 40ms

Env XDG_CONFIG_HOME "/tmp/clipsh-vhs-demo"

# Isolate config, seed a text clipboard, clear the screen.
Hide
Type `rm -rf /tmp/clipsh-vhs-demo && mkdir -p /tmp/clipsh-vhs-demo`
Enter
Type `printf 'Hello from clipsh!' | pbcopy`
Enter
Type "clear"
Enter
Show

# 1. Simplest possible invocation: send whatever is on the clipboard.
Type "clipsh dev.local"
Sleep 500ms
Enter
Sleep 2200ms

# 2. Upload a file by name — original name and extension are preserved
# via the {basename}.{ext} default-path behavior.
Type "clipsh -r '/tmp/{basename}.{ext}' dev.local ./README.md"
Sleep 500ms
Enter
Sleep 2500ms

Sleep 800ms

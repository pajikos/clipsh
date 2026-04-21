# Examples

## Send a screenshot to a remote terminal app

```sh
clipsh -p 2222 \
  -o StrictHostKeyChecking=no \
  -r '/home/me/.clipboard.{ext}' \
  me@dev.example.com
```

Then in the remote terminal: type `/image` (or whatever the app expects)
and paste the path from your clipboard. For a fully automated flow, pair
this with a `tmux:` hook — see [Configuration](config.md).

## Send a file by name

```sh
clipsh user@box ./design.pdf
```

The file extension drives `{ext}` in the remote path, so the upload lands at
`/tmp/clipsh-<epoch>.pdf` by default.

## Use an SSH config alias

```sshconfig
# ~/.ssh/config
Host mybox
  HostName box.example.com
  User deploy
  Port 2222
  ProxyJump bastion
```

```sh
clipsh mybox ./payload.tar.gz
```

## Dry-run before committing to an upload

```sh
clipsh -n -r '/uploads/{hostname}-{random}.{ext}' deploy@srv
# would upload 2481 bytes (png) to deploy@srv:/uploads/mymac-a1b2c3d4.png
```

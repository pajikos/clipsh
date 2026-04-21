# Examples

## Send a screenshot to Claude Code in a remote container

```sh
clipsh -p 2222 \
  -o StrictHostKeyChecking=no \
  -r '/home/vscode/repos/chargee/.clipboard.{ext}' \
  vscode@container.local
```

Then in the remote Claude Code session: type `/image` and paste the path from
your clipboard, or rely on a named profile + hook to have it typed for you
(see [Configuration](config.md)).

## Send a file by name

```sh
clipsh user@box ./design.pdf
```

The file extension drives `{ext}` in the remote path, so the upload lands at
`/tmp/clipsh-<epoch>.pdf` by default.

## Use an SSH config alias

```sh
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

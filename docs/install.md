# Install

bafi is a single program without dependencies. Download it, unpack it, run it.

## Download

Get the archive for your system from the [latest release](https://github.com/mmalcek/bafi/releases/latest):

| System | Archive |
|---|---|
| Windows (64-bit) | `bafi_vX.Y.Z_windows_amd64.zip` |
| macOS, Apple silicon (M1 and newer) | `bafi_vX.Y.Z_darwin_arm64.tgz` |
| macOS, Intel | `bafi_vX.Y.Z_darwin_amd64.tgz` |
| Linux, x86-64 | `bafi_vX.Y.Z_linux_amd64.tgz` |
| Linux, ARM 64-bit (Raspberry Pi 4/5, Graviton) | `bafi_vX.Y.Z_linux_arm64.tgz` |
| Linux, 32-bit x86 | `bafi_vX.Y.Z_linux_386.tgz` |

Each archive contains the `bafi` program, this documentation, the sample `testdata.xml` and
`template.tmpl`, and the example Lua functions in `lua/`.

### Verify the download

`SHA256SUMS.txt` in the release lists the checksum of every archive:

```sh
sha256sum -c SHA256SUMS.txt --ignore-missing      # Linux
shasum -a 256 -c SHA256SUMS.txt --ignore-missing  # macOS
```

On Windows: `Get-FileHash .\bafi_vX.Y.Z_windows_amd64.zip` and compare with the line in `SHA256SUMS.txt`.

## Unpack and run

### Linux

```sh
tar xzf bafi_vX.Y.Z_linux_amd64.tgz
./bafi -v
sudo mv bafi /usr/local/bin/      # optional: run it from anywhere
```

### macOS

```sh
tar xzf bafi_vX.Y.Z_darwin_arm64.tgz
xattr -d com.apple.quarantine ./bafi   # only needed when the archive was downloaded with a browser
./bafi -v
```

macOS refuses to open programs downloaded with a browser unless an Apple-registered developer signed
them; removing the quarantine flag once fixes that.

### Windows

Unzip the archive, open PowerShell in that folder and run:

```powershell
.\bafi.exe -v
```

PowerShell needs the `.\` prefix for programs in the current folder. To run `bafi` from anywhere, move
`bafi.exe` to a folder listed in the `PATH` environment variable. SmartScreen may warn the first time:
choose **More info**, then **Run anyway**.

## With Go

If you have Go 1.27.3 or newer:

```sh
go install github.com/mmalcek/bafi@latest
```

The program lands in `$(go env GOPATH)/bin`.

## First steps

```sh
bafi -i testdata.xml -t template.tmpl
```

Then pick a task from the [recipes](recipes/index.md).

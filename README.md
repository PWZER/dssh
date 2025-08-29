# DSSH

## Install Dependent libraries

```bash
# MacOS
brew install upx

# Linux
sudo apt install -y upx
```

## Build

```bash
./build.sh

# MacOS
cp bin/dssh-darwin-amd64 /usr/local/bin/ds

# Linux
cp bin/dssh-linux-amd64 /usr/local/bin/ds
```

## Usage

```bash
ds --help

# output
A command-line tools for ssh

Usage:
  ds {host}... [flags]
  ds [command]

Available Commands:
  completion  Generate completion script
  fix         fix ssh agent forward
  get         download files from remote host
  help        Help about any command
  host        host configs manage
  json        json tools.
  passwd      password generator
  put         upload local files to remote host
  server      simple file server

Flags:
  -c, --command string    remote run command
      --config string     config file (default is $HOME/.dssh.yaml)
  -f, --force             force run when failed
      --get-dest string   download local dest path
      --get-src string    download remote src path
  -h, --help              help for ds
      --host string       host name or remove host addr
  -j, --jump string       ssh jump proxy
  -m, --module string     remote run module
      --parallel int      max parallel run tasks num (default 1)
  -p, --port uint16       remote host port
      --put-dest string   upload remote dest path
      --put-src string    upload local src path
  -s, --script string     remote run script
  -t, --tags string       tags filter
  -u, --user string       username
  -v, --version           version for ds

Use "ds [command] --help" for more information about a command.
```

## Configuration

Default use `~/.dssh.yaml`.

```bash
modulesDir: ""

sshAuthSock: /root/.ssh/ssh_auth_sock
```

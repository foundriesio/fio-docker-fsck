# fio-docker-fsck
A tool to detect and fix a docker's image&amp;layer store issues

## How to use

### Build
```
make
```

### Run
```
./bin/fio-docker-fsck [<data-root>] [-fix-store]
```
`<data-root>` - refers to `/var/lib/docker` by default

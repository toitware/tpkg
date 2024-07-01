# Toit package management server

A docker container that exposes a package registry for the Toit package manager.

Features:
- List packages.
- Search.
- Automatic Toitdoc generation.
- API to register new packages or new versions of existing packages.

## Deployment

Run the docker container with the following variables:
- `REGISTRY_BRANCH` specifies the branch of the registry (for example `main`).
- `REGISTRY_URL` specifies the url of the registry to serve packages from (for
  example `github.com/toitware/test-registry`).
- `REGISTRY_SSH_KEY_FILE` specifies the SSH key file inside the docker container
  that grants read/write access to the registry. For example `/secrets/ssh-key`.
- `REGISTRY_SSH_KEY` is an alternative way to provide the SSH key as a string.
  The `REGISTRY_SSH_KEY_FILE` path must still be set, and will be populated with
  the content of the `REGISTYR_SSH_KEY` variable if the `REGISTRY_SSH_KEY_FILE`
  doesn't exist.

Use `REGISTRY_SSH_KEY_FILE` if you want to provide the key through a mounted volume.
Use `REGISTRY_SSH_KEY` if you want to provide the key as an environment variable.

The default port is `8733`. You can change it by setting the `PORT` environment variable.

## Example

Using a volume for the SSH key:

```shell
docker run -p 8733:8733 \
  -e REGISTRY_BRANCH=main \
  -e REGISTRY_URL=github.com/toitware/test-registry \
  -e REGISTRY_SSH_KEY_FILE=/secrets/ssh-key \
  -v /path/to/ssh-key:/secrets/ssh-key \
  toit-registry
```

Using an environment variable for the SSH key:

```shell
docker run -p 8733:8733 \
  -e REGISTRY_BRANCH=main \
  -e REGISTRY_URL=github.com/toitware/test-registry \
  -e REGISTRY_SSH_KEY="$(cat /path/to/ssh-key)" \
  toit-registry
```

## API

### Packages

List all packages:
```
$ curl 127.0.0.1:8733/api/v1/packages
{"result":{"package":{"name":"location","description":"Support for locations in a geographical coordinate system.","license":"MIT","url":"github.com/toitware/toit-location","latestVersion":"1.0.0"}}}
{"result":{"package":{"name":"morse","description":"Functions for International (ITU) Morse code.","license":"MIT","url":"github.com/toitware/toit-morse","latestVersion":"1.0.2"}}}
{"result":{"package":{"name":"morse_tutorial","description":"A tutorial version of the Morse package.","license":"MIT","url":"github.com/toitware/toit-morse-tutorial","latestVersion":"1.0.0"}}}
```

### Versions of a package

List all versions of a package:
```
$ curl 127.0.0.1:8733/api/v1/packages/github.com/toitware/toit-morse/versions
{"result":{"version":{"name":"morse","description":"Functions for International (ITU) Morse code.","license":"MIT","url":"github.com/toitware/toit-morse","version":"1.0.0","dependencies":[]}}}
{"result":{"version":{"name":"morse","description":"Functions for International (ITU) Morse code.","license":"MIT","url":"github.com/toitware/toit-morse","version":"1.0.1","dependencies":[]}}}
{"result":{"version":{"name":"morse","description":"Functions for International (ITU) Morse code.","license":"MIT","url":"github.com/toitware/toit-morse","version":"1.0.2","dependencies":[]}}}
```

### Sync the registry

Sync the registry:
```
$ curl -X POST 127.0.0.1:8733/api/v1/sync
{}
```

### Register a package

Register a package:
```
$ curl -X POST 127.0.0.1:8733/api/v1/register/github.com/toitware/ubx-message/version/2.1.1
{}
```

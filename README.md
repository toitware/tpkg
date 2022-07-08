# Toit Package Manager

Package server
==============
Run the package server with
```
REGISTRY_BRANCH=<branch> REGISTRY_URL=<registry-url> REGISTRY_SSH_KEY_FILE=<ssh-key-path> make run/registry
```
Environment variable explanation:
 - `REGISTRY_BRANCH` specifies the branch of the registry (e.g. `master`).
 - `REGISTRY_URL`specifies the url of the registry to serve packages from (e.g. `github.com/toitware/test-registry`).
 - `REGISTRY_SSH_KEY_FILE` specifies the SSH key file that grants read/write access to the registry (e.g. `/home/lau/toitware/toit/tools/tpkg/test-registry_deploy_key`).

By default the server runs on port `8733` (set environment variable `PORT` to change this).

Example uses:
Packages
```
$ curl 127.0.0.1:8733/api/v1/packages
{"result":{"package":{"name":"location","description":"Support for locations in a geographical coordinate system.","license":"MIT","url":"github.com/toitware/toit-location","latestVersion":"1.0.0"}}}
{"result":{"package":{"name":"morse","description":"Functions for International (ITU) Morse code.","license":"MIT","url":"github.com/toitware/toit-morse","latestVersion":"1.0.2"}}}
{"result":{"package":{"name":"morse_tutorial","description":"A tutorial version of the Morse package.","license":"MIT","url":"github.com/toitware/toit-morse-tutorial","latestVersion":"1.0.0"}}}
```
Versions of a package
```
$ curl 127.0.0.1:8733/api/v1/packages/github.com/toitware/toit-morse/versions
{"result":{"version":{"name":"morse","description":"Functions for International (ITU) Morse code.","license":"MIT","url":"github.com/toitware/toit-morse","version":"1.0.0","dependencies":[]}}}
{"result":{"version":{"name":"morse","description":"Functions for International (ITU) Morse code.","license":"MIT","url":"github.com/toitware/toit-morse","version":"1.0.1","dependencies":[]}}}
{"result":{"version":{"name":"morse","description":"Functions for International (ITU) Morse code.","license":"MIT","url":"github.com/toitware/toit-morse","version":"1.0.2","dependencies":[]}}}
```
Sync registry
```
$ curl -X POST 127.0.0.1:8733/api/v1/sync
{}
```
Register package
```
$ curl -X POST 127.0.0.1:8733/api/v1/register/github.com/toitware/ubx-message/version/2.1.1
{}
```

Tips and Tricks
===============

## Auto-update submodules

Run

```
$ git config core.hooksPath .githooks
```

in the repository to auto-update sub modules.

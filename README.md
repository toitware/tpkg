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
 - `REGISTRY_SSH_KEY` is an alternative way to provide the SSH key as a string. The `REGISTRY_SSH_KEY_FILE` path
    must still be set, and will be populated with the content of the `REGISTRY_SSH_KEY` variable if the
    `REGISTRY_SSH_KEY_FILE` doesn't exist.

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

For testing the image
=====================
* Create a fresh repository or fork the existing registry repository.
* Create a new ssh-key:
  ``` shell
  ssh-keygen -t ed25519
  ```
* Copy the public key into the deploy keys on Github: https://github.com/XXX/YYY/settings/keys.
  Don't forget to add write-acces.
* Create a known_hosts file:
  ``` shell
  ssh-keyscan github.com > known_hosts
  ```
* Start docker if not already running `sudo systemctl start docker`
* Run `make image`
* Start the image:
  ``` shell
  docker run -p 8733:8733 -e"REGISTRY_URL=github.com/XXX/YYY" toit-registry
  ```


Tips and Tricks
===============

## Auto-update submodules

Run

```
$ git config core.hooksPath .githooks
```

in the repository to auto-update sub modules.

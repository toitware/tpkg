# Toit package manager - development

## Testing the image

* Create a fresh repository or fork the existing registry repository.
* Create a new ssh-key:
  ``` shell
  ssh-keygen -t ed25519
  ```
* Copy the public key into the deploy keys on GitHub: https://github.com/XXX/YYY/settings/keys.
  Don't forget to add write-access.
* Create a known_hosts file:
  ``` shell
  ssh-keyscan github.com > known_hosts
  ```
* Start docker if not already running `sudo systemctl start docker`
* Run `make image`. Make sure `$HOME/go/bin` is in your path.
* Start the image (see the main README):
  ``` shell
  docker run -p 8733:8733 -e"REGISTRY_URL=github.com/XXX/YYY" -e"..." toit_registry
  ```


## Tips and tricks

### Auto-update submodules

Run

```
$ git config core.hooksPath .githooks
```

in the repository to auto-update sub modules.

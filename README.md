# Fan-out work

This library aims to assist in managing fan-out work in a distributed environment.

## How it works

After authenticating to GitHub via OAuth, the user can select a patch to apply to a target organization.

* A number of PRs will be created/updated based on the chosen patch/target organization.
* Optionally, if a "fan-out" repo exists in the target organization, a tracking issue can be created with links to generated PRs.

## Demo

https://github.com/user-attachments/assets/e9e3bae1-dcdc-4b3c-a6a7-da61d874d857

## Deploying

In addition to the `fan-out-work` binary that starts the webserver, you'll need:

* `multi-gitter` available your `PATH`
* a `patches` directory in the runtime current working directory
  - patches exist as arbitrarily named folders, which must include:
    * a `config.yml` config file defining the branch name, PR title, and PR body
    * a `patch` executable run in the context of cloned repositories (see [multi-gitter run docs](https://github.com/lindell/multi-gitter?tab=readme-ov-file#-usage-of-run))
  - see `src/fan-out-work/patches/example` as an example patch

See the included `Dockerfile`...with the following caveats:

* you likely want to pin to a specific version of `multi-gitter`
* you will need to `COPY` your own patches

## Acknowledgements

The [multi-gitter project](https://github.com/lindell/multi-gitter) does a ton of the heavy-lifting here!

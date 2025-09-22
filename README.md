# Fan-out work

Web UI for managing fan-out work in a distributed environment. Centralized patch management; distributed application.

## How it works

After authenticating to GitHub via OAuth, select a patch to apply to a target organization.

* A number of PRs will be created/updated based on the chosen patch/target organization.
* Optionally, if a "fan-out" repo exists in the target organization, a tracking issue will be created.

## Demo

https://github.com/user-attachments/assets/1f59299e-5c51-43e7-b805-f5e43abcf506

## Deploying

In addition to the `fan-out-work` binary that starts the webserver, you'll need:

* `multi-gitter` available your `PATH`
* a `patches` directory in the runtime current working directory
  - patches exist as arbitrarily named folders, which must include:
    * a `config.yml` config file defining the branch name, PR title, and PR body
    * a `patch` executable run in the context of cloned repositories (see [multi-gitter run docs](https://github.com/lindell/multi-gitter?tab=readme-ov-file#-usage-of-run))
  - see `src/fan-out-work/patches/example` as an example patch
* environment variable configuration (see `.env.example`)

See the included `Dockerfile`...with the following caveats:

* you likely want to pin to a specific version of `multi-gitter`
* you will need to `COPY` your own patches

## Acknowledgements

The [multi-gitter project](https://github.com/lindell/multi-gitter) does a ton of the heavy-lifting here!

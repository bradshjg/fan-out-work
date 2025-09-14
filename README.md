# Fan-out work

This library aims to assist in managing fan-out work in a distributed environment.

## How it works

After authenticating to GitHub via OAuth, the user can select a patch to apply to a target organization.

* A number of PRs will be created/updated based on the chosen patch/target organization.
* An issue will be created/updated in the "fan-out-work" repository (will be created if it doesn't already exist) with links to the generated PRs to track their merge status.

## Acknowledgements

The [multi-gitter project](https://github.com/lindell/multi-gitter) does a ton of the heavy-lifting here!

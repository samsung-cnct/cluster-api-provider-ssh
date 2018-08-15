# Contributing guidelines

## Sign the CLA

Kubernetes projects require that you sign a Contributor License Agreement (CLA)
before we can accept your pull requests.  Please see
[git.k8s.io/commonuity/CLA.md](https://git.k8s.io/community/CLA.md) for more
info.

### Contributing A Patch

1. Submit an issue describing your proposed change to the repo in question.
1. The [repo owners](OWNERS) will respond to your issue promptly.
1. If your proposed change is accepted, and you haven't already done so, sign
   a Contributor License Agreement (see details above).
1. Fork the desired repo, develop and test your code changes.
1. Submit a pull request.

### Using Linters

You can run the linters against the code with

```bash
make check
```

This will check go, yaml, and markdown for linting errors if you have the
required applications installed:

- go: The go linter,
  [golangci-lint](https://github.com/golangci/golangci-lint), will install
  itself.
  Follow the link for its documentation.
- yaml: The yaml linter,
  [yamllint](https://github.com/adrienverge/yamllint) is packaged for most
  distros.
  Installation instructions and documentation can be found at
  [yamllint.readthedocs.io](https://yamllint.readthedocs.io/).
- markdown: The markdown linter,
  [markdownlint-cli](https://github.com/DavidAnson/markdownlint) needs node.js
  and npm installed to install it.
  See the link for installation instructions and documentation.

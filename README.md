# hardhat [![GoDoc][doc-img]][doc] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov]

Hardhat is a Go- and git-specific build tool that fits in the sweet spot
between the standard Go tool and full-featured build systems like Bazel, Buck,
and Pants.

## Installation

```
go get -u github.com/akshayjshah/hardhat
```

## Current Status

This is currently a skunkworks project &mdash; caveat emptor. If it ever
stabilizes, I'll start cutting semver-compliant releases.

Right now, it supports two useful commands: `status` and `test`. See the
output of `hardhat --help`, `hardhat status --help`, and `hardhat test --help`
for details.

[doc-img]: https://godoc.org/github.com/akshayjshah/hardhat?status.svg
[doc]: https://godoc.org/github.com/akshayjshah/hardhat
[ci-img]: https://travis-ci.org/akshayjshah/hardhat.svg?branch=master
[ci]: https://travis-ci.org/akshayjshah/hardhat
[cov-img]: https://codecov.io/gh/akshayjshah/hardhat/branch/master/graph/badge.svg
[cov]: https://codecov.io/gh/akshayjshah/hardhat

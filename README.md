<p align="center">
    <img src="https://avatars3.githubusercontent.com/u/23257050" width="100"/>
</p>
<h1 align="center">karma.link</h1>

karma.link is a cross-platform RPC server that allows 'classic' IT systems to interact with smart contracts on the [Ethereum](https://ethereum.org) blockchain network through JSON.

[![GoDoc](https://godoc.org/github.com/karmarun/karma.link?status.svg)](https://godoc.org/github.com/karmarun/karma.link)
[![Go Report Card](https://goreportcard.com/badge/github.com/karmarun/karma.link?1)](https://goreportcard.com/report/github.com/karmarun/karma.link)

## Overview

karma.link aims to bridge the gap between 'classic' IT development and the blockchain world.
In particular, karma.link exposes an easy-to-use JSON API that allows desktop, web & mobile applications to execute parts of their workload on the Ethereum blockchain.
This makes it possible to build conventional apps that leverage blockchain technology where adequate.

In contrast to software like the [Mist browser](https://github.com/ethereum/mist), karma.link is designed as an infrastructure tool for businesses.
In this context, users act on behalf of the organization they represent, fulfilling their jobs as instructed by their employer instead of acting of their own accord.

## Features

### Private Key Management

karma.link is designed with businesses in mind, where identities are associated with companies and organizations rather than individuals.
As such, it is the system administrator's job to connect natural users with pertinent Ethereum identities, according to company policy and governance rules.
This is why private key management is the most important point in karma.link's feature set.

### Smart Contract Management

Writing smart contracts (also called *Ðapps*) in [Solidity](http://solidity.readthedocs.io/en/latest/) is a straightforward task for a developer.
However, deploying and managing contracts on a company scale requires tools; system administration tools.
karma.link can read, understand and manage smart contracts. As such, it can give an operator valuable insights into its API surface.
Futhermore, karma.link can deploy a managed contract easily and report on its state.

### Ease of Integration

Ethereum smart contracts speak a rather complex binary format called the [Solidity ABI](http://solidity.readthedocs.io/en/latest/abi-spec.html).
Software libraries like [web3](https://github.com/ethereum/?q=web3) (for Javascript & Python) make it possible to encode and decode data in the ABI format.
However, making effective use of these libraries is not always as easy as one could wish and a moderate amount of knowledge is required to do it well.
karma.link understands Solidity's type system and ABI. This allows it to abstract the binary format away completely.
External services can invoke smart contracts through karma.link using only JSON and read back the results as JSON just as easily.

## Documentation

Please visit our [Github wiki](https://github.com/karmarun/karma.link/wiki).

## Project State

karma.link was released on Thursday, July the 5th. It is currently in [MVP](https://en.wikipedia.org/wiki/Minimum_viable_product) phase.
Any and all APIs are subject to change without notice.

## Installation

### Requirements

 - [git](https://git-scm.com/)
 - [go](https://golang.org/dl/)

### Fetch & Build

```bash
$ go get github.com/karmarun/karma.link/link
```

## License & Legal

Copyright 2018 karma.run AG. All rights reserved.

karma.link is a product of karma.run AG, registered in Zürich, Switzerland.

No license is provided (yet). See [here](https://choosealicense.com/no-permission/) what this means for you.

This software makes use of external software libraries from the [go-ethereum project](https://github.com/ethereum/go-ethereum),
which are licensed under the [GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html).
# mongo-go-driver

MongoDB Driver for Go.
[![GoDoc](https://godoc.org/github.com/mongodb/mongo-go-driver/mongo?status.svg)](https://godoc.org/github.com/mongodb/mongo-go-driver/mongo)

-------------------------
- [Requirements](#requirements)
- [Installation](#installation)
- [Bugs/Feature Reporting](#bugs-feature-reporting)
- [Testing / Development](#testing--development)
- [Continuous Integration](#continuous-integration)
- [License](#license)

-------------------------
## Requirements

- Go 1.9 or higher. We aim to support the latest supported versions go.
- MongoDB 3.2 and higher.

-------------------------
## Installation

The recommended way to get started using the MongoDB Go driver is by using `dep` to install the dependency in your project.

```bash
dep ensure -add github.com/mongodb/mongo-go-driver/mongo
```

-------------------------
## Bugs / Feature Reporting

New Features and bugs can be reported on jira: https://jira.mongodb.org/browse/GODRIVER

-------------------------
## Testing / Development

To run driver tests, make sure a MongoDB server instance is running at localhost:27017. Using make, you can run `make` (on windows, run `nmake`).
This will run coverage, run go-lint, run go-vet, and build the examples.

The MongoDB Go Driver is not feature complete, so any help is appreciated. Check out the [project page](https://jira.mongodb.org/browse/GODRIVER)
for tickets that need completing. See our [contribution guidelines](CONTRIBUTING.md) for details.

-------------------------
## Continuous Integration

Commits to master are run automatically on [evergreen](https://evergreen.mongodb.com/waterfall/mongo-go-driver).

-------------------------
## License

The MongoDB Go Driver is licensed under the [Apache License](LICENSE).

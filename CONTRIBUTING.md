# Contributing to the MongoDB Go Driver

Thank you for your interest in contributing to the MongoDB Go driver.

We are building this software together and strongly encourage contributions from the community that are within the guidelines set forth 
below. 

## Bug Fixes and New Features

Before starting to write code, look for existing [tickets](https://jira.mongodb.org/browse/GODRIVER) or 
[create one](https://jira.mongodb.org/secure/CreateIssue!default.jspa) for your bug, issue, or feature request. This helps the community 
avoid working on something that might not be of interest or which has already been addressed.

## Pull Requests

Pull requests should generally be made against the master (default) branch and include relevant tests, if applicable. 

Code should compile and tests should pass under all go versions which the driver currently supports.  Currently the driver 
supports a minimum version of go 1.7.  Please run 'make' to confirm.   By default, running the tests requires that you started a 
mongod server on localhost, listening on the default port. At minimum, please test against the latest release version of the MongoDB 
server.

If any tests do not pass, or relevant tests are not included, the 
pull request will not be considered. 

## Talk To Us

If you want to work on something or have questions / complaints please reach out to us by creating a Question issue at 
(https://jira.mongodb.org/secure/CreateIssue!default.jspa).

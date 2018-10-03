# Config Watch

## Tests

### Purpose

The purpose of the configwatch tests is essentially to validate the ability to
find and open files.The `configwatch_test.go` source file requires a single
file in `test-cases/configwatch.json` exists.

----------

### JSON Structure

The structure of the JSON should be an object called `testcases` followed by
an array of keyed indexes with the following names, "name", "type",
"description", "path" (see example below). The "type" can be either
"configwatch" or "configmachine".  Anything else is currently ignored. If
`type` is missing then an error will be thrown. Both tests are technically
verifying paths. However, one verifies the path independently of object
creation and the other verifies the path for the `ConfigWatch` object.

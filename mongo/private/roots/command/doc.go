// Package command contains abstractions for operations that can be performed against a MongoDB
// deployment. The types in this package are meant to provide a general set of commands that a
// user can run against a MongoDB database without knowing the version of the database.
//
// Each type is multi-leveled to support a variety of different interaction modes. At the highest
// level is the Dispatch method. This method takes a topology and handles the full cycle of the
// operation, including server selection, retrying writes, encoding the operation, decoding the
// response, and parsing the response into a result. The second highest level is the RoundTrip
// method. It does not handle server selection or retries and instead directly accepts a
// ServerDescription and a Connection. The third level are the Encode/Decode/Result/Err methods.
// These methods can be used to handle encoding and decoding wire messages and then extracting the
// result from the command. The Err method is provided as a way to check the error from an Decode
// without having to extract a result.
//
// Not all command types support all three levels, although all of them do support the lower two levels.
package command

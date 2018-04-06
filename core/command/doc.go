// Package command contains abstractions for operations that can be performed against a MongoDB
// deployment. The types in this package are meant to provide a general set of commands that a
// user can run against a MongoDB database without knowing the version of the database.
//
// Each type consists of two levels of interaction. The lowest level are the Encode and Decode
// methods. These are meant to be symmetric eventually, but currently only support the driver
// side of commands. The higher level is the RoundTrip method. This only makes sense from the
// driver side of commands and this method handles the encoding of the request and decoding of
// the response using the given wiremessage.ReadWriter.
package command

# BSON Depth Tracking
To help ensure that users of the BSON library do not crash their applications do a stack overflow
from attempting to decode a malicious document, we need to add depth tracking to the BSON library’s
validation functions and allow users to set the maximum depth of a BSON document the library will
decode.

## Scope of Work
The `bson.Document`, `bson.Reader`, `bson.Element`, and `bson.Value` types need to have their
validate methods augmented with additional depth tracking parameters to track how deep the
validation should go. Each of these types has a `validate` method except for the `bson.Value` type.

All of the internal calls during the validation process should use the `validate` method. This
method will take the current depth and the maximum depth. An increase in the current depth number
will occur before validating a BSON element of type `0x03`, `0x04`, or `0x0F`.  If, when entering
the block for validation, the current depth is greater than the maximum depth, an
`ErrMaximumDepthExceeded` error must be returned.

To allow users to set the maximum depth, each of the public `Validate` methods will take an
additional parameter of `maxDepth` that will be used as the parameter of the same name to each of
the `validate` methods.

## Testing
Unit tests must be written for each of the paths through validation, ensuring that the current depth
is incremented properly, and that an error is returned when max depth is exceeded.

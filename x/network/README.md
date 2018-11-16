# WARNING: NOT FOR PUBLIC USE

**The API for packages in `core` have no stability guarantee.**

The packages within the `core` directory would normally be put into an
`internal` directory to prohibit their use outside the `mongo` directory.

However, some MongoDB tools require very low-level access to the building
blocks of a driver, so we have placed them into `core` to allow these
packages to be imported by projects that need them.

These package APIs may be modified in backwards-incompatible ways at any
time.

**You are strongly discouraged from directly using any packages in
`core`.**

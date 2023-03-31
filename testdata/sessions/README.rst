====================
Driver Session Tests
====================

.. contents::

----

Introduction
============

The YAML and JSON files in this directory are platform-independent tests
meant to exercise a driver's implementation of sessions. These tests utilize the
`Unified Test Format <../../unified-test-format/unified-test-format.rst>`__.

Several prose tests, which are not easily expressed in YAML, are also presented
in the Driver Sessions Spec. Those tests will need to be manually implemented
by each driver.

Snapshot session tests
======================
Snapshot sessions tests require server of version 5.0 or higher and
replica set or a sharded cluster deployment.
Default snapshot history window on the server is 5 minutes. Running the test in debug mode, or in any other slow configuration
may lead to `SnapshotTooOld` errors. Drivers can work around this issue by increasing the server's `minSnapshotHistoryWindowInSeconds` parameter, for example:

.. code:: python

    client.admin.command('setParameter', 1, minSnapshotHistoryWindowInSeconds=60)

Prose tests
```````````

1. Setting both ``snapshot`` and ``causalConsistency`` to true is not allowed
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

* ``client.startSession(snapshot = true, causalConsistency = true)``
* Assert that an error was raised by driver

2. Pool is LIFO
~~~~~~~~~~~~~~~

This test applies to drivers with session pools.

* Call ``MongoClient.startSession`` twice to create two sessions, let us call them ``A`` and ``B``.
* Call ``A.endSession``, then ``B.endSession``.
* Call ``MongoClient.startSession``: the resulting session must have the same session ID as ``B``.
* Call ``MongoClient.startSession`` again: the resulting session must have the same session ID  as ``A``.

3. ``$clusterTime`` in commands
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

* Turn ``heartbeatFrequencyMS`` up to a very large number.
* Register a command-started and a command-succeeded APM listener. If the driver has no APM support, inspect commands/replies in another idiomatic way, such as monkey-patching or a mock server.
* Send a ``ping`` command to the server with the generic ``runCommand`` method.
* Assert that the command passed to the command-started listener includes ``$clusterTime`` if and only if ``maxWireVersion`` >= 6.
* Record the ``$clusterTime``, if any, in the reply passed to the command-succeeded APM listener.
* Send another ``ping`` command.
* Assert that ``$clusterTime`` in the command passed to the command-started listener, if any, equals the ``$clusterTime`` in the previous server reply. (Turning ``heartbeatFrequencyMS`` up prevents an intervening heartbeat from advancing the ``$clusterTime`` between these final two steps.)

Repeat the above for:

* An aggregate command from the ``aggregate`` helper method
* A find command from the ``find`` helper method
* An insert command from the ``insert_one`` helper method

4. Explicit and implicit session arguments
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

* Register a command-started APM listener. If the driver has no APM support, inspect commands in another idiomatic way, such as monkey-patching or a mock server.
* Create ``client1``
* Get ``database`` from ``client1``
* Get ``collection`` from ``database``
* Start ``session`` from ``client1``
* Call ``collection.insertOne(session,...)``
* Assert that the command passed to the command-started listener contained the session ``lsid`` from ``session``.
* Call ``collection.insertOne(,...)`` (*without* a session argument)
* Assert that the command passed to the command-started listener contained a session ``lsid``.

Repeat the above for all methods that take a session parameter.

5. Session argument is for the right client
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

* Create ``client1`` and ``client2``
* Get ``database`` from ``client1``
* Get ``collection`` from ``database``
* Start ``session`` from ``client2``
* Call ``collection.insertOne(session,...)``
* Assert that an error was reported because ``session`` was not started from ``client1``

Repeat the above for all methods that take a session parameter.

6. No further operations can be performed using a session after ``endSession`` has been called
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

* Start a ``session``
* End the ``session``
* Call ``collection.InsertOne(session, ...)``
* Assert that the proper error was reported

Repeat the above for all methods that take a session parameter.

If your driver implements a platform dependent idiomatic disposal pattern, test
that also (if the idiomatic disposal pattern calls ``endSession`` it would be
sufficient to only test the disposal pattern since that ends up calling
``endSession``).

7. Authenticating as multiple users suppresses implicit sessions
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Skip this test if your driver does not allow simultaneous authentication with multiple users.

* Authenticate as two users
* Call ``findOne`` with no explicit session
* Capture the command sent to the server
* Assert that the command sent to the server does not have an ``lsid`` field

8. Client-side cursor that exhausts the results on the initial query immediately returns the implicit session to the pool
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

* Insert two documents into a collection
* Execute a find operation on the collection and iterate past the first document
* Assert that the implicit session is returned to the pool. This can be done in several ways:

  * Track in-use count in the server session pool and assert that the count has dropped to zero
  * Track the lsid used for the find operation (e.g. with APM) and then do another operation and
    assert that the same lsid is used as for the find operation.

9. Client-side cursor that exhausts the results after a ``getMore`` immediately returns the implicit session to the pool
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

* Insert five documents into a collection
* Execute a find operation on the collection with batch size of 3
* Iterate past the first four documents, forcing the final ``getMore`` operation
* Assert that the implicit session is returned to the pool prior to iterating past the last document

10. No remaining sessions are checked out after each functional test
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

At the end of every individual functional test of the driver, there SHOULD be an
assertion that there are no remaining sessions checked out from the pool. This
may require changes to existing tests to ensure that they close any explicit
client sessions and any unexhausted cursors.

11. For every combination of topology and readPreference, ensure that ``find`` and ``getMore`` both send the same session id
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

* Insert three documents into a collection
* Execute a ``find`` operation on the collection with a batch size of 2
* Assert that the server receives a non-zero lsid
* Iterate through enough documents (3) to force a ``getMore``
* Assert that the server receives a non-zero lsid equal to the lsid that ``find`` sent.

12. Session pool can be cleared after forking without calling ``endSession``
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Skip this test if your driver does not allow forking.

* Create ClientSession
* Record its lsid
* Delete it (so the lsid is pushed into the pool)
* Fork
* In the parent, create a ClientSession and assert its lsid is the same.
* In the child, create a ClientSession and assert its lsid is different.

13. Existing sessions are not checked into a cleared pool after forking
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Skip this test if your driver does not allow forking.

* Create ClientSession
* Record its lsid
* Fork
* In the parent, return the ClientSession to the pool, create a new ClientSession, and assert its lsid is the same.
* In the child, return the ClientSession to the pool, create a new ClientSession, and assert its lsid is different.

14. Implicit sessions only allocate their server session after a successful connection checkout
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

* Create a MongoClient with the following options: ``maxPoolSize=1`` and ``retryWrites=true``. If testing against a sharded deployment, the test runner MUST ensure that the MongoClient connects to only a single mongos host.
* Attach a command started listener that collects each command's lsid
* Initiate the following concurrent operations

  * ``insertOne({ }),``
  * ``deleteOne({ }),``
  * ``updateOne({ }, { $set: { a: 1 } }),``
  * ``bulkWrite([{ updateOne: { filter: { }, update: { $set: { a: 1 } } } }]),``
  * ``findOneAndDelete({ }),``
  * ``findOneAndUpdate({ }, { $set: { a: 1 } }),``
  * ``findOneAndReplace({ }, { a: 1 }),``
  * ``find().toArray()``

* Wait for all operations to complete successfully
* Assert the following across at least 5 retries of the above test:

  * Drivers MUST assert that exactly one session is used for all operations at
    least once across the retries of this test.
  * Note that it's possible, although rare, for >1 server session to be used
    because the session is not released until after the connection is checked in.
  * Drivers MUST assert that the number of allocated sessions is strictly less
    than the number of concurrent operations in every retry of this test. In
    this instance it would be less than (but NOT equal to) 8.

15. ``lsid`` is added inside ``$query`` when using OP_QUERY
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This test only applies to drivers that have not implemented OP_MSG and still use OP_QUERY.

* For a command to a mongos that includes a readPreference, verify that the
  ``lsid`` on query commands is added inside the ``$query`` field, and NOT as a
  top-level field.

16. Authenticating as a second user after starting a session results in a server error
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This test only applies to drivers that allow authentication to be changed on the fly.

* Authenticate as the first user
* Start a session by calling ``startSession``
* Authenticate as a second user
* Call ``findOne`` using the session as an explicit session
* Assert that the driver returned an error because multiple users are authenticated

17. Driver verifies that the session is owned by the current user
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This test only applies to drivers that allow authentication to be changed on the fly.

* Authenticate as user A
* Start a session by calling ``startSession``
* Logout user A
* Authenticate as user B
* Call ``findOne`` using the session as an explicit session
* Assert that the driver returned an error because the session is owned by a different user

Changelog
=========

:2019-05-15: Initial version.
:2021-06-15: Added snapshot-session tests. Introduced legacy and unified folders.
:2021-07-30: Use numbering for prose test
:2022-02-11: Convert legacy tests to unified format
:2022-06-13: Relocate prose test from spec document and apply new ordering

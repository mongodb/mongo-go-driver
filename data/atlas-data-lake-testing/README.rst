=====================
Atlas Data Lake Tests
=====================

.. contents::

----

Introduction
============

The YAML and JSON files in this directory tree are platform-independent tests
that drivers can use to assert compatibility with `Atlas Data Lake <https://www.mongodb.com/docs/datalake>`_.

Running these integration tests will require a running ``mongohoused``
with data available in its ``test.driverdata`` collection. See the
`ADL directory in drivers-evergreen-tools <https://github.com/mongodb-labs/drivers-evergreen-tools/tree/master/.evergreen/atlas_data_lake>`_
and `10gen/mongohouse README <https://github.com/10gen/mongohouse/blob/master/README.md>`_
for more information.

Several prose tests, which are not easily expressed in YAML, are also presented
in this file. Those tests will need to be manually implemented by each driver.

Test Format
===========

The same as the `CRUD Spec Test format <../../crud/tests/README.rst#Test-Format>`_.

Test Runner Implementation
==========================

The same as the `CRUD Spec Test Runner Implementation <../../crud/tests#test-runner-implementation>`_,
with one notable differences: the test runner for Atlas Data Lake Testing
MUST NOT drop the collection and/or database under test. In contrast to other
CRUD tests, which insert their own data fixtures into an empty collection, data
for these tests is specified in the ``mongohoused`` configuration file.

Prose Tests
===========

The following tests MUST be implemented to fully test compatibility with
Atlas Data Lake.

#. Test that the driver properly constructs and issues a
   `killCursors <https://www.mongodb.com/docs/manual/reference/command/killCursors/>`_
   command to Atlas Data Lake. For this test, configure an APM listener on a
   client and execute a query on the ``test.driverdata`` collection that will
   leave a cursor open on the server (e.g. specify ``batchSize=2`` for a query
   that would match 3+ documents). Drivers MAY iterate the cursor if necessary
   to execute the initial ``find`` command but MUST NOT iterate further to avoid
   executing a ``getMore``.

   Observe the CommandSucceededEvent event for the ``find`` command and extract
   the cursor's ID and namespace from the response document's ``cursor.id`` and
   ``cursor.ns`` fields, respectively. Destroy the cursor object and observe
   a CommandStartedEvent and CommandSucceededEvent for the ``killCursors``
   command. Assert that the cursor ID and target namespace in the outgoing
   command match the values from the ``find`` command's CommandSucceededEvent.
   When matching the namespace, note that the ``killCursors`` field will contain
   the collection name and the database may be inferred from either the ``$db``
   field or accessed via the CommandStartedEvent directly. Finally, assert that
   the ``killCursors`` CommandSucceededEvent indicates that the expected cursor
   was killed in the ``cursorsKilled`` field.

   Note: this test assumes that drivers only issue a ``killCursors`` command
   internally when destroying a cursor that may still exist on the server. If
   a driver constructs and issues ``killCursors`` commands in other ways (e.g.
   public API), this test MUST be adapted to test all such code paths.

#. Test that the driver can establish a connection with Atlas Data Lake
   without authentication. For these tests, create a MongoClient using a
   valid connection string without auth credentials and execute a ping
   command.

#. Test that the driver can establish a connection with Atlas Data Lake
   with authentication. For these tests, create a MongoClient using a
   valid connection string with SCRAM-SHA-1 and credentials from the
   drivers-evergreen-tools ADL configuration and execute a ping command.
   Repeat this test using SCRAM-SHA-256.

Changelog
=========

:2020-07-15: Link to CRUD test runner implementation and note that the collection
             under test must not be dropped before each test.

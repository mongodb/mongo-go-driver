===========
MongoDB AWS
===========

There are 5 scenarios drivers MUST test:

#. ``Regular Credentials``: Auth via an ``ACCESS_KEY_ID`` and ``SECRET_ACCESS_KEY`` pair
#. ``EC2 Credentials``: Auth from an EC2 instance via temporary credentials assigned to the machine
#. ``ECS Credentials``: Auth from an ECS instance via temporary credentials assigned to the task
#. ``Assume Role``: Auth via temporary credentials obtained from an STS AssumeRole request
#. ``AWS Lambda``: Auth via environment variables ``AWS_ACCESS_KEY_ID``, ``AWS_SECRET_ACCESS_KEY``, and ``AWS_SESSION_TOKEN``.

For brevity, this section gives the values ``<AccessKeyId>``, ``<SecretAccessKey>`` and ``<Token>`` in place of a valid access key ID, secret access key and session token (also known as a security token). Note that if these values are passed into the URI they MUST be URL encoded. Sample values are below.

.. code-block:: 

  AccessKeyId=AKIAI44QH8DHBEXAMPLE
  SecretAccessKey=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  Token=AQoDYXdzEJr...<remainder of security token>
|
.. sectnum::

Regular credentials
======================

Drivers MUST be able to authenticate by providing a valid access key id and secret access key pair as the username and password, respectively, in the MongoDB URI. An example of a valid URI would be:

.. code-block:: 

  mongodb://<AccessKeyId>:<SecretAccessKey>@localhost/?authMechanism=MONGODB-AWS
|
EC2 Credentials
===============

Drivers MUST be able to authenticate from an EC2 instance via temporary credentials assigned to the machine. A sample URI on an EC2 machine would be:

.. code-block::
  
  mongodb://localhost/?authMechanism=MONGODB-AWS
|
.. note:: No username, password or session token is passed into the URI. Drivers MUST query the EC2 instance endpoint to obtain these credentials. 

ECS instance
============

Drivers MUST be able to authenticate from an ECS container via temporary credentials. A sample URI in an ECS container would be:

.. code-block::

  mongodb://localhost/?authMechanism=MONGODB-AWS
|
.. note:: No username, password or session token is passed into the URI. Drivers MUST query the ECS container endpoint to obtain these credentials. 

AssumeRole
==========

Drivers MUST be able to authenticate using temporary credentials returned from an assume role request. These temporary credentials consist of an access key ID, a secret access key, and a security token passed into the URI. A sample URI would be: 

.. code-block::

  mongodb://<AccessKeyId>:<SecretAccessKey>@localhost/?authMechanism=MONGODB-AWS&authMechanismProperties=AWS_SESSION_TOKEN:<Token>
|
AWS Lambda
==========

Drivers MUST be able to authenticate via an access key ID, secret access key and optional session token taken from the environment variables, respectively: 

.. code-block::

  AWS_ACCESS_KEY_ID
  AWS_SECRET_ACCESS_KEY 
  AWS_SESSION_TOKEN
|

Sample URIs both with and without optional session tokens set are shown below. Drivers MUST test both cases.

.. code-block:: bash

  # without a session token
  export AWS_ACCESS_KEY_ID="<AccessKeyId>"
  export AWS_SECRET_ACCESS_KEY="<SecretAccessKey>"

  URI="mongodb://localhost/?authMechanism=MONGODB-AWS"
|
.. code-block:: bash

  # with a session token
  export AWS_ACCESS_KEY_ID="<AccessKeyId>"
  export AWS_SECRET_ACCESS_KEY="<SecretAccessKey>"
  export AWS_SESSION_TOKEN="<Token>"

  URI="mongodb://localhost/?authMechanism=MONGODB-AWS"
|
.. note:: No username, password or session token is passed into the URI. Drivers MUST check the environment variables listed above for these values. If the session token is set Drivers MUST use it.

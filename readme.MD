Aws utils
=============
This is a go branching of the old ruby repository.
Check health status of aws servers based on the tagging system.

Requirements
--------------
* Current version was compiled under go 1.75 and go 1.8
* for easy building you'll need `make` installed.

Setting up
-----------------
You should have your GOPATH setup.
just run `make setup` to download any prerequisites. it will be installed into the current GOPATH and not the current directory.


Building
-------------
Run `make` to build the program, `make run` to run it.
The binary object will be created under the bin folder.

Environment variables
---------------------
currently there is one - PHONE_NUMBER which is the phone to send the text messages from aws in case of failure.

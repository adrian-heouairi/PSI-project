# PEER TO PEER FILE SHARING CLIENT AND SERVER
Go implementation of a peer to peer client and server using `jch.irif.fr` as REST server and main peer. 
## Usage
Install Go &gt;= 1.21, with `sudo snap install go --classic` on Ubuntu.
In the project root, run `go run . [--debug] [command to run]...` or `go run . help`.
## Features
+ NAT traversal
+ List connected peers and their addresses (IP + port)
+ Download a file at a given path (`<PEERNAME>/PATH`) in `PSI-download/PEERNAME/PATH`
+ Share data put in `PSI-shared-files/` to other peers
+ Readline CLI with tab completion
+ Signature of messages with `ECDSA P-256`
## Contributors
DERVISHI Sevi  
HEOUAIRI Adrian

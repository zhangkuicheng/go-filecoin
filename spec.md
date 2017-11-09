# filecoin network protocol spec

## New Blocks
When you publish a new block, send a message to all of your peers containing:

- The block
- Its hash
- Transactions in the block
- Its 'Score'

Upon receiving this message, each peer should then validate the message (see
'validating new blocks' below) and if successful, send it to each of their peers who
is not already known to have that block. Each peer should track the latest
blocks it knows about for each of its peers to avoid unnecessary
retransmissions.

### Validating new blocks
To validate a block, first check that the block in the message matches the hash
in the message, and that the block's score matches the score listed in the
message. Then, reconstruct the $TRANSACTION_STORAGE_DATA_STRUCTURE from the
transactions in the message and check that it matches the root in the block.
While rebuilding, verify that each transaction is itself valid when applied to
the current state. [TODO: detail other validations, including proofs]

TODO: other peers will likely already have most if not all of the transactions
in the block, perhaps don't send them immediately? Use bitswap to fetch ones
that are needed. Since we will likely be keeping track of which transactions we
have sent to which of our peers, this might be fairly trivial.

## New Transactions

New transactions created by a peer should be broadcasted to all connected
peers. Upon receiving a new transaction from a peer, each peer should validate
the transaction (given some validation rules, not necessarily just a ecdsa
signature), then they should validate that the state change specified in the
transaction is valid given the current state trie.

Each peer should keep a set of which recently broadcasted transactions have
been sent to which of its connected peers to prevent wasteful retransmissions.

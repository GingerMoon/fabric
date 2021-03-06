/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package statebasedval

import (
	"encoding/base64"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator/internal"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/fpga"
	fpgapb "github.com/hyperledger/fabric/protos/fpga"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/protos/peer"
)

var logger = flogging.MustGetLogger("statebasedval")

// Validator validates a tx against the latest committed state
// and preceding valid transactions with in the same block
type Validator struct {
	db privacyenabledstate.DB
}

// NewValidator constructs StateValidator
func NewValidator(db privacyenabledstate.DB) *Validator {
	return &Validator{db}
}

// preLoadCommittedVersionOfRSet loads committed version of all keys in each
// transaction's read set into a cache.
func (v *Validator) preLoadCommittedVersionOfRSet(block *internal.Block) error {

	// Collect both public and hashed keys in read sets of all transactions in a given block
	var pubKeys []*statedb.CompositeKey
	var hashedKeys []*privacyenabledstate.HashedCompositeKey

	// pubKeysMap and hashedKeysMap are used to avoid duplicate entries in the
	// pubKeys and hashedKeys. Though map alone can be used to collect keys in
	// read sets and pass as an argument in LoadCommittedVersionOfPubAndHashedKeys(),
	// array is used for better code readability. On the negative side, this approach
	// might use some extra memory.
	pubKeysMap := make(map[statedb.CompositeKey]interface{})
	hashedKeysMap := make(map[privacyenabledstate.HashedCompositeKey]interface{})

	for _, tx := range block.Txs {
		for _, nsRWSet := range tx.RWSet.NsRwSets {
			for _, kvRead := range nsRWSet.KvRwSet.Reads {
				compositeKey := statedb.CompositeKey{
					Namespace: nsRWSet.NameSpace,
					Key:       kvRead.Key,
				}
				if _, ok := pubKeysMap[compositeKey]; !ok {
					pubKeysMap[compositeKey] = nil
					pubKeys = append(pubKeys, &compositeKey)
				}

			}
			for _, colHashedRwSet := range nsRWSet.CollHashedRwSets {
				for _, kvHashedRead := range colHashedRwSet.HashedRwSet.HashedReads {
					hashedCompositeKey := privacyenabledstate.HashedCompositeKey{
						Namespace:      nsRWSet.NameSpace,
						CollectionName: colHashedRwSet.CollectionName,
						KeyHash:        string(kvHashedRead.KeyHash),
					}
					if _, ok := hashedKeysMap[hashedCompositeKey]; !ok {
						hashedKeysMap[hashedCompositeKey] = nil
						hashedKeys = append(hashedKeys, &hashedCompositeKey)
					}
				}
			}
		}
	}

	// Load committed version of all keys into a cache
	if len(pubKeys) > 0 || len(hashedKeys) > 0 {
		err := v.db.LoadCommittedVersionsOfPubAndHashedKeys(pubKeys, hashedKeys)
		if err != nil {
			return err
		}
	}

	return nil
}

// internal.Block is an internal package, hence cannot be accessed outside
func generateBlock4mvcc(block *internal.Block) *fpgapb.Block4Mvcc {
	logger.Debugf("generating block for mvcc...")
	blockmvcc := new(fpgapb.Block4Mvcc)
	blockmvcc.Num = block.Num
	logger.Debugf("block number: %v", blockmvcc.Num)

	txs := make([]*fpgapb.Transaction4Mvcc, len(block.Txs))
	for i, tx := range block.Txs {
		txs[i] = &fpgapb.Transaction4Mvcc{}
		txs[i].Id = tx.ID // for those config tx, tx.ID is empty.
		txs[i].IndexInBlock = int32(tx.IndexInBlock)
		logger.Debugf("tx id: %v", txs[i].Id)
		logger.Debugf("tx id index: %v", txs[i].IndexInBlock)

		// the key of fpgapb.TxRS and fpgapb.TxWS need to be constructed by "namespace key".
		// [stateleveldb] ApplyUpdates -> DEBU 160 Channel [mychannel]:
		// Applying key(string)=[mycc 2] key(bytes)=[[]byte{0x6d, 0x79, 0x63, 0x63, 0x0, 0x32}]
		for _, rwset := range tx.RWSet.NsRwSets {
			logger.Debugf("namespace is %v.", rwset.NameSpace)
			// read set
			txs[i].RdCount += uint32(len(rwset.KvRwSet.Reads))
			for _, e := range rwset.KvRwSet.Reads {
				rs := &fpgapb.TxRS{}
				rs.Key = rwset.NameSpace + " " + e.Key
				rs.Version = e.Version
				txs[i].Rs = append(txs[i].Rs, rs)
				logger.Debugf("txs(%v) rs: %+v appended to read set.", txs[i].Id, rs)
			}

			// wirte set
			txs[i].WtCount += uint32(len(rwset.KvRwSet.Writes))
			for _, e := range rwset.KvRwSet.Writes {
				ws := &fpgapb.TxWS{}
				ws.Key = rwset.NameSpace + " " + e.Key
				ws.Value = e.Value
				ws.IsDel = e.IsDelete
				txs[i].Ws = append(txs[i].Ws, ws)
				logger.Debugf("txs(%v) ws: {Key: %v, Value(base64 encoded hash): %v, IsDel: %v, } appended to write set.",
					txs[i].Id, ws.Key, base64.StdEncoding.EncodeToString(util.ComputeSHA256(ws.Value)), ws.IsDel)
			}
		}
		logger.Debugf("txs(%v) RdCount is %v.", txs[i].Id, txs[i].RdCount)
		logger.Debugf("txs(%v) WtCount is %v.", txs[i].Id, txs[i].WtCount)
	}
	blockmvcc.Txs = txs
	logger.Debugf("block for mvcc generated.")
	return blockmvcc
}

// ValidateAndPrepareBatch implements method in Validator interface
func (v *Validator) ValidateAndPrepareBatch(block *internal.Block, doMVCCValidation bool) (*internal.PubAndHashUpdates, error) {
	// Check whether statedb implements BulkOptimizable interface. For now,
	// only CouchDB implements BulkOptimizable to reduce the number of REST
	// API calls from peer to CouchDB instance.
	if v.db.IsBulkOptimizable() {
		err := v.preLoadCommittedVersionOfRSet(block)
		if err != nil {
			return nil, err
		}
	}

	updates := internal.NewPubAndHashUpdates()

	if block.Num == 0 {
		for _, tx := range block.Txs {
			var validationCode peer.TxValidationCode
			var err error
			if validationCode, err = v.validateEndorserTX(tx.RWSet, doMVCCValidation, updates); err != nil {
				return nil, err
			}

			tx.ValidationCode = validationCode
			if validationCode == peer.TxValidationCode_VALID {
				logger.Debugf("Block [%d] Transaction index [%d] TxId [%s] marked as valid by state validator", block.Num, tx.IndexInBlock, tx.ID)
				committingTxHeight := version.NewHeight(block.Num, uint64(tx.IndexInBlock))
				updates.ApplyWriteSet(tx.RWSet, committingTxHeight, v.db)
			} else {
				logger.Warningf("Block [%d] Transaction index [%d] TxId [%s] marked as invalid by state validator. Reason code [%s]",
					block.Num, tx.IndexInBlock, tx.ID, validationCode.String())
			}
		}
	} else {
		if doMVCCValidation {
			// in the original code,
			// state_based_validator.go
			// func (v *Validator) validateKVRead(ns string, kvRead *kvrwset.KVRead, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
			//	if updates.Exists(ns, kvRead.Key) {
			//		return false, nil
			reply := fpga.SendBlock4MvccBlockRpc(block.CommonBlock)

			for i, tx := range block.Txs {
				txReply := reply.TxReplies[tx.IndexInBlock] // !!! TXReply need to be put in the indexInBlock position. Else Fabric will need to interate through the BlockReply.

				if !txReply.SgValid{ // vscc failed
					if i == 0 {
						tx.ValidationCode = peer.TxValidationCode_BAD_CREATOR_SIGNATURE
						logger.Warningf("Block [%d] Transaction index [%d] TxId [%s] marked as invalid by state validator. The reason is invalid tx creator's signature",
							block.Num, tx.IndexInBlock, tx.ID)
					} else {
						tx.ValidationCode = peer.TxValidationCode_INVALID_ENDORSER_TRANSACTION
						logger.Warningf("Block [%d] Transaction index [%d] TxId [%s] marked as invalid by state validator. The reason is invalid signature",
							block.Num, tx.IndexInBlock, tx.ID)
					}
					continue
				}

				bMvcc := true
				for _, readReply := range txReply.RdChecks {
					if !readReply.RdValid {
						logger.Warningf("Block [%d] Transaction index [%d] TxId [%s] marked as invalid by state validator. The reason is read version conflict of the key [%s]",
							block.Num, tx.IndexInBlock, tx.ID, readReply.RdKey)
						bMvcc = false
						break
					}
				}
				if !bMvcc {
					tx.ValidationCode = peer.TxValidationCode_MVCC_READ_CONFLICT
					continue
				}

				tx.ValidationCode = peer.TxValidationCode_VALID
				logger.Debugf("Block [%d] Transaction index [%d] TxId [%s] marked as valid by state validator", block.Num, tx.IndexInBlock, tx.ID)
				committingTxHeight := version.NewHeight(block.Num, uint64(tx.IndexInBlock))
				updates.ApplyWriteSet(tx.RWSet, committingTxHeight, v.db)

				//if txReply.TxValid {
				//	tx.ValidationCode = peer.TxValidationCode_VALID
				//	logger.Debugf("Block [%d] Transaction index [%d] TxId [%s] marked as valid by state validator", block.Num, tx.IndexInBlock, tx.ID)
				//	committingTxHeight := version.NewHeight(block.Num, uint64(tx.IndexInBlock))
				//	updates.ApplyWriteSet(tx.RWSet, committingTxHeight, v.db)
				//} else if !txReply.SgValid{ // vscc failed
				//	tx.ValidationCode = peer.TxValidationCode_INVALID_ENDORSER_TRANSACTION
				//	logger.Warningf("Block [%d] Transaction index [%d] TxId [%s] marked as invalid by state validator. The reason is invalid signature",
				//		block.Num, tx.IndexInBlock, tx.ID)
				//} else { // mvcc failed
				//	tx.ValidationCode = peer.TxValidationCode_MVCC_READ_CONFLICT
				//	for _, readReply := range txReply.RdChecks {
				//		if !readReply.RdValid {
				//			logger.Warningf("Block [%d] Transaction index [%d] TxId [%s] marked as invalid by state validator. The reason is read version conflict of the key [%s]",
				//				block.Num, tx.IndexInBlock, tx.ID, readReply.RdKey)
				//		}
				//	}
				//}
			}
		}
		//block4mvcc := generateBlock4mvcc(block)
		//if block4mvcc != nil {
		//	response := fpga.SendBlock4Mvcc(block4mvcc)
		//	if !response.Result {
		//		logger.Panicf("fpga mvcc failed!")
		//	}
		//}
	}

	return updates, nil
}

// validateEndorserTX validates endorser transaction
func (v *Validator) validateEndorserTX(
	txRWSet *rwsetutil.TxRwSet,
	doMVCCValidation bool,
	updates *internal.PubAndHashUpdates) (peer.TxValidationCode, error) {

	var validationCode = peer.TxValidationCode_VALID
	var err error
	//mvccvalidation, may invalidate transaction
	if doMVCCValidation {
		validationCode, err = v.validateTx(txRWSet, updates)
	}
	return validationCode, err
}

func (v *Validator) validateTx(txRWSet *rwsetutil.TxRwSet, updates *internal.PubAndHashUpdates) (peer.TxValidationCode, error) {
	// Uncomment the following only for local debugging. Don't want to print data in the logs in production
	//logger.Debugf("validateTx - validating txRWSet: %s", spew.Sdump(txRWSet))
	for _, nsRWSet := range txRWSet.NsRwSets {
		ns := nsRWSet.NameSpace
		// Validate public reads
		if valid, err := v.validateReadSet(ns, nsRWSet.KvRwSet.Reads, updates.PubUpdates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_MVCC_READ_CONFLICT, nil
		}
		// Validate range queries for phantom items
		if valid, err := v.validateRangeQueries(ns, nsRWSet.KvRwSet.RangeQueriesInfo, updates.PubUpdates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_PHANTOM_READ_CONFLICT, nil
		}
		// Validate hashes for private reads
		if valid, err := v.validateNsHashedReadSets(ns, nsRWSet.CollHashedRwSets, updates.HashUpdates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_MVCC_READ_CONFLICT, nil
		}
	}
	return peer.TxValidationCode_VALID, nil
}

////////////////////////////////////////////////////////////////////////////////
/////                 Validation of public read-set
////////////////////////////////////////////////////////////////////////////////
func (v *Validator) validateReadSet(ns string, kvReads []*kvrwset.KVRead, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	for _, kvRead := range kvReads {
		if valid, err := v.validateKVRead(ns, kvRead, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// validateKVRead performs mvcc check for a key read during transaction simulation.
// i.e., it checks whether a key/version combination is already updated in the statedb (by an already committed block)
// or in the updates (by a preceding valid transaction in the current block)
func (v *Validator) validateKVRead(ns string, kvRead *kvrwset.KVRead, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	if updates.Exists(ns, kvRead.Key) {
		return false, nil
	}
	committedVersion, err := v.db.GetVersion(ns, kvRead.Key)
	if err != nil {
		return false, err
	}

	logger.Debugf("Comparing versions for key [%s]: committed version=%#v and read version=%#v",
		kvRead.Key, committedVersion, rwsetutil.NewVersion(kvRead.Version))
	if !version.AreSame(committedVersion, rwsetutil.NewVersion(kvRead.Version)) {
		logger.Debugf("Version mismatch for key [%s:%s]. Committed version = [%#v], Version in readSet [%#v]",
			ns, kvRead.Key, committedVersion, kvRead.Version)
		return false, nil
	}
	return true, nil
}

////////////////////////////////////////////////////////////////////////////////
/////                 Validation of range queries
////////////////////////////////////////////////////////////////////////////////
func (v *Validator) validateRangeQueries(ns string, rangeQueriesInfo []*kvrwset.RangeQueryInfo, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	for _, rqi := range rangeQueriesInfo {
		if valid, err := v.validateRangeQuery(ns, rqi, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// validateRangeQuery performs a phantom read check i.e., it
// checks whether the results of the range query are still the same when executed on the
// statedb (latest state as of last committed block) + updates (prepared by the writes of preceding valid transactions
// in the current block and yet to be committed as part of group commit at the end of the validation of the block)
func (v *Validator) validateRangeQuery(ns string, rangeQueryInfo *kvrwset.RangeQueryInfo, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	logger.Debugf("validateRangeQuery: ns=%s, rangeQueryInfo=%s", ns, rangeQueryInfo)

	// If during simulation, the caller had not exhausted the iterator so
	// rangeQueryInfo.EndKey is not actual endKey given by the caller in the range query
	// but rather it is the last key seen by the caller and hence the combinedItr should include the endKey in the results.
	includeEndKey := !rangeQueryInfo.ItrExhausted

	combinedItr, err := newCombinedIterator(v.db, updates.UpdateBatch,
		ns, rangeQueryInfo.StartKey, rangeQueryInfo.EndKey, includeEndKey)
	if err != nil {
		return false, err
	}
	defer combinedItr.Close()
	var validator rangeQueryValidator
	if rangeQueryInfo.GetReadsMerkleHashes() != nil {
		logger.Debug(`Hashing results are present in the range query info hence, initiating hashing based validation`)
		validator = &rangeQueryHashValidator{}
	} else {
		logger.Debug(`Hashing results are not present in the range query info hence, initiating raw KVReads based validation`)
		validator = &rangeQueryResultsValidator{}
	}
	validator.init(rangeQueryInfo, combinedItr)
	return validator.validate()
}

////////////////////////////////////////////////////////////////////////////////
/////                 Validation of hashed read-set
////////////////////////////////////////////////////////////////////////////////
func (v *Validator) validateNsHashedReadSets(ns string, collHashedRWSets []*rwsetutil.CollHashedRwSet,
	updates *privacyenabledstate.HashedUpdateBatch) (bool, error) {
	for _, collHashedRWSet := range collHashedRWSets {
		if valid, err := v.validateCollHashedReadSet(ns, collHashedRWSet.CollectionName, collHashedRWSet.HashedRwSet.HashedReads, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

func (v *Validator) validateCollHashedReadSet(ns, coll string, kvReadHashes []*kvrwset.KVReadHash,
	updates *privacyenabledstate.HashedUpdateBatch) (bool, error) {
	for _, kvReadHash := range kvReadHashes {
		if valid, err := v.validateKVReadHash(ns, coll, kvReadHash, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// validateKVReadHash performs mvcc check for a hash of a key that is present in the private data space
// i.e., it checks whether a key/version combination is already updated in the statedb (by an already committed block)
// or in the updates (by a preceding valid transaction in the current block)
func (v *Validator) validateKVReadHash(ns, coll string, kvReadHash *kvrwset.KVReadHash,
	updates *privacyenabledstate.HashedUpdateBatch) (bool, error) {
	if updates.Contains(ns, coll, kvReadHash.KeyHash) {
		return false, nil
	}
	committedVersion, err := v.db.GetKeyHashVersion(ns, coll, kvReadHash.KeyHash)
	if err != nil {
		return false, err
	}

	if !version.AreSame(committedVersion, rwsetutil.NewVersion(kvReadHash.Version)) {
		logger.Debugf("Version mismatch for key hash [%s:%s:%#v]. Committed version = [%s], Version in hashedReadSet [%s]",
			ns, coll, kvReadHash.KeyHash, committedVersion, kvReadHash.Version)
		return false, nil
	}
	return true, nil
}

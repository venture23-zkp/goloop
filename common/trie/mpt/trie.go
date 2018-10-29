package mpt

import (
	"bytes"
	"reflect"
	"sync"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

type (
	mpt struct {
		root    node
		objType reflect.Type
		// committedHash is root hash in database
		// committedHash is updated by Flush()
		committedHash hash

		rootHashed bool
		// Set() inserts key & value into requestPool
		requestPool  map[string]trieValue
		prevSnapshot *mpt
		mutex        sync.Mutex
		db           db.DB
	}
)

/*
 */
func newMpt(db db.DB, initialHash hash) *mpt {
	return &mpt{root: hash(append([]byte(nil), []byte(initialHash)...)),
		committedHash: hash(append([]byte(nil), []byte(initialHash)...)),
		requestPool:   make(map[string]trieValue), db: db, objType: reflect.TypeOf([]byte{})}
}

func newImmutable(db db.DB, committedHash hash) *mpt {
	return &mpt{root: nil,
		committedHash: hash(append([]byte(nil), []byte(committedHash)...)),
		requestPool:   make(map[string]trieValue), db: db, objType: reflect.TypeOf([]byte{})}
}

func bytesToNibbles(k []byte) []byte {
	nibbles := make([]byte, len(k)*2)
	for i, v := range k {
		nibbles[i*2] = v >> 4 & 0x0F
		nibbles[i*2+1] = v & 0x0F
	}
	return nibbles
}

func (m *mpt) get(n node, k []byte) (node, trieValue, error) {
	var result trieValue
	var err error
	switch n := n.(type) {
	case *branch:
		if len(k) != 0 {
			n.nibbles[k[0]], result, err = m.get(n.nibbles[k[0]], k[1:])
		} else {
			result = n.value
		}
	case *extension:
		match := compareHex(n.sharedNibbles, k)
		n.next, result, err = m.get(n.next, k[match:])
		if err != nil {
			return nil, nil, err
		}
	case *leaf:
		return n, n.value, nil
	// if node is hash, get serialized value with hash from db then deserialize it.
	case hash:
		serializedValue, err := m.db.Get(n)
		if err != nil {
			return nil, nil, err
		}
		deserializedNode := deserialize(serializedValue, m.objType)
		switch m := deserializedNode.(type) {
		case *branch:
			m.hashedValue = n
		case *extension:
			m.hashedValue = n
		case *leaf:
			m.hashedValue = n
		}
		return m.get(deserializedNode, k)
	}
	return n, result, err
}

func (m *mpt) Get(k []byte) ([]byte, error) {
	k = bytesToNibbles(k)
	if v, ok := m.requestPool[string(k)]; ok {
		return v.(byteValue), nil
	}
	var value trieValue
	var err error
	m.root, value, err = m.get(m.root, k)
	if err != nil {
		return nil, err
	}
	return value.Bytes(), nil
}

func (m *mpt) evaluateTrie(requestPool map[string]trieValue) {
	for k, v := range requestPool {
		if v == nil {
			m.root, _, _ = m.delete(m.root, []byte(k))
		} else {
			m.root, _ = m.set(m.root, []byte(k), v)
		}
	}
}

/*
	RootHash
*/
func (m *mpt) RootHash() []byte {
	if m.rootHashed == true {
		return m.root.hash()
	}
	pool, lastCommitedHash := m.mergeSnapshot()
	committedHash := lastCommitedHash
	if len(committedHash) != 0 {
		m.root = committedHash
	}
	// That length of pool is zero means that this trie is already calculated
	if len(pool) == 0 {
		return m.root.hash()
	}
	m.evaluateTrie(pool)
	h := m.root.hash()
	m.rootHashed = true
	// Do not set nil to requestPool and prevSnapshot because next snapshot want data in previous snapshot
	//m.requestPool = nil
	//m.prevSnapshot = nil
	return h
}

// TODO: check set() code.
// return true if current node or child node is changed
func (m *mpt) set(n node, k []byte, v trieValue) (node, bool) {
	//fmt.Println("set n ", n,", k ", k, ", v : ", string(v.(byteValue)))
	switch n := n.(type) {
	case *branch:
		if len(k) == 0 {
			n.value = v
			return n, true
		}
		n.nibbles[k[0]], n.dirty = m.set(n.nibbles[k[0]], k[1:], v)
	case *extension:
		match := compareHex(k, n.sharedNibbles)
		// case 1 : match = 0 -> new branch
		switch {
		case match == 0:
			newBranch := &branch{}
			newBranch.nibbles[k[0]], _ = m.set(nil, k[1:], v)
			if len(n.sharedNibbles) == 1 {
				newBranch.nibbles[n.sharedNibbles[0]] = n.next
			} else {
				newBranch.nibbles[n.sharedNibbles[0]] = n
				n.sharedNibbles = n.sharedNibbles[1:]
			}
			return newBranch, true

		// case 2 : 0 < match < len(sharedNibbles) -> new extension
		case match < len(n.sharedNibbles):
			newBranch := &branch{}
			newExt := &extension{}
			newExt.sharedNibbles = k[:match]
			newExt.next = newBranch
			if match+1 == len(n.sharedNibbles) {
				newBranch.nibbles[n.sharedNibbles[match]] = n.next
			} else {
				newBranch.nibbles[n.sharedNibbles[match]] = n
				n.sharedNibbles = n.sharedNibbles[match+1:]
			}
			if match == len(k) {
				newBranch.value = v
			} else {
				newBranch.nibbles[k[match]], _ = m.set(nil, k[match+1:], v)
			}
			return newExt, true
		// case 3 : match == len(sharedNibbles) -> go to next
		case match < len(k):
			n.next, n.dirty = m.set(n.next, k[match:], v)
		//case match == len(n.sharedNibbles):
		default:
			nextBranch := n.next.(*branch)
			nextBranch.value = v
		}
	case *leaf:
		match := compareHex(k, n.keyEnd)
		// case 1 : match = 0 -> new branch
		switch {
		case match == 0:
			if v.Compare(n.value) == true {
				return n, false
			}
			newBranch := &branch{}
			if len(k) == 0 {
				newBranch.value = v
			} else {
				newBranch.nibbles[k[0]], _ = m.set(nil, k[1:], v)
			}
			if len(n.keyEnd) == 0 {
				newBranch.value = n.value
			} else {
				newBranch.nibbles[n.keyEnd[0]], _ = m.set(nil, n.keyEnd[1:], n.value)
			}

			return newBranch, true
		// case 2 : 0 < match < len(n,value) -> new extension
		case match < len(n.keyEnd):
			newExt := &extension{}
			newExt.sharedNibbles = k[:match]
			newBranch := &branch{}
			newExt.next = newBranch
			if match == len(k) {
				newBranch.value = v
			} else {
				newBranch.nibbles[k[match]], _ = m.set(nil, k[match+1:], v)
			}
			newBranch.nibbles[n.keyEnd[match]], _ = m.set(nil, n.keyEnd[match+1:], n.value)
			return newExt, true
		// case match == len(n.keyEnd)
		case match < len(k):
			newExt := &extension{}
			newExt.sharedNibbles = k[:match]
			newBranch := &branch{}
			newExt.next = newBranch
			newBranch.value = n.value
			newBranch.nibbles[k[match]], _ = m.set(nil, k[match+1:], v)
			return newExt, true
		// case 3 : match == len(n.value) -> update value
		default:
			n.value = v
		}
	case hash:
		// TODO: have to check error.
		if len(n) == 0 {
			return &leaf{keyEnd: k[:], value: v}, true
		}
		serializedValue, _ := m.db.Get(n)
		return m.set(deserialize(serializedValue, m.objType), k, v)

	default:
		return &leaf{keyEnd: k[:], value: v}, true
	}
	return n, true
}

/*
Set inserts key and value into requestPool.
RootHash, Proof, Flush insert keys and values in requestPool into trie
*/
func (m *mpt) Set(k, v []byte) error {
	// TODO: if k or v is nil, return error for invalid param
	if k == nil || v == nil {
		return nil // TODO: proper error
	}
	k = bytesToNibbles(k)
	m.mutex.Lock()
	copied := make([]byte, len(v))
	copy(copied, v)
	m.requestPool[string(k)] = byteValue(append([]byte(nil), v...))
	m.mutex.Unlock()
	//tr.root, _ = set(tr.root, k, v)
	return nil
}

// TODO: check delete code
// return node, dirty, error
func (m *mpt) delete(n node, k []byte) (node, bool, error) {
	//fmt.Println("delete n = ", n, ", k = ", k)
	var nextNode node
	switch n := n.(type) {
	case *branch:
		if nextNode, n.dirty, _ = m.delete(n.nibbles[k[0]], k[1:]); n.dirty == false {
			return n, false, nil
		}
		n.nibbles[k[0]] = nextNode

		// check remaining nibbles on n(current node)
		// 1. if n has only 1 remaining node after deleting, n will be removed and the remaining node will be changed to extension.
		// 2. if n has only value with no remaining node after deleting, node must be changed to leaf
		// Branch has least 2 nibbles before deleting so branch cannot be empty after deleting
		remainingNibble := 16
		for i, nn := range n.nibbles {
			if nn != nil {
				if remainingNibble != 16 { // already met another nibble
					remainingNibble = -1
					break
				}
				remainingNibble = i
			}
		}

		//If remainingNibble is -1, branch has 2 more nibbles.
		if remainingNibble != -1 {
			if remainingNibble == 16 {
				return &leaf{value: n.value}, true, nil
			} else {
				// check nextNode.
				// if nextNode is extension or branch, n must be extension
				switch nn := n.nibbles[remainingNibble].(type) {
				case *extension:
					return &extension{sharedNibbles: append([]byte{byte(remainingNibble)}, nn.sharedNibbles...), next: nn.next}, true, nil
				case *branch:
					return &extension{sharedNibbles: []byte{byte(remainingNibble)}, next: nn}, true, nil
				case *leaf:
					return &leaf{keyEnd: append([]byte{byte(remainingNibble)}, nn.keyEnd...), value: nn.value}, true, nil
				}
			}
		}

	case *extension:
		// cannot find data. Not exist
		if nextNode, n.dirty, _ = m.delete(n.next, k[len(n.sharedNibbles):]); n.dirty == false {
			return n, false, nil
		}
		switch nn := nextNode.(type) {
		// if child node is extension node, merge current node.
		// It can not be possible to link extension from extension directly.
		// extension has only branch as next node.
		case *extension:
			n.sharedNibbles = append(n.sharedNibbles, nn.sharedNibbles...)
			n.next = nn.next
		// if child node is leaf after deleting, this extension must merge next node and be changed to leaf.
		// if child node is leaf, new leaf(keyEnd = extension.key + child.keyEnd, val = child.val)
		case *leaf: // make new leaf and return it
			return &leaf{keyEnd: append(n.sharedNibbles, nn.keyEnd...), value: nn.value}, true, nil
		}
		n.next = nextNode

	case *leaf:
		if bytes.Compare(n.keyEnd, k) != 0 {
			return n, false, nil
		}
		return nil, true, nil

	case hash:
		if m.db == nil {
			return n, true, nil // TODO: proper error
		}
		serializedValue, err := m.db.Get(n)
		if err != nil {
			return n, true, err
		}
		return m.delete(deserialize(serializedValue, m.objType), k)

	default:
		return n, false, nil
	}

	return n, true, nil
}

func (m *mpt) Delete(k []byte) error {
	var err error
	k = bytesToNibbles(k)
	m.requestPool[string(k)] = nil
	return err
}

func (m *mpt) GetSnapshot() trie.Snapshot {
	mpt := newImmutable(m.db, m.committedHash)
	m.mutex.Lock()
	mpt.requestPool = m.requestPool
	mpt.prevSnapshot = m.prevSnapshot
	m.prevSnapshot = mpt
	m.requestPool = make(map[string]trieValue)
	m.mutex.Unlock()

	return mpt
}

func (m *mpt) mergeSnapshot() (map[string]trieValue, hash) {
	mergePool := make(map[string]trieValue)
	var committedHash hash
	for snapshot := m; snapshot != nil; snapshot = snapshot.prevSnapshot {
		for k, v := range snapshot.requestPool {
			// add only not existing key
			if _, ok := mergePool[k]; ok == false {
				mergePool[k] = v
			}
		}
		committedHash = snapshot.committedHash
	}
	return mergePool, committedHash
}

func traversalCommit(db db.DB, n node) error {
	switch n := n.(type) {
	case *branch:
		for _, v := range n.nibbles {
			if err := traversalCommit(db, v); err != nil {
				return err
			}
		}
	case *extension:
		if err := traversalCommit(db, n.next); err != nil {
			return err
		}
	case *leaf:
		serialized := n.serialize()
		// if length of serialized leaf is smaller hashable(32), parent node (branch) must have serialized data of this
		if len(serialized) < hashableSize {
			return nil
		}
	default:
		return nil
	}
	return db.Set(n.hash(), n.serialize())
}

/*
	Flush saves all updated nodes to db.
	Requested data are inserted to db so the requested data in pool are cleared
	And preve
*/
func (m *mpt) Flush() error {
	pool, lastCommitedHash := m.mergeSnapshot()
	m.committedHash = lastCommitedHash

	m.requestPool = pool
	if len(pool) != 0 {
		if m.rootHashed == false {
			if len(lastCommitedHash) != 0 {
				m.root = lastCommitedHash
			}
			m.evaluateTrie(pool)
			m.rootHashed = true
		}
		if err := traversalCommit(m.db, m.root); err != nil {
			return err
		}
		m.committedHash = m.root.hash()
	} else {
		m.root = m.committedHash
	}

	m.requestPool = nil
	m.prevSnapshot = nil
	return nil
}

func addProof(buf [][]byte, index int, hash []byte) {
	if len(buf) == index {
		buf = make([][]byte, len(buf)+10)
	}
	copy(buf[index], hash)
}

func (m *mpt) proof(n node, k []byte) ([][]byte, int) {
	var buf [][]byte
	var i int
	switch n := n.(type) {
	case *branch:
		buf, i = m.proof(n.nibbles[k[0]], k[1:])
		if n.hashedValue == nil {
			addProof(buf, i, n.serialize())
		} else {
			addProof(buf, i, n.hashedValue)
		}
	case *extension:
		match := compareHex(n.sharedNibbles, k)
		buf, i = m.proof(n.next, k[match:])
		if n.hashedValue == nil {
			addProof(buf, i, n.serialize())
		} else {
			addProof(buf, i, n.hashedValue)
		}
	case *leaf:
		return nil, 0
	case hash:
		// TODO: have to check error
		serializedValued, _ := m.db.Get(k)
		decodeingNode := deserialize(serializedValued, m.objType)
		return m.proof(decodeingNode, k)
	}
	return buf, i + 1
}

// TODO: Implement Proof
func (m *mpt) Proof(k []byte) [][]byte {
	m.root.serialize()
	k = bytesToNibbles(k)
	buf, _ := m.proof(m.root, k)
	return buf
}

func (m *mpt) Load(db db.DB, root []byte) error {
	// use db to check validation
	if _, err := db.Get(root); err != nil {
		return err
	}

	m.committedHash = root
	m.root = hash(root)
	m.db = db
	return nil
}

// TODO: proper error
func (m *mpt) Reset(immutable trie.Immutable) error {
	immutableTrie, ok := immutable.(*mpt)
	if ok == false {
		return nil
	}

	requestPool := make(map[string]trieValue)
	// This immutableTrie is reused to another mutable trie.
	// So data in requestPool has to be copied to mutable trie's request pool
	for snapshot := immutableTrie; snapshot != nil; snapshot = snapshot.prevSnapshot {
		for k, v := range snapshot.requestPool {
			if requestPool[k] == nil {
				requestPool[k] = v
			}
		}
	}

	m.requestPool = requestPool
	m.committedHash = make([]byte, len(immutableTrie.committedHash))
	copy(m.committedHash, immutableTrie.committedHash)
	rootHash := make([]byte, len(immutableTrie.committedHash))
	copy(rootHash, immutableTrie.committedHash)
	m.root = hash(rootHash)
	m.db = immutableTrie.db
	return nil
}

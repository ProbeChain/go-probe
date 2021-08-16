// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package trie implements Merkle Patricia Tries.
package trie

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/edsrzf/mmap-go"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/keycard-go/hexutils"
	"os"
	"reflect"
	"sort"
	"sync"
	"unsafe"
)

var (
	// emptyRoot is the known root hash of an empty trie.
	emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	// emptyState is the known hash of an empty state trie entry.
	emptyState = crypto.Keccak256Hash(nil)

	//
	maxBinaryLeafLen = 1000
)

// LeafCallback is a callback type invoked when a trie operation reaches a leaf
// node.
//
// The paths is a path tuple identifying a particular trie node either in a single
// trie (account) or a layered trie (account -> storage). Each path in the tuple
// is in the raw format(32 bytes).
//
// The hexpath is a composite hexary path identifying the trie node. All the key
// bytes are converted to the hexary nibbles and composited with the parent path
// if the trie node is in a layered trie.
//
// It's used by state sync and commit to allow handling external references
// between account and storage tries. And also it's used in the state healing
// for extracting the raw states(leaf nodes) with corresponding paths.
type LeafCallback func(paths [][]byte, hexpath []byte, leaf []byte, parent common.Hash) error

// unique 对一个已排序的数组去重
func unique(a []int) []int {
	length := len(a)
	if length == 0 {
		return a
	}
	i := 0
	for j := 1; j < length; j++ {
		if a[i] != a[j] {
			i++
			a[i] = a[j]
		}
	}
	return a[:i+1]
}

func intToBytes(data uint32) []byte {
	bytebuf := bytes.NewBuffer([]byte{})
	binary.Write(bytebuf, binary.BigEndian, data)
	return bytebuf.Bytes()
}

// Trie is a Merkle Patricia Trie.
// The zero value is an empty trie with no database.
// Use New to create a trie that sits on top of a database.
//
// Trie is not safe for concurrent use.
type Trie struct {
	db   *Database
	root node
	// Keep track of the number leafs which have been inserted since the last
	// hashing operation. This number will not directly map to the number of
	// actually unhashed nodes
	unhashed int

	mem             mmap.MMap		 // MMap
	f               *os.File		 // 文件指针
	depth           int              // 二叉树深度
	unhashedIndex   []int            // 需要重新计算的叶子节点的索引
	uncommitedIndex []int            // 需要提交的叶子节点索引
	binaryHashNodes []binaryHashNode // 二叉节点
	binaryLeafs     []binaryLeaf     // 叶子节点
}

// newFlag returns the cache flag value for a newly created node.
func (t *Trie) newFlag() nodeFlag {
	return nodeFlag{dirty: true}
}

// New creates a trie with an existing root node from db.
//
// If root is the zero hash or the sha3 hash of an empty string, the
// trie is initially empty and does not require a database. Otherwise,
// New will panic if db is nil and returns a MissingNodeError if root does
// not exist in the database. Accessing the trie loads nodes from db on demand.
func New(root common.Hash, db *Database) (*Trie, error) {
	if db == nil {
		panic("trie.New called without a database")
	}
	trie := &Trie{
		db:    db,
	}
	db.trie = trie
	if root != (common.Hash{}) && root != emptyRoot {
		rootnode, err := trie.resolveHash(root[:], nil)
		if err != nil {
			return nil, err
		}
		trie.root = rootnode
	}
	return trie, nil
}

var instanceTrie *Trie
var once sync.Once
// NewBinary creates a binary trie.
func NewBinary(root common.Hash, db *Database, depth int) (*Trie, error) {
	once.Do(func() {
		if db == nil {
			panic("trie.New called without a database")
		}
		trie := &Trie{
			db:    db,
			depth: depth,
		}

		length := int(math.BigPow(2, int64(depth+1)).Int64()) - 1
		nBytes := length * int(unsafe.Sizeof(binaryHashNode{}))
		trie.binaryHashNodes = make([]binaryHashNode, length, length)
		trie.binaryLeafs = make([]binaryLeaf, length/2+1, length/2+1)

		var f *os.File
		var err error
		init := false
		//usr, _ := user.Current()
		triePath := "C:\\Users\\lcq\\go\\src\\github.com\\ethereum\\go-ethereum\\build\\bin\\data\\geth\\trie.bin" //filepath.Join(usr.HomeDir, "AppData", "Local", "Trie", "trie.bin")
		_, err = os.Lstat(triePath)
		if os.IsNotExist(err) {
			f, err = os.Create(triePath)
			if err == nil {
				err = f.Truncate(int64(nBytes))
				init = true
			}
		} else {
			f, err = os.OpenFile(triePath, os.O_RDWR, 0644)
		}

		if err == nil {
			trie.f = f
			mem, err := mmap.Map(f, os.O_RDWR, 0)
			if err == nil {
				shDst := (*reflect.SliceHeader)(unsafe.Pointer(&trie.binaryHashNodes))
				shDst.Data = uintptr(unsafe.Pointer(&mem[0]))
				shDst.Len = length
				shDst.Cap = length
				trie.mem = mem
			} else {
				//return nil, errors.New("mmap.Map fail")
			}
		} else {
			//return nil, errors.New("trie file error")
		}
		if init && (root == (common.Hash{}) || root == emptyRoot) {
			// depth == 21 耗时 140ms
			// depth == 20 耗时 60ms
			curDepth := depth
			for curDepth >= 0 {
				start := math.BigPow(2, int64(curDepth)).Int64() - 1 // 左闭
				end := math.BigPow(2, int64(curDepth+1)).Int64() - 1 // 右开

				var curHash []byte
				if curDepth == trie.depth {
					curHash = crypto.Keccak256(nil) // 下面没挂元素用空值计算哈希
				} else {
					data := make([]byte, 66, 66)
					copy(data, trie.binaryHashNodes[end].Hash[:])
					data = bytes.Repeat(data, 2)
					curHash = crypto.Keccak256(data)
				}
				//curHash = []byte{0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41}
				var hash [32]byte
				copy(hash[:], curHash)
				for i := start; i < end; i += 1 {
					trie.binaryHashNodes[i] = binaryHashNode{hash, uint32(0)}
				}
				curDepth -= 1
			}
			trie.mem.Flush()
		}
		trie.root = trie.binaryHashNodes[0]
		instanceTrie = trie
	})
	instanceTrie.db = db 	// 创世块初始化的数据库与后续出块的数据库不是同一个数据库
	tr := instanceTrie 		// 仅仅为了调试方便
	db.trie = tr
	return tr, nil
}

func (t *Trie) Close() {
	if t.mem != nil {
		t.mem.Unmap()
	}

	if t.f != nil {
		t.f.Close()
	}
}

// binary 是否是一个二叉默克尔树
func (t *Trie) binary() bool {
	return t.depth > 0
}

// relatedIndexs 将key相关的binaryHashNodes，binaryLeafs索引全部返回来，数组最后一个是binaryLeafs受影响的
func (t *Trie) relatedIndexs(key []byte) ([]int, int) {
	k := keybytesToHex(key)
	index := 0
	leafIndex := -1
	indexs := []int{index}
	for i, b := range k {
		index = 2*index + 1
		// @todo 需要拆成二进制
		if b >= 8 {
			index += 1 // 索引往右边走
		}
		indexs = append(indexs, index)
		if i+1 == t.depth {
			// 最后一个 binaryLeafs 对应的索引
			leafIndex = index - int(math.BigPow(2, int64(t.depth)).Int64()-1)
			break
		}
	}

	return indexs, leafIndex
}

// uniqueSortUnhashedIndex 对需要重新计算的叶子节点的索引进行排序去重
func (t *Trie) uniqueSortUnhashedIndex() {
	if len(t.unhashedIndex) > 0 {
		sort.Ints(t.unhashedIndex)
		t.unhashedIndex = unique(t.unhashedIndex)
	}
}

// NodeIterator returns an iterator that returns nodes of the trie. Iteration starts at
// the key after the given start key.
func (t *Trie) NodeIterator(start []byte) NodeIterator {
	return newNodeIterator(t, start)
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *Trie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryGet(key []byte) ([]byte, error) {
	if t.binary() {
		leaf := t.TryGetBinaryLeaf(key)
		for _, node := range leaf {
			if bytes.Compare(key, node.Key) == 0 {
				return node.Val, nil
			}
		}
		return nil, nil
	} else {
		value, newroot, didResolve, err := t.tryGet(t.root, keybytesToHex(key), 0)
		if err == nil && didResolve {
			t.root = newroot
		}
		return value, err
	}
}

func (t *Trie) tryGet(origNode node, key []byte, pos int) (value []byte, newnode node, didResolve bool, err error) {
	switch n := (origNode).(type) {
	case nil:
		return nil, nil, false, nil
	case valueNode:
		return n, n, false, nil
	case *shortNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {
			// key not found in trie
			return nil, n, false, nil
		}
		value, newnode, didResolve, err = t.tryGet(n.Val, key, pos+len(n.Key))
		if err == nil && didResolve {
			n = n.copy()
			n.Val = newnode
		}
		return value, n, didResolve, err
	case *fullNode:
		value, newnode, didResolve, err = t.tryGet(n.Children[key[pos]], key, pos+1)
		if err == nil && didResolve {
			n = n.copy()
			n.Children[key[pos]] = newnode
		}
		return value, n, didResolve, err
	case hashNode:
		child, err := t.resolveHash(n, key[:pos])
		if err != nil {
			return nil, n, true, err
		}
		value, newnode, _, err := t.tryGet(child, key, pos)
		return value, newnode, true, err
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
	}
}

// TryGetBinaryLeaf 根据key从内存中或者leveldb中返回叶子节点
func (t *Trie) TryGetBinaryLeaf(key []byte) []binaryNode {
	indexs, leafIndex := t.relatedIndexs(key)
	leaf := t.binaryLeafs[leafIndex]
	// 此时需要从数据库中加载一次
	if leaf == nil {
		hash := t.binaryHashNodes[indexs[len(indexs)-1]].Hash
		sliceHash := make([]byte, 32, 32)
		copy(sliceHash, hash[:])
		node, _ := t.resolveHash(sliceHash, nil)
		if node != nil {
			switch n := (node).(type) {
			case binaryLeaf:
				leaf = n
			default:
				panic(fmt.Sprintf("%T: invalid node: %v", node, node))
			}
		} else {
			leaf = make([]binaryNode, 0) // 创建一个空切片，防止下次再次去数据库搜索
		}
		t.binaryLeafs[leafIndex] = leaf
	}
	return leaf
}

// TryGetNode attempts to retrieve a trie node by compact-encoded path. It is not
// possible to use keybyte-encoding as the path might contain odd nibbles.
func (t *Trie) TryGetNode(path []byte) ([]byte, int, error) {
	item, newroot, resolved, err := t.tryGetNode(t.root, compactToHex(path), 0)
	if err != nil {
		return nil, resolved, err
	}
	if resolved > 0 {
		t.root = newroot
	}
	if item == nil {
		return nil, resolved, nil
	}
	return item, resolved, err
}

func (t *Trie) tryGetNode(origNode node, path []byte, pos int) (item []byte, newnode node, resolved int, err error) {
	// If we reached the requested path, return the current node
	if pos >= len(path) {
		// Although we most probably have the original node expanded, encoding
		// that into consensus form can be nasty (needs to cascade down) and
		// time consuming. Instead, just pull the hash up from disk directly.
		var hash hashNode
		if node, ok := origNode.(hashNode); ok {
			hash = node
		} else {
			hash, _ = origNode.cache()
		}
		if hash == nil {
			return nil, origNode, 0, errors.New("non-consensus node")
		}
		blob, err := t.db.Node(common.BytesToHash(hash))
		return blob, origNode, 1, err
	}
	// Path still needs to be traversed, descend into children
	switch n := (origNode).(type) {
	case nil:
		// Non-existent path requested, abort
		return nil, nil, 0, nil

	case valueNode:
		// Path prematurely ended, abort
		return nil, nil, 0, nil

	case *shortNode:
		if len(path)-pos < len(n.Key) || !bytes.Equal(n.Key, path[pos:pos+len(n.Key)]) {
			// Path branches off from short node
			return nil, n, 0, nil
		}
		item, newnode, resolved, err = t.tryGetNode(n.Val, path, pos+len(n.Key))
		if err == nil && resolved > 0 {
			n = n.copy()
			n.Val = newnode
		}
		return item, n, resolved, err

	case *fullNode:
		item, newnode, resolved, err = t.tryGetNode(n.Children[path[pos]], path, pos+1)
		if err == nil && resolved > 0 {
			n = n.copy()
			n.Children[path[pos]] = newnode
		}
		return item, n, resolved, err

	case hashNode:
		child, err := t.resolveHash(n, path[:pos])
		if err != nil {
			return nil, n, 1, err
		}
		item, newnode, resolved, err := t.tryGetNode(child, path, pos)
		return item, newnode, resolved + 1, err

	default:
		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
	}
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *Trie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryUpdate associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
//
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryUpdate(key, value []byte) error {
	log.Info("trie TryUpdate", "key", hexutils.BytesToHex(key), "value", hexutils.BytesToHex(value))
	if t.binary() {
		_, leafIndex := t.relatedIndexs(key)
		leaf := t.TryGetBinaryLeaf(key)
		find := false
		insertIndex := 0

		// @todo 此处可以使用二分法加快搜索
		for i, node := range leaf {
			ret := bytes.Compare(key, node.Key)
			if ret == 0 {
				leaf[i].Val = value
				find = true
				t.unhashedIndex = append(t.unhashedIndex, leafIndex)
			} else if ret < 0 {
				insertIndex = i
			} else {
				break
			}
		}
		if !find {
			// 按照key的顺序插入进去方便进行哈希计算
			length := len(leaf)
			if length == 0 {
				t.binaryLeafs[leafIndex] = append(leaf, binaryNode{key, value})
				t.unhashedIndex = append(t.unhashedIndex, leafIndex)
			} else if length > maxBinaryLeafLen {
				return errors.New("exceed max binary leaf size")
			} else {
				left := leaf[:insertIndex+1]
				right := append([]binaryNode{{key, value}}, leaf[insertIndex+1:]...)
				t.binaryLeafs[leafIndex] = append(left, right...)
				t.unhashedIndex = append(t.unhashedIndex, leafIndex)
			}
		}
	} else {
		t.unhashed++
		k := keybytesToHex(key)
		if len(value) != 0 {
			_, n, err := t.insert(t.root, nil, k, valueNode(value))
			if err != nil {
				return err
			}
			t.root = n
		} else {
			_, n, err := t.delete(t.root, nil, k)
			if err != nil {
				return err
			}
			t.root = n
		}
	}

	return nil
}

func (t *Trie) insert(n node, prefix, key []byte, value node) (bool, node, error) {
	if len(key) == 0 {
		if v, ok := n.(valueNode); ok {
			return !bytes.Equal(v, value.(valueNode)), value, nil
		}
		return true, value, nil
	}
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key, n.Key)
		// If the whole key matches, keep this short node as is
		// and only update the value.
		if matchlen == len(n.Key) {
			dirty, nn, err := t.insert(n.Val, append(prefix, key[:matchlen]...), key[matchlen:], value)
			if !dirty || err != nil {
				return false, n, err
			}
			return true, &shortNode{n.Key, nn, t.newFlag()}, nil
		}
		// Otherwise branch out at the index where they differ.
		branch := &fullNode{flags: t.newFlag()}
		var err error
		_, branch.Children[n.Key[matchlen]], err = t.insert(nil, append(prefix, n.Key[:matchlen+1]...), n.Key[matchlen+1:], n.Val)
		if err != nil {
			return false, nil, err
		}
		_, branch.Children[key[matchlen]], err = t.insert(nil, append(prefix, key[:matchlen+1]...), key[matchlen+1:], value)
		if err != nil {
			return false, nil, err
		}
		// Replace this shortNode with the branch if it occurs at index 0.
		if matchlen == 0 {
			return true, branch, nil
		}
		// Otherwise, replace it with a short node leading up to the branch.
		return true, &shortNode{key[:matchlen], branch, t.newFlag()}, nil

	case *fullNode:
		dirty, nn, err := t.insert(n.Children[key[0]], append(prefix, key[0]), key[1:], value)
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = t.newFlag()
		n.Children[key[0]] = nn
		return true, n, nil

	case nil:
		return true, &shortNode{key, value, t.newFlag()}, nil

	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and insert into it. This leaves all child nodes on
		// the path to the value in the trie.
		rn, err := t.resolveHash(n, prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := t.insert(rn, prefix, key, value)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil

	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

// Delete removes any existing value for key from the trie.
func (t *Trie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryDelete(key []byte) error {
	if t.binary() {
		_, leafIndex := t.relatedIndexs(key)
		leaf := t.binaryLeafs[leafIndex]
		for i, node := range leaf {
			if bytes.Compare(key, node.Key) == 0 {
				t.binaryLeafs[leafIndex] = append(leaf[:i], leaf[i+1:]...)
				t.unhashedIndex = append(t.unhashedIndex, leafIndex)
				return nil
			}
		}
		return errors.New("not found")
	} else {
		t.unhashed++
		k := keybytesToHex(key)
		_, n, err := t.delete(t.root, nil, k)
		if err != nil {
			return err
		}
		t.root = n
	}
	return nil
}

// delete returns the new root of the trie with key deleted.
// It reduces the trie to minimal form by simplifying
// nodes on the way up after deleting recursively.
func (t *Trie) delete(n node, prefix, key []byte) (bool, node, error) {
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key, n.Key)
		if matchlen < len(n.Key) {
			return false, n, nil // don't replace n on mismatch
		}
		if matchlen == len(key) {
			return true, nil, nil // remove n entirely for whole matches
		}
		// The key is longer than n.Key. Remove the remaining suffix
		// from the subtrie. Child can never be nil here since the
		// subtrie must contain at least two other values with keys
		// longer than n.Key.
		dirty, child, err := t.delete(n.Val, append(prefix, key[:len(n.Key)]...), key[len(n.Key):])
		if !dirty || err != nil {
			return false, n, err
		}
		switch child := child.(type) {
		case *shortNode:
			// Deleting from the subtrie reduced it to another
			// short node. Merge the nodes to avoid creating a
			// shortNode{..., shortNode{...}}. Use concat (which
			// always creates a new slice) instead of append to
			// avoid modifying n.Key since it might be shared with
			// other nodes.
			return true, &shortNode{concat(n.Key, child.Key...), child.Val, t.newFlag()}, nil
		default:
			return true, &shortNode{n.Key, child, t.newFlag()}, nil
		}

	case *fullNode:
		dirty, nn, err := t.delete(n.Children[key[0]], append(prefix, key[0]), key[1:])
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = t.newFlag()
		n.Children[key[0]] = nn

		// Because n is a full node, it must've contained at least two children
		// before the delete operation. If the new child value is non-nil, n still
		// has at least two children after the deletion, and cannot be reduced to
		// a short node.
		if nn != nil {
			return true, n, nil
		}
		// Reduction:
		// Check how many non-nil entries are left after deleting and
		// reduce the full node to a short node if only one entry is
		// left. Since n must've contained at least two children
		// before deletion (otherwise it would not be a full node) n
		// can never be reduced to nil.
		//
		// When the loop is done, pos contains the index of the single
		// value that is left in n or -2 if n contains at least two
		// values.
		pos := -1
		for i, cld := range &n.Children {
			if cld != nil {
				if pos == -1 {
					pos = i
				} else {
					pos = -2
					break
				}
			}
		}
		if pos >= 0 {
			if pos != 16 {
				// If the remaining entry is a short node, it replaces
				// n and its key gets the missing nibble tacked to the
				// front. This avoids creating an invalid
				// shortNode{..., shortNode{...}}.  Since the entry
				// might not be loaded yet, resolve it just for this
				// check.
				cnode, err := t.resolve(n.Children[pos], prefix)
				if err != nil {
					return false, nil, err
				}
				if cnode, ok := cnode.(*shortNode); ok {
					k := append([]byte{byte(pos)}, cnode.Key...)
					return true, &shortNode{k, cnode.Val, t.newFlag()}, nil
				}
			}
			// Otherwise, n is replaced by a one-nibble short node
			// containing the child.
			return true, &shortNode{[]byte{byte(pos)}, n.Children[pos], t.newFlag()}, nil
		}
		// n still contains at least two values and cannot be reduced.
		return true, n, nil

	case valueNode:
		return true, nil, nil

	case nil:
		return false, nil, nil

	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and delete from it. This leaves all child nodes on
		// the path to the value in the trie.
		rn, err := t.resolveHash(n, prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := t.delete(rn, prefix, key)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil

	default:
		panic(fmt.Sprintf("%T: invalid node: %v (%v)", n, n, key))
	}
}

func concat(s1 []byte, s2 ...byte) []byte {
	r := make([]byte, len(s1)+len(s2))
	copy(r, s1)
	copy(r[len(s1):], s2)
	return r
}

func (t *Trie) resolve(n node, prefix []byte) (node, error) {
	if n, ok := n.(hashNode); ok {
		return t.resolveHash(n, prefix)
	}
	return n, nil
}

func (t *Trie) resolveHash(n hashNode, prefix []byte) (node, error) {
	hash := common.BytesToHash(n)
	if node := t.db.node(hash); node != nil {
		return node, nil
	}
	return nil, &MissingNodeError{NodeHash: hash, Path: prefix}
}

// Hash returns the root hash of the trie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (t *Trie) Hash() common.Hash {
	if t.binary() {
		// @todo 如果待计算的个数较多可使用多线程计算
		t.uniqueSortUnhashedIndex()
		unhashedIndex := make([]int, 0)

		// 计算叶子节点的哈希值
		for _, index := range t.unhashedIndex {
			num := len(t.binaryLeafs[index])

			binaryNodeIndex := index + int(math.BigPow(2, int64(t.depth)).Int64()-1) // 对应哈希节点的索引值
			t.binaryHashNodes[binaryNodeIndex].Num = uint32(num)
			t.binaryHashNodes[binaryNodeIndex].Hash = t.binaryLeafs[index].Hash()

			// 哈希节点已更新，将这个哈希节点的索引值放到数组中开始从下往上计算哈希值
			unhashedIndex = append(unhashedIndex, (binaryNodeIndex-1)/2)
		}

		// 由于计算叶子节点的哈希是从左至右的那么肯定是排序好的，但是最后一层哈希节点可能指向相同的父亲所以要去重防止重复计算
		unhashedIndex = unique(unhashedIndex)
		for {
			temp := make([]int, 0) // 下次迭代需要计算的哈希节点索引列表
			for _, index := range unhashedIndex {
				li := index*2 + 1
				ri := index*2 + 2
				var lHash []byte
				var rHash []byte
				copy(lHash, t.binaryHashNodes[li].Hash[:])
				copy(rHash, t.binaryHashNodes[ri].Hash[:])
				lNum := t.binaryHashNodes[li].Num
				rNum := t.binaryHashNodes[ri].Num
				t.binaryHashNodes[index].Num = lNum + rNum
				// @todo 数字转byte需要分别处理大端小端的问题
				curHash := crypto.Keccak256(concat(concat(lHash, intToBytes(lNum)...), concat(rHash, intToBytes(rNum)...)...))
				copy(t.binaryHashNodes[index].Hash[:], curHash)

				// 如果是最后一个，那就不要反复计算了
				if index > 0 {
					temp = append(temp, (index-1)/2)
				}
			}
			temp = unique(temp)
			unhashedIndex = temp[:]
			if len(unhashedIndex) == 0 {
				t.root = t.binaryHashNodes[0]
				break
			}
		}
		t.uncommitedIndex = append(t.uncommitedIndex, t.unhashedIndex...)
		t.unhashedIndex = make([]int, 0)

		var hash []byte
		copy(hash, t.binaryHashNodes[0].Hash[:])
		hash = crypto.Keccak256(concat(hash, intToBytes(t.binaryHashNodes[0].Num)...))
		return common.BytesToHash(hash)
	} else {
		// 返回的cached是将算过的哈希值保存到nodeFlag中防止下次计算哈希需要重新计算，其他的值均保持不变
		hash, cached, _ := t.hashRoot()
		t.root = cached
		return common.BytesToHash(hash.(hashNode))
	}
}

// Commit writes all nodes to the trie's memory database, tracking the internal
// and external (for account tries) references.
func (t *Trie) Commit(onleaf LeafCallback) (root common.Hash, err error) {
	if t.db == nil {
		panic("commit called on trie with nil database")
	}
	if t.root == nil {
		return emptyRoot, nil
	}
	// Derive the hash for all dirty nodes first. We hold the assumption
	// in the following procedure that all nodes are hashed.
	rootHash := t.Hash()

	// 直接将kv对往内存数据库提交
	if t.binary() {
		if len(t.uncommitedIndex) > 0 {
			sort.Ints(t.uncommitedIndex)
			t.uncommitedIndex = unique(t.uncommitedIndex)
		}
		for _, index := range t.uncommitedIndex {
			hash := make([]byte, 32, 32)
			binaryNodeIndex := index + int(math.BigPow(2, int64(t.depth)).Int64()-1) // 对应哈希节点的索引值
			copy(hash, t.binaryHashNodes[binaryNodeIndex].Hash[:])
			t.db.insert(common.BytesToHash(hash), estimateSize(t.binaryLeafs[index]), t.binaryLeafs[index])
		}
		// 最后提交一个总的哈希，也就是哈希节点的第一个值，一来用于判断创世块是否存在。二来可以统计每个区块上面存在的账号数据
		var hash []byte
		copy(hash, t.binaryHashNodes[0].Hash[:])
		hash = crypto.Keccak256(concat(hash, intToBytes(t.binaryHashNodes[0].Num)...))
		t.db.insert(common.BytesToHash(hash), estimateSize(t.binaryHashNodes[0]), t.binaryHashNodes[0])

		t.uncommitedIndex = make([]int, 0)
		t.mem.Flush()

		return rootHash, nil
	}

	h := newCommitter()
	defer returnCommitterToPool(h)

	// Do a quick check if we really need to commit, before we spin
	// up goroutines. This can happen e.g. if we load a trie for reading storage
	// values, but don't write to it.
	if _, dirty := t.root.cache(); !dirty {
		return rootHash, nil
	}
	var wg sync.WaitGroup
	if onleaf != nil {
		h.onleaf = onleaf
		h.leafCh = make(chan *leaf, leafChanSize)
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.commitLoop(t.db)
		}()
	}
	var newRoot hashNode
	newRoot, err = h.Commit(t.root, t.db)
	if onleaf != nil {
		// The leafch is created in newCommitter if there was an onleaf callback
		// provided. The commitLoop only _reads_ from it, and the commit
		// operation was the sole writer. Therefore, it's safe to close this
		// channel here.
		close(h.leafCh)
		wg.Wait()
	}
	if err != nil {
		return common.Hash{}, err
	}
	t.root = newRoot
	return rootHash, nil
}

// hashRoot calculates the root hash of the given trie
func (t *Trie) hashRoot() (node, node, error) {
	if t.root == nil {
		return hashNode(emptyRoot.Bytes()), nil, nil
	}
	// If the number of changes is below 100, we let one thread handle it
	h := newHasher(t.unhashed >= 100)
	defer returnHasherToPool(h)
	hashed, cached := h.hash(t.root, true)
	t.unhashed = 0
	return hashed, cached, nil
}

// Reset drops the referenced root node and cleans all internal state.
func (t *Trie) Reset() {
	t.root = nil
	t.unhashed = 0
}

func (t *Trie) GetKey(key []byte) []byte {
	return key
}

func (t *Trie) Copy() *Trie {
	cpy := *t
	return &cpy
}

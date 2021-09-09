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
	"path"
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

	rollBackMaxStep = 10 // 能够回滚的最大步数
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

func bytesToInt(b []byte) int {
	b = append([]byte{0, 0, 0, 0}, b...)
	b = b[len(b)-4:]
	bytesBuffer := bytes.NewBuffer(b)
	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return int(x)
}

type DiffLeaf struct {
	Index uint32     // 被修改的叶子节点索引
	Leaf  binaryLeaf // 叶子节点数据
}

type Alter struct {
	commit    bool       // 是否已提交
	PreRoot   [32]byte   // 上一个状态树
	CurRoot   [32]byte   // 当前状态书
	DiffLeafs []DiffLeaf // 上一个状态树到当前状态树被
}

type BinaryTree struct {
	curDiffLeafs    []DiffLeaf       // 当前修改
	alters          []Alter          // 修改历史
	mem             mmap.MMap        // MMap
	f               *os.File         // 文件指针
	depth           int              // 二叉树深度
	unhashedIndex   []int            // 需要重新计算的叶子节点的索引
	uncommitedIndex []int            // 需要提交的叶子节点索引
	binaryHashNodes []binaryHashNode // 二叉节点
	binaryLeafs     []binaryLeaf     // 叶子节点
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
	bt       *BinaryTree
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
		db: db,
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

var normalBtOnce sync.Once
var normalBt *BinaryTree

// NewNormalBinary creates a normal account binary trie.
func NewNormalBinary(root common.Hash, db *Database) (*Trie, error) {
	normalBtOnce.Do(func() {
		normalBt = initBinaryTree(root, db, 2, "./data/geth/trie.bin")
	})
	return newBinary(root, db, normalBt)
}

// newBinary creates a binary trie.
func newBinary(root common.Hash, db *Database, bt *BinaryTree) (*Trie, error) {
	if db == nil {
		panic("trie.New called without a database")
	}
	trie := &Trie{
		db: db,
		bt: bt,
	}

	db.trie = trie // 后续数据库需要根据MPT或者是BMPT进行数据的读取
	curRoot := trie.binaryRoot()

	// 如果不是要找回当前的root，那么尝试去恢复一下，如果能恢复成功，认为这颗BMPT存在，否则返回错误
	if !(root == curRoot || root == (common.Hash{}) || root == emptyRoot) {
		node, err := trie.resolveHash(root[:], nil)
		if err == nil {
			trie.root = node
			return trie, err
		} else {
			return nil, err
		}
	} else {
		trie.root = trie.bt.binaryHashNodes[0]
	}
	return trie, nil
}

// initBinaryTree init a binary tree.
func initBinaryTree(root common.Hash, db *Database, depth int, triePath string) *BinaryTree {
	binaryTree := &BinaryTree{
		depth: depth,
	}

	length := int(math.BigPow(2, int64(depth+1)).Int64()) - 1
	nBytes := length * int(unsafe.Sizeof(binaryHashNode{}))
	binaryTree.binaryHashNodes = make([]binaryHashNode, length, length)
	binaryTree.binaryLeafs = make([]binaryLeaf, length/2+1, length/2+1)

	var f *os.File
	var err error
	init := false
	_, err = os.Lstat(triePath)
	if os.IsNotExist(err) {
		dir, _ := path.Split(triePath)
		os.MkdirAll(dir, os.ModePerm)
		f, err = os.Create(triePath)
		if err == nil {
			err = f.Truncate(int64(nBytes))
			init = true
		}
	} else {
		f, err = os.OpenFile(triePath, os.O_RDWR, 0644)
	}

	if err == nil {
		binaryTree.f = f
		mem, err := mmap.Map(f, os.O_RDWR, 0)
		if err == nil {
			shDst := (*reflect.SliceHeader)(unsafe.Pointer(&binaryTree.binaryHashNodes))
			shDst.Data = uintptr(unsafe.Pointer(&mem[0]))
			shDst.Len = length
			shDst.Cap = length
			binaryTree.mem = mem
		} else {
			panic("mmap.Map fail")
		}
	} else {
		panic("trie open file error")
	}
	if init && (root == (common.Hash{}) || root == emptyRoot) {
		// depth == 21 耗时 140ms
		// depth == 20 耗时 60ms
		curDepth := depth
		for curDepth >= 0 {
			start := math.BigPow(2, int64(curDepth)).Int64() - 1 // 左闭
			end := math.BigPow(2, int64(curDepth+1)).Int64() - 1 // 右开

			var curHash []byte
			if curDepth == binaryTree.depth {
				curHash = crypto.Keccak256(nil) // 下面没挂元素用空值计算哈希
			} else {
				data := make([]byte, 66, 66)
				copy(data, binaryTree.binaryHashNodes[end].Hash[:])
				data = bytes.Repeat(data, 2)
				curHash = crypto.Keccak256(data)
			}
			//curHash = []byte{0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41}
			var hash [32]byte
			copy(hash[:], curHash)
			for i := start; i < end; i += 1 {
				binaryTree.binaryHashNodes[i] = binaryHashNode{hash, uint32(0)}
			}
			curDepth -= 1
		}
		binaryTree.Flush()
	}

	binaryTree.alters = db.resolveAlters(binaryTree.binaryHashNodes[0].CalcHash())
	binaryTree.curDiffLeafs = make([]DiffLeaf, 0, 0)

	return binaryTree
}

func (t *Trie) Close() {
	if t.bt != nil && t.bt.mem != nil {
		t.bt.mem.Unmap()
	}
	if t.bt != nil && t.bt.f != nil {
		t.bt.f.Close()
	}
}

func (t *Trie) Flush() {
	if t.bt != nil && t.bt.mem != nil {
		t.bt.Flush()
		//t.print()
	}
}
func (bt *BinaryTree) Flush() {
	if bt.mem != nil {
		bt.mem.Flush()
		//bt.print()
	}
}

// Binary 是否是一个二叉默克尔树
func (t *Trie) Binary() bool {
	if t.bt != nil {
		return t.bt.depth > 0
	}
	return false
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
		if i+1 == t.bt.depth {
			// 最后一个 binaryLeafs 对应的索引
			leafIndex = index - int(math.BigPow(2, int64(t.bt.depth)).Int64()-1)
			break
		}
	}

	return indexs, leafIndex
}

// uniqueSortUnhashedIndex 对需要重新计算的叶子节点的索引进行排序去重
func (t *Trie) uniqueSortUnhashedIndex() {
	if len(t.bt.unhashedIndex) > 0 {
		sort.Ints(t.bt.unhashedIndex)
		t.bt.unhashedIndex = unique(t.bt.unhashedIndex)
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
	if t.Binary() {
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
func (t *Trie) TryGetBinaryLeaf(key []byte) binaryLeaf {
	indexs, leafIndex := t.relatedIndexs(key)
	binaryRoot := t.binaryRoot()
	trieRoot := t.trieRoot()
	leaf := t.bt.binaryLeafs[leafIndex]

	// 这是要查历史数据
	if binaryRoot != trieRoot {
		for _, alter := range t.bt.alters {
			if alter.PreRoot == trieRoot {
				for _, diffLeaf := range alter.DiffLeafs {
					if diffLeaf.Index == uint32(leafIndex) {
						return diffLeaf.Leaf
					}
				}
			}
		}
		leaf = make([]binaryNode, 0, 0)
	} else {
		// 此时需要从数据库中加载一次
		if leaf == nil {
			hash := t.bt.binaryHashNodes[indexs[len(indexs)-1]].Hash
			sliceHash := make([]byte, 32, 32)
			copy(sliceHash, hash[:])
			node, _ := t.resolveHash(sliceHash, nil)
			if node != nil {
				if data, ok := node.(binaryLeaf); ok {
					leaf = data
				} else {
					leaf = make([]binaryNode, 0)
				}
			} else {
				leaf = make([]binaryNode, 0) // 创建一个空切片，防止下次再次去数据库搜索
			}
			t.bt.binaryLeafs[leafIndex] = leaf
		}
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
	if t.Binary() {
		_, leafIndex := t.relatedIndexs(key)
		leaf := t.TryGetBinaryLeaf(key)
		find := false
		insertIndex := 0

		// 记录更改
		for i, diffLeaf := range t.bt.curDiffLeafs {
			if diffLeaf.Index == uint32(leafIndex) {
				t.bt.curDiffLeafs[i] = DiffLeaf{Index: uint32(leafIndex), Leaf: leaf.Copy()}
				find = true
				break
			}
		}
		if !find {
			t.bt.curDiffLeafs = append(t.bt.curDiffLeafs, DiffLeaf{Index: uint32(leafIndex), Leaf: leaf.Copy()})
		}

		find = false

		// @todo 此处可以使用二分法加快搜索
		for i, node := range leaf {
			ret := bytes.Compare(key, node.Key)
			if ret == 0 {
				leaf[i].Val = value
				find = true
				t.bt.unhashedIndex = append(t.bt.unhashedIndex, leafIndex)
				break
			} else if ret < 0 {
				insertIndex = i // 找到插入的位置了
				break
			}
			insertIndex = i + 1 // 如果是最后一个则插入到末尾
		}
		if !find {
			// 按照key的顺序插入进去方便进行哈希计算
			length := len(leaf)
			if length == 0 {
				t.bt.binaryLeafs[leafIndex] = append(leaf, binaryNode{key, value})
				t.bt.unhashedIndex = append(t.bt.unhashedIndex, leafIndex)
			} else if length > maxBinaryLeafLen {
				return errors.New("exceed max binary leaf size")
			} else {
				left := leaf[:insertIndex]
				right := append([]binaryNode{{key, value}}, leaf[insertIndex:]...)
				t.bt.binaryLeafs[leafIndex] = append(left, right...)
				t.bt.unhashedIndex = append(t.bt.unhashedIndex, leafIndex)
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
	if t.Binary() {
		_, leafIndex := t.relatedIndexs(key)
		leaf := t.bt.binaryLeafs[leafIndex]
		for i, node := range leaf {
			if bytes.Compare(key, node.Key) == 0 {
				t.bt.binaryLeafs[leafIndex] = append(leaf[:i], leaf[i+1:]...)
				t.bt.unhashedIndex = append(t.bt.unhashedIndex, leafIndex)
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
	if t.Binary() {
		preRoot := t.binaryRoot()

		// 没有需要计算的哈希索引直接返回上一次计算的结果
		if len(t.bt.unhashedIndex) == 0 {
			return preRoot
		}

		// @todo 如果待计算的个数较多可使用多线程计算
		t.uniqueSortUnhashedIndex()
		unhashedIndex := make([]int, 0)

		// 计算叶子节点的哈希值
		for _, index := range t.bt.unhashedIndex {
			num := len(t.bt.binaryLeafs[index])

			binaryNodeIndex := index + int(math.BigPow(2, int64(t.bt.depth)).Int64()-1) // 对应哈希节点的索引值
			t.bt.binaryHashNodes[binaryNodeIndex].Num = uint32(num)
			t.bt.binaryHashNodes[binaryNodeIndex].Hash = t.bt.binaryLeafs[index].Hash()

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
				lHash := make([]byte, 32, 32)
				rHash := make([]byte, 32, 32)
				copy(lHash, t.bt.binaryHashNodes[li].Hash[:])
				copy(rHash, t.bt.binaryHashNodes[ri].Hash[:])
				lNum := t.bt.binaryHashNodes[li].Num
				rNum := t.bt.binaryHashNodes[ri].Num
				t.bt.binaryHashNodes[index].Num = lNum + rNum
				// @todo 数字转byte需要分别处理大端小端的问题
				curHash := crypto.Keccak256(concat(concat(lHash, intToBytes(lNum)...), concat(rHash, intToBytes(rNum)...)...))
				copy(t.bt.binaryHashNodes[index].Hash[:], curHash)

				// 如果是最后一个，那就不要反复计算了
				if index > 0 {
					temp = append(temp, (index-1)/2)
				}
			}
			temp = unique(temp)
			unhashedIndex = temp[:]
			if len(unhashedIndex) == 0 {
				t.root = t.bt.binaryHashNodes[0]
				break
			}
		}
		// 纪录变化值
		curRoot := t.binaryRoot()

		t.bt.uncommitedIndex = append(t.bt.uncommitedIndex, t.bt.unhashedIndex...)
		t.bt.unhashedIndex = make([]int, 0)

		diffLeafs := make([]DiffLeaf, len(t.bt.curDiffLeafs), len(t.bt.curDiffLeafs))
		copy(diffLeafs, t.bt.curDiffLeafs)

		alter := Alter{
			PreRoot:   preRoot,
			CurRoot:   curRoot,
			DiffLeafs: diffLeafs,
		}
		t.bt.alters = append(t.bt.alters, alter)
		t.bt.curDiffLeafs = make([]DiffLeaf, 0, 0)

		return curRoot
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
	if t.Binary() {
		if len(t.bt.uncommitedIndex) > 0 {
			sort.Ints(t.bt.uncommitedIndex)
			t.bt.uncommitedIndex = unique(t.bt.uncommitedIndex)
		}
		for _, index := range t.bt.uncommitedIndex {
			hash := make([]byte, 32, 32)
			binaryNodeIndex := index + int(math.BigPow(2, int64(t.bt.depth)).Int64()-1) // 对应哈希节点的索引值
			copy(hash, t.bt.binaryHashNodes[binaryNodeIndex].Hash[:])
			t.db.insertLeaf(index, common.BytesToHash(hash), estimateSize(t.bt.binaryLeafs[index]), t.bt.binaryLeafs[index])
		}
		// 最后提交一个总的哈希，也就是哈希节点的第一个值，一来用于判断创世块是否存在。二来可以统计每个区块上面存在的账号数据
		t.db.insert(rootHash, estimateSize(t.bt.binaryHashNodes[0]), t.bt.binaryHashNodes[0])

		t.bt.uncommitedIndex = make([]int, 0)

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
	t.db.insert(common.BytesToHash(newRoot), 0, nil)
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

// RollBack roolback the data for the give hash
func (t *Trie) RollBack(rootHash common.Hash) error {
	alters := t.bt.alters
	find := false
	//查找是否具备有目标roothash的一棵树
	for _, alter := range alters {
		if alter.CurRoot == rootHash {
			find = true
		}
	}

	if find {
		t.bt.unhashedIndex = make([]int, 0, 0)
		//查找当前修改，进行逆序回滚
		for i := len(t.bt.curDiffLeafs) - 1; i >= 0; i-- {
			copy(t.bt.binaryLeafs[t.bt.curDiffLeafs[i].Index], t.bt.curDiffLeafs[i].Leaf)
			t.bt.unhashedIndex = append(t.bt.unhashedIndex, int(t.bt.curDiffLeafs[i].Index))
			t.bt.uncommitedIndex = append(t.bt.uncommitedIndex, int(t.bt.curDiffLeafs[i].Index))
		}

		// alters 排序
		t.sortAlter()

		//查找历史修改，进行逆序回滚
		for i := len(alters) - 1; i >= 0; i-- {
			for j := len(alters[i].DiffLeafs) - 1; j >= 0; j-- {
				copy(t.bt.binaryLeafs[alters[i].DiffLeafs[j].Index], alters[i].DiffLeafs[j].Leaf)
				t.bt.unhashedIndex = append(t.bt.unhashedIndex, int(alters[i].DiffLeafs[j].Index))
				t.bt.uncommitedIndex = append(t.bt.uncommitedIndex, int(alters[i].DiffLeafs[j].Index))
			}
			//回滚结束，删去修改记录，清零当前修改，且进行hash的重新计算
			if t.Hash() == rootHash {
				t.bt.alters = t.bt.alters[:i]
				t.bt.curDiffLeafs = make([]DiffLeaf, 0, 0)
				break
			}
		}
		// 排序去重  uncommitedIndex
		sort.Ints(t.bt.uncommitedIndex)
		unique(t.bt.uncommitedIndex)
	} else {
		return errors.New("no such rootHash")
	}
	return nil
}

func (t *Trie) sortAlter() {
	var alters []Alter
	if len(t.bt.alters) != 0 {
		alters = append(alters, t.bt.alters[0])
		t.bt.alters = t.bt.alters[1:]
	}

	for len(t.bt.alters) != 0 {
		preRoot := alters[0].PreRoot
		curRoot := alters[len(alters)-1].CurRoot
		length := len(alters)
		for i, alter := range t.bt.alters {
			if alter.PreRoot == curRoot {
				alters = append(alters, alter)
			} else if alter.CurRoot == preRoot {
				alters = append([]Alter{alter}, alters...)
			}
			if length+1 == len(alters) {
				t.bt.alters = append(t.bt.alters[:i], t.bt.alters[i+1:]...)
				break
			}
		}
	}
	t.bt.alters = alters
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

// binaryRoot calculates the BMPT root hash of the given trie
func (t *Trie) binaryRoot() common.Hash {
	if t.Binary() {
		hash := make([]byte, 32, 32+4)
		copy(hash, t.bt.binaryHashNodes[0].Hash[0:])
		hash = crypto.Keccak256(concat(hash, intToBytes(t.bt.binaryHashNodes[0].Num)...))
		return common.BytesToHash(hash)
	}
	return common.Hash{}
}

// trieRoot calculates the trie root hash of the given trie

func (t *Trie) trieRoot() common.Hash {
	if t.Binary() {
		if node, ok := t.root.(binaryHashNode); ok {
			hash := make([]byte, 32, 32+4)
			copy(hash, node.Hash[:])
			hash = crypto.Keccak256(concat(hash, intToBytes(node.Num)...))
			return common.BytesToHash(hash)
		} else {
			return common.Hash{}
		}
	}
	return common.Hash{}
}

// printTrie
func (t *Trie) print() {
	for i, node := range t.bt.binaryHashNodes {
		hash := make([]byte, 32, 32)
		copy(hash, node.Hash[:])
		log.Info("BinaryPrint hashNodes", "i", i, "hash", common.Bytes2Hex(hash), "num", node.Num)
	}
	for i, node := range t.bt.binaryLeafs {
		for j, n := range node {
			log.Info("BinaryPrint leafs", "i", i, "j", j, "Key", common.Bytes2Hex(n.Key), "Val", common.Bytes2Hex(n.Val))
		}
	}
}

func (bt *BinaryTree) print() {
	for i, node := range bt.binaryHashNodes {
		hash := make([]byte, 32, 32)
		copy(hash, node.Hash[:])
		log.Info("BinaryPrint hashNodes", "i", i, "hash", common.Bytes2Hex(hash), "num", node.Num)
	}
	for i, node := range bt.binaryLeafs {
		for j, n := range node {
			log.Info("BinaryPrint leafs", "i", i, "j", j, "Key", common.Bytes2Hex(n.Key), "Val", common.Bytes2Hex(n.Val))
		}
	}
}

func (alter *Alter) print() {
	log.Info("Alter", "PreRoot", common.Bytes2Hex(alter.PreRoot[:]), "CurRoot", common.Bytes2Hex(alter.CurRoot[:]))
	for _, diffLeaf := range alter.DiffLeafs {
		log.Info("Alter", "Index", diffLeaf.Index, "diffLeaf", diffLeaf.Leaf)
	}
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

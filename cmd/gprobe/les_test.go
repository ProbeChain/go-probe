package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/probeum/go-probeum/p2p"
	"github.com/probeum/go-probeum/rpc"
)

type gproberpc struct {
	name     string
	rpc      *rpc.Client
	gprobe   *testgprobe
	nodeInfo *p2p.NodeInfo
}

func (g *gproberpc) killAndWait() {
	g.gprobe.Kill()
	g.gprobe.WaitExit()
}

func (g *gproberpc) callRPC(result interface{}, method string, args ...interface{}) {
	if err := g.rpc.Call(&result, method, args...); err != nil {
		g.gprobe.Fatalf("callRPC %v: %v", method, err)
	}
}

func (g *gproberpc) addPeer(peer *gproberpc) {
	g.gprobe.Logf("%v.addPeer(%v)", g.name, peer.name)
	enode := peer.getNodeInfo().Enode
	peerCh := make(chan *p2p.PeerEvent)
	sub, err := g.rpc.Subscribe(context.Background(), "admin", peerCh, "peerEvents")
	if err != nil {
		g.gprobe.Fatalf("subscribe %v: %v", g.name, err)
	}
	defer sub.Unsubscribe()
	g.callRPC(nil, "admin_addPeer", enode)
	dur := 14 * time.Second
	timeout := time.After(dur)
	select {
	case ev := <-peerCh:
		g.gprobe.Logf("%v received event: type=%v, peer=%v", g.name, ev.Type, ev.Peer)
	case err := <-sub.Err():
		g.gprobe.Fatalf("%v sub error: %v", g.name, err)
	case <-timeout:
		g.gprobe.Error("timeout adding peer after", dur)
	}
}

// Use this function instead of `g.nodeInfo` directly
func (g *gproberpc) getNodeInfo() *p2p.NodeInfo {
	if g.nodeInfo != nil {
		return g.nodeInfo
	}
	g.nodeInfo = &p2p.NodeInfo{}
	g.callRPC(&g.nodeInfo, "admin_nodeInfo")
	return g.nodeInfo
}

func (g *gproberpc) waitSynced() {
	// Check if it's synced now
	var result interface{}
	g.callRPC(&result, "probe_syncing")
	syncing, ok := result.(bool)
	if ok && !syncing {
		g.gprobe.Logf("%v already synced", g.name)
		return
	}

	// Actually wait, subscribe to the event
	ch := make(chan interface{})
	sub, err := g.rpc.Subscribe(context.Background(), "probe", ch, "syncing")
	if err != nil {
		g.gprobe.Fatalf("%v syncing: %v", g.name, err)
	}
	defer sub.Unsubscribe()
	timeout := time.After(4 * time.Second)
	select {
	case ev := <-ch:
		g.gprobe.Log("'syncing' event", ev)
		syncing, ok := ev.(bool)
		if ok && !syncing {
			break
		}
		g.gprobe.Log("Other 'syncing' event", ev)
	case err := <-sub.Err():
		g.gprobe.Fatalf("%v notification: %v", g.name, err)
		break
	case <-timeout:
		g.gprobe.Fatalf("%v timeout syncing", g.name)
		break
	}
}

// ipcEndpoint resolves an IPC endpoint based on a configured value, taking into
// account the set data folders as well as the designated platform we're currently
// running on.
func ipcEndpoint(ipcPath, datadir string) string {
	// On windows we can only use plain top-level pipes
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(ipcPath, `\\.\pipe\`) {
			return ipcPath
		}
		return `\\.\pipe\` + ipcPath
	}
	// Resolve names into the data directory full paths otherwise
	if filepath.Base(ipcPath) == ipcPath {
		if datadir == "" {
			return filepath.Join(os.TempDir(), ipcPath)
		}
		return filepath.Join(datadir, ipcPath)
	}
	return ipcPath
}

// nextIPC ensures that each ipc pipe gets a unique name.
// On linux, it works well to use ipc pipes all over the filesystem (in datadirs),
// but windows require pipes to sit in "\\.\pipe\". Therefore, to run several
// nodes simultaneously, we need to distinguish between them, which we do by
// the pipe filename instead of folder.
var nextIPC = uint32(0)

func startGprobeWithIpc(t *testing.T, name string, args ...string) *gproberpc {
	ipcName := fmt.Sprintf("gprobe-%d.ipc", atomic.AddUint32(&nextIPC, 1))
	args = append([]string{"--networkid=42", "--port=0", "--ipcpath", ipcName}, args...)
	t.Logf("Starting %v with rpc: %v", name, args)

	g := &gproberpc{
		name:   name,
		gprobe: runGprobe(t, args...),
	}
	ipcpath := ipcEndpoint(ipcName, g.gprobe.Datadir)
	// We can't know exactly how long gprobe will take to start, so we try 10
	// times over a 5 second period.
	var err error
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if g.rpc, err = rpc.Dial(ipcpath); err == nil {
			return g
		}
	}
	t.Fatalf("%v rpc connect to %v: %v", name, ipcpath, err)
	return nil
}

func initGprobe(t *testing.T) string {
	args := []string{"--networkid=42", "init", "./testdata/clique.json"}
	t.Logf("Initializing gprobe: %v ", args)
	g := runGprobe(t, args...)
	datadir := g.Datadir
	g.WaitExit()
	return datadir
}

func startLightServer(t *testing.T) *gproberpc {
	datadir := initGprobe(t)
	t.Logf("Importing keys to gprobe")
	runGprobe(t, "--datadir", datadir, "--password", "./testdata/password.txt", "account", "import", "./testdata/key.prv", "--lightkdf").WaitExit()
	account := "0x02f0d131f1f97aef08aec6e3291b957d9efe7105"
	server := startGprobeWithIpc(t, "lightserver", "--allow-insecure-unlock", "--datadir", datadir, "--password", "./testdata/password.txt", "--unlock", account, "--mine", "--light.serve=100", "--light.maxpeers=1", "--nodiscover", "--nat=extip:127.0.0.1", "--verbosity=4")
	return server
}

func startClient(t *testing.T, name string) *gproberpc {
	datadir := initGprobe(t)
	return startGprobeWithIpc(t, name, "--datadir", datadir, "--nodiscover", "--syncmode=light", "--nat=extip:127.0.0.1", "--verbosity=4")
}

func TestPriorityClient(t *testing.T) {
	lightServer := startLightServer(t)
	defer lightServer.killAndWait()

	// Start client and add lightServer as peer
	freeCli := startClient(t, "freeCli")
	defer freeCli.killAndWait()
	freeCli.addPeer(lightServer)

	var peers []*p2p.PeerInfo
	freeCli.callRPC(&peers, "admin_peers")
	if len(peers) != 1 {
		t.Errorf("Expected: # of client peers == 1, actual: %v", len(peers))
		return
	}

	// Set up priority client, get its nodeID, increase its balance on the lightServer
	prioCli := startClient(t, "prioCli")
	defer prioCli.killAndWait()
	// 3_000_000_000 once we move to Go 1.13
	tokens := uint64(3000000000)
	lightServer.callRPC(nil, "les_addBalance", prioCli.getNodeInfo().ID, tokens)
	prioCli.addPeer(lightServer)

	// Check if priority client is actually syncing and the regular client got kicked out
	prioCli.callRPC(&peers, "admin_peers")
	if len(peers) != 1 {
		t.Errorf("Expected: # of prio peers == 1, actual: %v", len(peers))
	}

	nodes := map[string]*gproberpc{
		lightServer.getNodeInfo().ID: lightServer,
		freeCli.getNodeInfo().ID:     freeCli,
		prioCli.getNodeInfo().ID:     prioCli,
	}
	time.Sleep(1 * time.Second)
	lightServer.callRPC(&peers, "admin_peers")
	peersWithNames := make(map[string]string)
	for _, p := range peers {
		peersWithNames[nodes[p.ID].name] = p.ID
	}
	if _, freeClientFound := peersWithNames[freeCli.name]; freeClientFound {
		t.Error("client is still a peer of lightServer", peersWithNames)
	}
	if _, prioClientFound := peersWithNames[prioCli.name]; !prioClientFound {
		t.Error("prio client is not among lightServer peers", peersWithNames)
	}
}

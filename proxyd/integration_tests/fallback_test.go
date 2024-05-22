package integration_tests

import (
	"context"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/proxyd"
	ms "github.com/ethereum-optimism/optimism/proxyd/tools/mockserver/handler"
	"github.com/stretchr/testify/require"
)

// type nodeContext struct {
// 	backend     *proxyd.Backend   // this is the actual backend impl in proxyd
// 	mockBackend *MockBackend      // this is the fake backend that we can use to mock responses
// 	handler     *ms.MockedHandler // this is where we control the state of mocked responses
// }

func setup_v1(t *testing.T) (map[string]nodeContext, *proxyd.BackendGroup, *ProxydHTTPClient, func()) {
	// setup mock servers
	node1 := NewMockBackend(nil)
	node2 := NewMockBackend(nil)

	dir, err := os.Getwd()
	require.NoError(t, err)

	responses := path.Join(dir, "testdata/consensus_responses.yml")

	h1 := ms.MockedHandler{
		Overrides:    []*ms.MethodTemplate{},
		Autoload:     true,
		AutoloadFile: responses,
	}
	h2 := ms.MockedHandler{
		Overrides:    []*ms.MethodTemplate{},
		Autoload:     true,
		AutoloadFile: responses,
	}

	require.NoError(t, os.Setenv("NODE1_URL", node1.URL()))
	require.NoError(t, os.Setenv("NODE2_URL", node2.URL()))

	node1.SetHandler(http.HandlerFunc(h1.Handler))
	node2.SetHandler(http.HandlerFunc(h2.Handler))

	// setup proxyd
	config := ReadConfig("fallback")
	svr, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)

	// expose the proxyd client
	client := NewProxydClient("http://127.0.0.1:8545")

	// expose the backend group
	bg := svr.BackendGroups["node"]
	require.NotNil(t, bg)
	require.NotNil(t, bg.Consensus)
	require.Equal(t, 2, len(bg.Backends)) // should match config

	// convenient mapping to access the nodes by name
	nodes := map[string]nodeContext{
		"node1": {
			mockBackend: node1,
			backend:     bg.Backends[0],
			handler:     &h1,
		},
		"node2": {
			mockBackend: node2,
			backend:     bg.Backends[1],
			handler:     &h2,
		},
	}

	return nodes, bg, client, shutdown
}

func FallbackMode(t *testing.T) {
	nodes, bg, client, shutdown := setup(t)
	defer nodes["node1"].mockBackend.Close()
	defer nodes["node2"].mockBackend.Close()
	defer shutdown()

	ctx := context.Background()

	// poll for updated consensus
	update := func() {
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)
	}

	// convenient methods to manipulate state and mock responses
	reset := func() {
		for _, node := range nodes {
			node.handler.ResetOverrides()
			node.mockBackend.Reset()
		}
		bg.Consensus.ClearListeners()
		bg.Consensus.Reset()
	}

	override := func(node string, method string, block string, response string) {
		nodes[node].handler.AddOverride(&ms.MethodTemplate{
			Method:   method,
			Block:    block,
			Response: response,
		})
	}

	// overrideBlock := func(node string, blockRequest string, blockResponse string) {
	// 	override(node,
	// 		"eth_getBlockByNumber",
	// 		blockRequest,
	// 		buildResponse(map[string]string{
	// 			"number": blockResponse,
	// 			"hash":   "hash_" + blockResponse,
	// 		}))
	// }

	// overrideBlockHash := func(node string, blockRequest string, number string, hash string) {
	// 	override(node,
	// 		"eth_getBlockByNumber",
	// 		blockRequest,
	// 		buildResponse(map[string]string{
	// 			"number": number,
	// 			"hash":   hash,
	// 		}))
	// }

	overridePeerCount := func(node string, count int) {
		override(node, "net_peerCount", "", buildResponse(hexutil.Uint64(count).String()))
	}

	// overrideNotInSync := func(node string) {
	// 	override(node, "eth_syncing", "", buildResponse(map[string]string{
	// 		"startingblock": "0x0",
	// 		"currentblock":  "0x0",
	// 		"highestblock":  "0x100",
	// 	}))
	// }

	// force ban node2 and make sure node1 is the only one in consensus
	// useOnlyNode1 := func() {
	// 	overridePeerCount("node2", 0)
	// 	update()

	// 	consensusGroup := bg.Consensus.GetConsensusGroup()
	// 	require.Equal(t, 1, len(consensusGroup))
	// 	require.Contains(t, consensusGroup, nodes["node1"].backend)
	// 	nodes["node1"].mockBackend.Reset()
	// }

	// Ban Both nodes
	useNoNodes := func() {
		overridePeerCount("node2", 0)
		overridePeerCount("node1", 0)
		update()
		consensusGroup := bg.Consensus.GetConsensusGroup()
		require.Equal(t, 0, len(consensusGroup))
		require.NotContains(t, consensusGroup, nodes["node1"].backend)
		require.NotContains(t, consensusGroup, nodes["node2"].backend)
		nodes["node1"].mockBackend.Reset()
		nodes["node2"].mockBackend.Reset()
	}

	t.Run("Fallover Test V1", func(t *testing.T) {
		reset()
		useNoNodes()
		// Query with no nodes to trigger fallback mode
		_, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x101", false})

		// Expect one request to fail
		require.NoError(t, err)
		require.Equal(t, 503, statusCode)
		require.Equal(t, bg.Consensus.GetFallbackMode(), true)

		_, statusCode, err = client.SendRPC("eth_getBlockByNumber", []interface{}{"0x101", false})
		require.NoError(t, err)
		require.Equal(t, 200, statusCode)
		require.Equal(t, bg.Consensus.GetFallbackMode(), true)

		// first poll
		update()

		// _, statusCode, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x101", false})
		// as a default we use:
		// - latest at 0x101 [257]
		// - safe at 0xe1 [225]
		// - finalized at 0xc1 [193]

		// consensus at block 0x101
		require.Equal(t, "0x101", bg.Consensus.GetLatestBlockNumber().String())
		require.Equal(t, "0xe1", bg.Consensus.GetSafeBlockNumber().String())
		require.Equal(t, "0xc1", bg.Consensus.GetFinalizedBlockNumber().String())
	})
}
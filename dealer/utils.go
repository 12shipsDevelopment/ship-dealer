package dealer

import (
	"context"
	"fmt"
	"io"

	"github.com/filecoin-project/lotus/build"
	blockservice "github.com/ipfs/go-blockservice"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-cidutil"
	bstore "github.com/ipfs/go-ipfs-blockstore"
	chunker "github.com/ipfs/go-ipfs-chunker"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	ipld "github.com/ipfs/go-ipld-format"
	merkledag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs/importer/balanced"
	ihelper "github.com/ipfs/go-unixfs/importer/helpers"
	mh "github.com/multiformats/go-multihash"
)

func buildUnixFS(ctx context.Context, reader io.Reader, into bstore.Blockstore, filestore bool) (cid.Cid, error) {
	b, err := unixFSCidBuilder()
	if err != nil {
		return cid.Undef, err
	}

	bsvc := blockservice.New(into, offline.Exchange(into))
	dags := merkledag.NewDAGService(bsvc)
	bufdag := ipld.NewBufferedDAG(ctx, dags)

	params := ihelper.DagBuilderParams{
		Maxlinks:   build.UnixfsLinksPerLevel,
		RawLeaves:  true,
		CidBuilder: b,
		Dagserv:    bufdag,
		NoCopy:     filestore,
	}

	db, err := params.New(chunker.NewSizeSplitter(reader, int64(build.UnixfsChunkSize)))
	if err != nil {
		return cid.Undef, err
	}
	nd, err := balanced.Layout(db)
	if err != nil {
		return cid.Undef, err
	}

	if err := bufdag.Commit(); err != nil {
		return cid.Undef, err
	}

	return nd.Cid(), nil
}

var DefaultHashFunction = uint64(mh.BLAKE2B_MIN + 31)

func unixFSCidBuilder() (cid.Builder, error) {
	prefix, err := merkledag.PrefixForCidVersion(1)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize UnixFS CID Builder: %w", err)
	}
	prefix.MhType = DefaultHashFunction
	b := cidutil.InlineBuilder{
		Builder: prefix,
		Limit:   126,
	}
	return b, nil
}

package dealer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/12shipsDevelopment/ship-dealer/utils"
	"github.com/filecoin-project/go-commp-utils/ffiwrapper"
	"github.com/filecoin-project/go-fil-markets/stores"
	"github.com/filecoin-project/go-padreader"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-cidutil/cidenc"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipld/go-car"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/ipld/go-ipld-prime/traversal/selector"
	"github.com/ipld/go-ipld-prime/traversal/selector/builder"
	"github.com/multiformats/go-multibase"
	"golang.org/x/xerrors"
)

type CarTask struct {
	Cfg    utils.CarConfig
	Dealer *Dealer
}

func (t *CarTask) LoopImport(chrunkchan chan string) {
	for {
		chrunk := <-chrunkchan
		t.importData(chrunk)
	}
}

func (t *CarTask) Run() {
	log.Printf("start car task\n")

	chrunkch := make(chan string, 1)
	if t.Cfg.Thread == 0 {
		t.Cfg.Thread = 1
	}
	n := 0
	for n < t.Cfg.Thread {
		go t.LoopImport(chrunkch)
		n += 1
	}
	for {
		chrunks := t.Lookupchrunk()
		count := 0
		for _, chrunk := range chrunks {
			sdata := t.Dealer.Model.GetDataByFilename(chrunk)
			carfile := fmt.Sprintf("%s/%s.car", t.Cfg.CarDir, chrunk)
			_, car_err := os.Stat(carfile)
			if sdata.Id == 0 || len(sdata.Cid) == 0 || len(sdata.Commp) == 0 || sdata.Size == 0 || car_err != nil {
				count += 1
				log.Printf("look up %s\n", chrunk)
				t.Dealer.Model.UpdateDataCommp("", 1, chrunk)
				chrunkch <- chrunk
			}
		}
		if count == 0 {
			time.Sleep(600 * time.Second)
		}
	}
}

func (t *CarTask) Lookupchrunk() []string {

	chrunks := make([]string, 0)
	for _, data := range t.Dealer.Model.ListNewData() {
		chrunks = append(chrunks, data.Filename)
	}
	return chrunks
}

func (t *CarTask) importData(path string) {
	/*
		p1 := lotusapi.FileRef{
			Path:  fmt.Sprintf("%s/%s", t.Chrunkdir, path),
			IsCAR: false,
		} */
	fullpath := fmt.Sprintf("%s/%s", t.Cfg.ChrunkDir, path)
	var datacid cid.Cid
	var commp cid.Cid
	var pieceSize abi.UnpaddedPieceSize
	var err error

	data := t.Dealer.Model.GetDataByFilename(path)
	encoder := cidenc.Encoder{Base: multibase.MustNewEncoder(multibase.Base32)}

	carfile := fmt.Sprintf("%s/%s.car", t.Cfg.CarDir, path)
	_, carerr := os.Stat(carfile)

	filename := filepath.Base(path)

	if data.Cid == "" || carerr != nil {
		/*
			if t.ImportData {
				log.Printf("start to import data %s\n", path)
				_, err = t.Dealer.api.ClientImport(context.Background(), p1)
				if err != nil {
					log.Println("failed to import data", err)
					return
				}
				t.Dealer.Model.SetDataImport(path)
			}*/
		log.Printf("start to generate car %s\n", path)
		datacid, err = t.generateCar(fullpath, "./"+filename+".car")
		if err != nil {
			log.Println("failed to gen car", err)
			return
		}
		fcid := encoder.Encode(datacid)
		log.Println("cid", fcid)
		t.Dealer.Model.UpdateDataCid(fcid, path)
	}

	if data.Commp == "" || data.Size == 0 {
		log.Printf("start to calculate commp %s\n", path)
		commp, pieceSize, err = t.calcCommp(filename + ".car")
		if err != nil {
			log.Println("failed to cal commp", err)
			return
		}

		fcommp := encoder.Encode(commp)
		t.Dealer.Model.UpdateDataCommp(fcommp, int64(pieceSize), path)
		log.Println("commp", commp)
		log.Println("Piece size: ", types.SizeStr(types.NewInt(uint64(pieceSize))))
	}
	cmd := exec.Command("mv", filename+".car", carfile)
	_, err = cmd.Output()
	if err != nil {
		log.Println("failed to gen car", err, fullpath)
		return
	}
}

func (t *CarTask) generateCar(srcPath string, outPath string) (cid.Cid, error) {
	ctx := context.Background()
	tmpPath := fmt.Sprintf("%s%s", outPath, ".tmp")
	defer os.Remove(tmpPath)
	var c cid.Cid

	src, err := os.Open(srcPath)
	if err != nil {
		return c, xerrors.Errorf("failed to open input file: %w", err)
	}
	defer src.Close() //nolint:errcheck

	stat, err := src.Stat()
	if err != nil {
		return c, xerrors.Errorf("failed to stat file :%w", err)
	}

	file, err := files.NewReaderPathFile(srcPath, src, stat)
	if err != nil {
		return c, xerrors.Errorf("failed to create reader path file: %w", err)
	}

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return c, xerrors.Errorf("failed to create temp file: %w", err)
	}
	_ = f.Close() // close; we only want the path.

	tmp := f.Name()
	defer os.Remove(tmp) //nolint:errcheck

	// Step 1. Compute the UnixFS DAG and write it to a CARv2 file to get
	// the root CID of the DAG.
	fstore, err := stores.ReadWriteFilestore(tmp)
	if err != nil {
		return c, xerrors.Errorf("failed to create temporary filestore: %w", err)
	}

	finalRoot1, err := buildUnixFS(ctx, file, fstore, true)
	if err != nil {
		_ = fstore.Close()
		return c, xerrors.Errorf("failed to import file to store to compute root: %w", err)
	}
	c = finalRoot1

	if err := fstore.Close(); err != nil {
		return c, xerrors.Errorf("failed to finalize car filestore: %w", err)
	}

	// Step 2. We now have the root of the UnixFS DAG, and we can write the
	// final CAR for real under `dst`.
	bs, err := stores.ReadWriteFilestore(tmpPath, finalRoot1)
	if err != nil {
		return c, xerrors.Errorf("failed to create a carv2 read/write filestore: %w", err)
	}

	// rewind file to the beginning.
	if _, err := src.Seek(0, 0); err != nil {
		return c, xerrors.Errorf("failed to rewind file: %w", err)
	}

	finalRoot2, err := buildUnixFS(ctx, file, bs, true)
	if err != nil {
		_ = bs.Close()
		return c, xerrors.Errorf("failed to create UnixFS DAG with carv2 blockstore: %w", err)
	}

	if err := bs.Close(); err != nil {
		return c, xerrors.Errorf("failed to finalize car blockstore: %w", err)
	}

	if finalRoot1 != finalRoot2 {
		return c, xerrors.New("roots do not match")
	}

	root := finalRoot1
	if err != nil {
		return c, xerrors.Errorf("failed to import file using unixfs: %w", err)
	}

	// open the positional reference CAR as a filestore.
	fs, err := stores.ReadOnlyFilestore(tmp)
	if err != nil {
		return c, xerrors.Errorf("failed to open filestore from carv2 in path %s: %w", tmp, err)
	}
	defer fs.Close() //nolint:errcheck

	// build a dense deterministic CAR (dense = containing filled leaves)
	ssb := builder.NewSelectorSpecBuilder(basicnode.Prototype.Any)
	allSelector := ssb.ExploreRecursive(
		selector.RecursionLimitNone(),
		ssb.ExploreAll(ssb.ExploreRecursiveEdge())).Node()
	sc := car.NewSelectiveCar(ctx, fs, []car.Dag{{Root: root, Selector: allSelector}})
	ff, err := os.Create(outPath)
	if err != nil {
		return c, err
	}
	if err = sc.Write(ff); err != nil {
		return c, xerrors.Errorf("failed to write CAR to output file: %w", err)
	}

	return c, ff.Close()
}

func (t *CarTask) calcCommp(inputPath string) (cid.Cid, abi.UnpaddedPieceSize, error) {

	var (
		commP     cid.Cid
		pieceSize abi.UnpaddedPieceSize
	)

	arbitraryProofType := abi.RegisteredSealProof_StackedDrg32GiBV1

	rdr, err := os.Open(inputPath)
	if err != nil {
		return commP, pieceSize, err
	}
	defer rdr.Close() //nolint:errcheck

	stat, err := rdr.Stat()
	if err != nil {
		return commP, pieceSize, err
	}

	// check that the data is a car file; if it's not, retrieval won't work
	_, _, err = car.ReadHeader(bufio.NewReader(rdr))
	if err != nil {
		return commP, pieceSize, xerrors.Errorf("not a car file: %w", err)
	}

	if _, err := rdr.Seek(0, io.SeekStart); err != nil {
		return commP, pieceSize, xerrors.Errorf("seek to start: %w", err)
	}

	pieceReader, pieceSize := padreader.New(rdr, uint64(stat.Size()))
	commP, err = ffiwrapper.GeneratePieceCIDFromFile(arbitraryProofType, pieceReader, pieceSize)

	if err != nil {
		return commP, pieceSize, xerrors.Errorf("computing commP failed: %w", err)
	}

	return commP, pieceSize, nil
}

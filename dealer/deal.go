package dealer

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/12shipsDevelopment/ship-dealer/model"
	"github.com/12shipsDevelopment/ship-dealer/utils"
	address "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	lotusapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-cidutil/cidenc"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multibase"
)

type DealTask struct {
	Cfg    utils.DealConfig
	Dealer *Dealer
}

func (t *DealTask) Run() {
	log.Printf("start deal #%s# task\n", t.Cfg.Miner)
	go t.syncDealStatus()
	for {
		var data model.Data
		data = t.Dealer.Model.GetAvailableData(t.Cfg.Miner)
		if data.Id > 0 {
			deals := t.Dealer.Model.GetPendingDeal(t.Cfg.Miner)
			client_address, _ := address.NewFromString(t.Cfg.Client)
			client_datacap, _ := t.Dealer.api.StateVerifiedClientStatus(context.Background(), client_address, types.EmptyTSK)
			miner_datacap := t.Dealer.Model.GetMinerDatacapUsed(t.Cfg.Client, t.Cfg.Miner)
			orgfile, err := os.Stat(fmt.Sprintf("%s/%s", t.Dealer.Config.Car.ChrunkDir, data.Filename))
			if err == nil {
				carfile, err := os.Stat(fmt.Sprintf("%s/%s.car", t.Dealer.Config.Car.ChrunkDir, data.Filename))
				if err != nil || carfile.Size() < orgfile.Size() {
					fmt.Printf("invalid car file %s.car\n", data.Filename)
					//os.Remove(fmt.Sprintf("%s/%s.car", t.Dealer.Config.TrunkDir, data.Filename))
					t.Dealer.Model.UpdateDataCommp("", 0, data.Filename)
				}
			}
			if data.Id > 0 && data.Commp != "" && data.Cid != "" && data.Size != 0 && len(deals) < t.Cfg.PendingLimit && client_datacap.Int64() > int64(100*1024*1024*1024) && miner_datacap < int64(t.Cfg.Datacap*1024*1024*1024) && t.Dealer.CurrentEpoch > 0 {
				t.makeDeal(data.Filename)
			} else {
				if len(deals) >= t.Cfg.PendingLimit {
					log.Printf("%s max deal error\n", t.Cfg.Miner)
				}

				if client_datacap.Int64() < int64(100*1024*1024*1024) {
					log.Printf("client datacap error\n")
				}
				if miner_datacap >= int64(t.Cfg.Datacap*1024*1024*1024) {
					log.Printf("%s miner datacap error\n", t.Cfg.Miner)
				}
				if t.Dealer.CurrentEpoch <= 0 {
					log.Printf("%s invalid epoch\n", t.Cfg.Miner)
				}
			}
		} else {
			log.Printf("no avaliable file for %s\n", t.Cfg.Miner)
		}
		time.Sleep(time.Duration(t.Cfg.DealWait) * time.Second)
	}
}

func (t *DealTask) makeDeal(path string) {

	//t.Dealer.Model.DeleteDataByFilename(path)
	//return
	log.Printf("make deal %s\n", path)
	data := t.Dealer.Model.GetDataByFilename(path)
	if data.Id == 0 {
		return
	}

	if data.Cid == "" {
		return
	}

	if data.Commp == "" {
		return
	}

	if data.Size == 0 {
		return
	}

	encoder := cidenc.Encoder{Base: multibase.MustNewEncoder(multibase.Base32)}
	datacid, _ := cid.Parse(data.Cid)
	commp, _ := cid.Parse(data.Commp)
	pieceSize := abi.UnpaddedPieceSize(data.Size)

	var transfer_type string
	transfer_type = storagemarket.TTManual
	ref := &storagemarket.DataRef{
		TransferType: transfer_type,
		Root:         datacid,
	}
	ref.PieceCid = &commp
	ref.PieceSize = pieceSize
	sprice := "0.00000000"
	price, _ := types.ParseFIL(sprice)
	dur := t.Cfg.Duration * 2880
	var provCol big.Int
	from_address, _ := address.NewFromString(t.Cfg.Client)
	miner_address, _ := address.NewFromString(t.Cfg.Miner)
	sdParams := &lotusapi.StartDealParams{
		Data:               ref,
		Wallet:             from_address,
		Miner:              miner_address,
		EpochPrice:         types.BigInt(price),
		MinBlocksDuration:  uint64(dur),
		DealStartEpoch:     abi.ChainEpoch(t.Dealer.CurrentEpoch + 20160),
		FastRetrieval:      true,
		VerifiedDeal:       t.Cfg.VerifiedDeal,
		ProviderCollateral: provCol,
	}
	addr, err := peer.AddrInfoFromString(t.Cfg.Node)
	if err == nil {
		t.Dealer.api.NetConnect(context.Background(), *addr)
	}
	proposal, err := t.Dealer.api.ClientStartDeal(context.Background(), sdParams)
	if err != nil {
		log.Println("failed to start deal", err)
		return
	}
	pid := encoder.Encode(*proposal)
	t.Dealer.Model.CreateClientDeal(data.Id, t.Cfg.Miner, pid, sprice, dur)
	log.Printf("create deal for %s success, pid %s\n", path, pid)
}

func (t *DealTask) syncDealStatus() {
	for {
		deals := t.Dealer.Model.GetPendingDeal(t.Cfg.Miner)
		for _, deal := range deals {
			ctx := context.Background()
			proposal_cid, _ := cid.Parse(deal.ProposalCid)
			info, err := t.Dealer.api.ClientGetDealInfo(context.Background(), proposal_cid)
			if err != nil {
				continue
			}

			if info.State == 7 || info.State == 26 {
				t.Dealer.Model.UpdateClientState(deal.ProposalCid, int(info.State))
				continue
			}

			ts, _ := t.Dealer.api.ChainHead(ctx)
			deal_state, err := t.Dealer.api.StateMarketStorageDeal(context.Background(), info.DealID, ts.Key())
			if err != nil {
				continue
			}
			deal_id, _ := strconv.ParseInt(deal_state.State.SectorStartEpoch.String(), 10, 64)
			if deal_id > 0 {
				t.Dealer.Model.UpdateClientState(deal.ProposalCid, 7)
			}
		}
		time.Sleep(60 * time.Second)
	}
}

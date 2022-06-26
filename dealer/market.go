package dealer

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/12shipsDevelopment/ship-dealer/utils"
	jsonrpc "github.com/filecoin-project/go-jsonrpc"
	lotusapi "github.com/filecoin-project/lotus/api"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-cidutil/cidenc"
	multibase "github.com/multiformats/go-multibase"
)

type MarketTask struct {
	Cfg    utils.MarketConfig
	Dealer *Dealer
	API    lotusapi.StorageMinerStruct
}

func (t *MarketTask) ParseArgs(args []interface{}) {
}

func (t *MarketTask) MarketImportData(miner string, dealchan chan cid.Cid) {
	encoder := cidenc.Encoder{Base: multibase.MustNewEncoder(multibase.Base32)}
	var api lotusapi.StorageMinerStruct
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+t.Cfg.Token)
	closer, err := jsonrpc.NewMergeClient(context.Background(), "http://"+t.Cfg.RPC+"/rpc/v0", "Filecoin", lotusapi.GetInternalStructs(&api), headers)
	if err != nil {
		log.Fatal("failed to connect market")
		return
	}
	t.API = api
	defer closer()
	for {
		cid := <-dealchan
		cdeal := t.Dealer.Model.GetClientDealByProposalCid(encoder.Encode(cid))
		if cdeal.Id <= 0 {
			log.Println("failed to get client deal", cid)
			continue
		}
		mdeal := t.Dealer.Model.GetMarketDealByProposalCid(encoder.Encode(cid))
		if mdeal.Id > 0 {
			log.Printf("cid %s already import\n", cid)
			continue
		}
		sdata := t.Dealer.Model.GetDataById(cdeal.DataId)
		if sdata.Id == 0 {
			log.Printf("missing file for proposal %s\n", cid)
			continue
		}
		carfile := fmt.Sprintf("%s/%s.car", t.Cfg.CarDir, sdata.Filename)
		_, carerr := os.Stat(carfile)
		if carerr != nil {
			//fil.TrunkChan <- sdata.Filename
			t.Dealer.Model.UpdateDataCommp("", 0, sdata.Filename)
			log.Printf("%s not found\n", carfile)
			continue
		}
		log.Printf("start to import data %s to proposal %s\n", sdata.Filename, cid)
		t.Dealer.Model.UpdateClientState(encoder.Encode(cid), 99)
		err := api.DealsImportData(context.Background(), cid, carfile)
		if err != nil {
			t.Dealer.Model.UpdateClientState(encoder.Encode(cid), 0)
			log.Printf("failed to import data %s to proposal %s, err: %s\n", sdata.Filename, cid, err)
			continue
		} else {
			t.Dealer.Model.UpdateClientState(encoder.Encode(cid), 98)
			t.Dealer.Model.CreateMarketDeal(encoder.Encode(cid))
		}
		log.Printf("import data %s to proposal %s finished \n", sdata.Filename, cid)
		time.Sleep(time.Duration(t.Cfg.ImportWait) * time.Second)
	}
}

func (t *MarketTask) Run() {
	log.Printf("start market %s task\n", t.Cfg.Miner)
	var api lotusapi.StorageMinerStruct
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+t.Cfg.Token)
	closer, err := jsonrpc.NewMergeClient(context.Background(), "http://"+t.Cfg.RPC+"/rpc/v0", "Filecoin", lotusapi.GetInternalStructs(&api), headers)
	if err != nil {
		log.Println("failed to connect market")
		return
	}

	encoder := cidenc.Encoder{Base: multibase.MustNewEncoder(multibase.Base32)}
	dealchan := make(chan cid.Cid, 1)

	go t.MarketImportData(t.Cfg.Miner, dealchan)

	defer closer()
	for {
		deals, err := api.MarketListIncompleteDeals(context.Background())
		if err == nil {
			for _, deal := range deals {
				//log.("propocal_cid: %s, state: %d\n", deal.ProposalCid, deal.State)
				if deal.State == 18 {
					cdeal := t.Dealer.Model.GetClientDealByProposalCid(encoder.Encode(deal.ProposalCid))
					if cdeal.Id == 0 {
						if deal.Proposal.Client.String() != t.Dealer.Config.Client {
							continue
						}
						log.Printf("missing client deal %s from %s \n", encoder.Encode(deal.ProposalCid), deal.Proposal.Client.String())
						continue
					}
					if cdeal.Id > 0 && cdeal.State != 99 && cdeal.State != 98 {
						dealchan <- deal.ProposalCid
					}
				}
				if deal.State == 7 || deal.State == 26 {
					t.Dealer.Model.UpdateClientState(encoder.Encode(deal.ProposalCid), int(deal.State))
				}
			}
		} else {
			log.Println("faield to get market deals", err)
		}
		time.Sleep(60 * time.Second)
	}
}

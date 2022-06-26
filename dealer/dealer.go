package dealer

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/12shipsDevelopment/ship-dealer/model"
	"github.com/12shipsDevelopment/ship-dealer/utils"
	jsonrpc "github.com/filecoin-project/go-jsonrpc"
	lotusapi "github.com/filecoin-project/lotus/api"
	cid "github.com/ipfs/go-cid"
)

type Task interface {
	Run()
	ParseArgs([]interface{})
}

type Dealer struct {
	ctx          context.Context
	api          lotusapi.FullNodeStruct
	Config       *utils.Config
	Version      string
	CurrentEpoch uint64
	Model        *model.Model
	dealChan     chan cid.Cid
	TrunkChan    chan string
	CarTask      *CarTask
	DealTask     *DealTask
	MarketTask   *MarketTask
}

func NewDealer(cfg *utils.Config) *Dealer {
	sm, err := model.NewModel(cfg.Database, cfg.Debug)
	if err != nil {
		log.Println(err)
		return nil
	}
	dealer := Dealer{
		ctx:       context.Background(),
		Config:    cfg,
		Model:     sm,
		Version:   "1.2.2",
		TrunkChan: make(chan string, 1),
	}
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+cfg.Token)
	if cfg.Node != "" {
		var api lotusapi.FullNodeStruct
		_, err = jsonrpc.NewMergeClient(context.Background(), "http://"+cfg.Node+"/rpc/v0", "Filecoin", []interface{}{&api.Internal, &api.CommonStruct.Internal}, headers)
		if err != nil {
			log.Fatalf("connecting with lotus failed: %s\n", err)
		}
		if err != nil {
			log.Fatalf("calling chain head: %s\n", err)
		}
		dealer.api = api
	}

	car := CarTask{
		Cfg:    cfg.Car,
		Dealer: &dealer,
	}
	deal := DealTask{
		Cfg:    cfg.Deal,
		Dealer: &dealer,
	}
	market := MarketTask{
		Cfg:    cfg.Market,
		Dealer: &dealer,
	}

	dealer.CarTask = &car
	dealer.DealTask = &deal
	dealer.MarketTask = &market
	return &dealer
}

func (dealer *Dealer) syncEpoch() {
	ctx := context.Background()
	ticker := time.NewTicker(30 * time.Second)
	for {
		status, _ := dealer.api.NodeStatus(ctx, true)
		dealer.CurrentEpoch = status.SyncStatus.Epoch
		log.Printf("current epoch %d\n", dealer.CurrentEpoch)
		<-ticker.C
	}
}

func (dealer *Dealer) Run() {
	done := make(chan int, 1)
	log.Println("Version", dealer.Version)
	go dealer.syncEpoch()
	if dealer.Config.Car.Enable {
		go dealer.CarTask.Run()
	}
	if dealer.Config.Deal.Enable {
		go dealer.DealTask.Run()
	}
	if dealer.Config.Market.Enable {
		go dealer.MarketTask.Run()
	}
	<-done
}

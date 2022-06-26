package model

type ClientDeals struct {
	Id          uint
	DataId      uint
	ProposalCid string
	State       int
	Provider    string
	Price       string
	Duration    int
	DealId      string
}

func (model *Model) ListDeals() []ClientDeals {
	var deals []ClientDeals
	model.DB.Find(&deals)
	return deals
}

func (model *Model) GetClientDealByProposalCid(proposal_cid string) ClientDeals {
	var deal ClientDeals
	model.DB.Where("proposal_cid", proposal_cid).First(&deal)
	return deal
}

func (model *Model) CreateClientDeal(data_id uint, provider, proposal_cid string, price string, duration int) {
	data := model.GetDataById(data_id)
	if data.Cid == "" {
		return
	}
	deal := ClientDeals{
		DataId:      data_id,
		Provider:    provider,
		ProposalCid: proposal_cid,
		Price:       price,
		Duration:    duration,
	}
	model.DB.Create(&deal)
}

func (model *Model) GetPendingDeal(miner string) []ClientDeals {
	var deals []ClientDeals
	model.DB.Model(&ClientDeals{}).Where("provider", miner).Not(map[string]interface{}{"state": []int{7, 26}}).Find(&deals)
	return deals
}

func (model *Model) UpdateClientState(proposal_cid string, state int) {
	model.DB.Model(&ClientDeals{}).Where("proposal_cid", proposal_cid).Update("state", state)
}

func (model *Model) UpdateDealId(proposal_cid string, deal_id string) {
	model.DB.Model(&ClientDeals{}).Where("proposal_cid", proposal_cid).Update("deal_id", deal_id)
}

func (model *Model) UpdateClientDealId(proposal_cid string, deal_id string) {
	model.DB.Model(&ClientDeals{}).Where("proposal_cid", proposal_cid).Update("deal_id", deal_id)
}

func (model *Model) GetMinerDatacapUsed(client, miner string) int64 {
	var datacap int64
	model.DB.Raw("select COALESCE(sum(size), 0)  from (select client_deals.id, d.size from client_deals  left join `data` d on client_deals.data_id = d.id  where client_deals.provider = '" + miner + "' and client_deals.state != 26) t;").Scan(&datacap)
	return datacap
}

package model

type MarketDeal struct {
	Id          uint
	ProposalCid string
	State       int
	DealId      string
}

func (model *Model) CreateMarketDeal(proposal_cid string) {
	deal := MarketDeal{
		ProposalCid: proposal_cid,
	}
	model.DB.Create(&deal)
}

func (model *Model) GetMarketDealByProposalCid(proposal_cid string) MarketDeal {
	var deal MarketDeal
	model.DB.Where("proposal_cid", proposal_cid).First(&deal)
	return deal
}

func (model *Model) UpdateMarketState(proposal_cid string, state int) {
	model.DB.Model(&ClientDeals{}).Where("proposal_cid", proposal_cid).Update("state", state)
}

func (model *Model) UpdateMarketDealId(proposal_cid string, deal_id string) {
	model.DB.Model(&ClientDeals{}).Where("proposal_cid", proposal_cid).Update("deal_id", deal_id)
}

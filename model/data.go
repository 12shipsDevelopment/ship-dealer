package model

type Data struct {
	Id       uint
	Filename string
	Status   int
	Cid      string
	Commp    string
	Client   string
	Size     int64
}

func (model *Model) ListData() []Data {
	var datas []Data
	model.DB.Find(&datas)
	return datas
}

func (model *Model) GetDataById(data_id uint) Data {
	var sdata Data
	model.DB.Where("id", data_id).First(&sdata)
	return sdata
}

func (model *Model) GetDataByFilename(filename string) Data {
	var sdata Data
	model.DB.Where("filename", filename).First(&sdata)
	return sdata
}

func (model *Model) DeleteDataByFilename(filename string) {
	model.DB.Where("filename", filename).Delete(Data{})
}

func (model *Model) NewData(filename string) {
	data := Data{
		Filename: filename,
		Status:   0,
	}
	model.DB.Create(&data)
}

func (model *Model) UpdateDataCid(cid string, filename string) {
	model.DB.Model(&Data{}).Where("filename", filename).Update("cid", cid)
	model.DB.Commit()
}

func (model *Model) UpdateDataCommp(commp string, size int64, filename string) {
	model.DB.Model(&Data{}).Where("filename", filename).Update("commp", commp).Update("size", size)
	model.DB.Commit()
}

func (model *Model) UpdateStatus(status int, filename string) {
	model.DB.Model(&Data{}).Where("filename", filename).Update("status", status)
	model.DB.Commit()
}

func (model *Model) UpdateFilename(src_filename, dst_filename string) {
	model.DB.Model(&Data{}).Where("filename", src_filename).Update("filename", dst_filename)
	model.DB.Commit()
}

func (model *Model) CleanDirtyData() {
	model.DB.Where("cid", "").Delete(Data{})
	model.DB.Where("commp", "").Delete(Data{})
}

func (model *Model) GetAvailableData(miner string) Data {
	var data Data
	model.DB.Raw("select data.*, t.c from `data` left join (select count(data_id)c,  data_id from client_deals where provider = '" + miner + "' and state != 26 group by data_id) t on t.data_id = data.id where data.cid != '' and data.commp != '' and data.size > 0  and (t.c < 1 or t.c is null) order by c asc limit 1").Scan(&data)
	return data
}

func (model *Model) ListNewData() []Data {
	var datas []Data
	model.DB.Find(&datas, "commp = '' or size = 0")
	return datas
}

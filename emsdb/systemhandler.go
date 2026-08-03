package emsdb

type ProjectSn struct {
	Sn       string `gorm:"column:SN;primary_key;type:char(30)"`
	ProjName string `gorm:"column:PROJECT_NAME;type:char(128)"`
	ProjAddr string `gorm:"column:PROJECT_ADDR;type:char(255)"`
}

func (ProjectSn) TableName() string {
	return "mqtt_project"
}

type SnData struct {
	Sn     string `json:"sn"`
	Name   string `json:"name"`
	Addr   string `json:"addr"`
	CabNum int    `json:"cabnum"`
}

func GetProjectSn2() (*ProjectSn, error) {
	// snDB, err := db.NewMsDB()
	// if err != nil {
	// 	log.Info("GetProjectSn open db err:", err)
	// 	return nil, fmt.Errorf("GetProjectSn open db err:%v", err)
	// }

	// var projectInfo ProjectSn
	// tx := snDB.First(&projectInfo)
	// if tx.Error != nil {
	// 	log.Error("GetProjectSn First err:", tx.Error)
	// 	return nil, fmt.Errorf("GetProjectSn First err:%v", tx.Error)
	// }
	// // if len(projectInfo) == 0 {
	// // 	log.Error("GetProjectSn from mqtt_project empty.")
	// // 	return nil, fmt.Errorf("GetProjectSn from mqtt_project empty.")
	// // }
	// log.Info("GetProjectSn. info:", projectInfo)
	// return &projectInfo, nil

	return &ProjectSn{}, nil
}

func GetProjectSn() (*ProjectSn, error) {
	// snDB, err := db.NewMsDB()
	// if err != nil {
	// 	log.Info("GetProjectSn open db err:", err)
	// 	return nil, fmt.Errorf("GetProjectSn open db err:%v", err)
	// }

	// var projectInfo []ProjectSn
	// tx := snDB.Find(&projectInfo)
	// if tx.Error != nil {
	// 	log.Error("GetProjectSn First err:", tx.Error)
	// 	return nil, fmt.Errorf("GetProjectSn First err:%v", tx.Error)
	// }
	// if len(projectInfo) == 0 {
	// 	log.Error("GetProjectSn from mqtt_project empty.")
	// 	return &ProjectSn{}, nil
	// }
	// log.Info("GetProjectSn. info:", projectInfo)
	// return &projectInfo[0], nil

	return &ProjectSn{}, nil
}

func UpdateProjectSn(snInfo ProjectSn) error {
	// snDB, err := db.NewMsDB()
	// if err != nil {
	// 	log.Info("UpdateProjectSn open db err:", err)
	// 	return fmt.Errorf("UpdateProjectSn open db err:%v", err)
	// }

	// var projectInfo []ProjectSn
	// tx := snDB.Find(&projectInfo)
	// if tx.Error != nil {
	// 	log.Error("UpdateProjectSn First err:", tx.Error)
	// 	return fmt.Errorf("UpdateProjectSn First err:%v", tx.Error)
	// }
	// if len(projectInfo) == 0 {
	// 	log.Info("UpdateProjectSn while mqtt_project empty.")
	// 	_, err := snDB.Create(&snInfo)
	// 	if err != nil {
	// 		log.Error("UpdateProjectSn Create err:", err)
	// 		return fmt.Errorf("UpdateProjectSn Create err:%v", err)
	// 	}
	// 	return nil
	// }
	// log.Info("UpdateProjectSn while mqtt_project not empty.")

	// data := make(map[string]interface{})
	// var projModel ProjectSn
	// data["sn"] = snInfo.Sn
	// data["project_name"] = snInfo.ProjName
	// data["project_addr"] = snInfo.ProjAddr
	// _, err = snDB.Updates(&projModel, data, "SN = ? ", projectInfo[0].Sn)
	// if err != nil {
	// 	log.Error("UpdateProjectSn Updates err:", err)
	// 	return fmt.Errorf("UpdateProjectSn Updates err:%v", err)
	// }
	return nil
}

package mongo

import (
	"errors"
	"fmt"
	//"github.com/globalsign/mgo/bson"

	"go.mongodb.org/mongo-driver/bson"

	//"go.mongodb.org/mongo-driver/bson"

	//"go.mongodb.org/mongo-driver/bson"
	"reflect"
)

type Setting struct {
	ID              string `bson:"_id" json:"id"`
	FindByTmmId     int    `bson:"find_f_tid" json:"find_f_tid"`       // search by tmm id
	FindByPhone     uint8  `bson:"find_f_phone" json:"find_f_phone"`   // search by phone
	AddFromGroup    uint8  `bson:"add_f_group" json:"add_f_group"`     // added friend by group
	AddFromQrCode   uint8  `bson:"add_f_qrcode" json:"add_f_qrcode"`   // added friend by qr code
	AddFromCard     uint8  `bson:"add_f_card" json:"add_f_card"`       // added by card
	AddFromMoments  uint8  `bson:"add_f_moments" json:"add_f_moments"` //
	AddFromDiscover uint8  `bson:"add_f_disc" json:"add_f_disc"`
	Language        string `bson:"language" json:"language"`
	SubLang         string `bson:"sub_lang" json:"sub_lang"` // if language equal 1 (auto) ,sub_lang is system language
	CreatedAt       int64  `bson:"create_time" json:"create_time"`
	UpdatedAt       int64  `bson:"update_time" json:"update_time"`
	Seq             int64  `bson:"seq" json:"seq"`
}

const (
	AutoLang  = "1"
	StatusNo  = 0
	StatusYes = 1
)

var (
	DefaultSetting = Setting{
		FindByTmmId:     StatusYes,
		FindByPhone:     StatusYes,
		AddFromGroup:    StatusYes,
		AddFromQrCode:   StatusYes,
		AddFromCard:     StatusYes,
		AddFromMoments:  StatusYes,
		AddFromDiscover: StatusYes,
	}
)

func (s Setting) TableName() string {
	return "user_config"
}

func New() *Setting {
	return new(Setting)
}

func (s Setting) getCollection(mode ...Mode) *CollectionInfo {
	rpMode := PrimaryMode
	if len(mode) > 0 && mode[0] > 0 {
		rpMode = mode[0]
	}
	return Database("tmm").SetTable(s.TableName(), rpMode)
}

func (s Setting) GetByID(id, fields string) (Setting, error) {
	var data Setting
	err := s.getCollection(SecondaryPreferredMode).Fields(GetMongoFieldsBsonByString(fields)).FindByID(id, &data)
	return data, err
}

func (s Setting) Add(data Setting) bool {
	_, err := s.getCollection().InsertOne(data)
	return err == nil
}

func (s Setting) GetFieldName(field string) string {
	ins := Setting{}
	reflectIns := reflect.TypeOf(ins)
	if fieldObj, ok := reflectIns.FieldByName(field); ok {
		return fieldObj.Tag.Get("bson")
	}
	return ""
}

func (s Setting) UpdateValue(id, field string, value interface{}) (bool, error) {
	f := s.GetFieldName(field)
	if f == "" {
		return false, errors.New("field not exist")
	}
	uData := map[string]interface{}{
		f: value,
	}
	res, err := s.getCollection().UpByID(id, uData)
	if err == nil && res.ModifiedCount == 1 {
		return true, nil
	} else if err == nil && res.MatchedCount == 0 {
		uData["_id"] = id
		_, err = s.getCollection().InsertOne(uData)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, err
}

func (s Setting) GetByIDs(ids []string, fields string) []Setting {
	data := make([]Setting, 0)
	fmt.Println(ids)
	s.getCollection().Where(bson.M{"_id": bson.M{"$in": ids}}).Fields(GetMongoFieldsBsonByString(fields)).FindMany(&data)
	fmt.Println(data)
	return data
}

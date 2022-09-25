package utils

import "go.mongodb.org/mongo-driver/bson/primitive"

type DBNameListItem struct {
	Time primitive.Timestamp `bson:"time"`
	Name string              `bson:"name"`
}

type DBSkinListItem struct {
	Time primitive.Timestamp `bson:"time"`

	SkinDataHash string `bson:"skin_data_hash"`
	SkinId       string `bson:"skin_id"`

	CapeDataHash string `bson:"cape_data_hash"`
	CapeId       string `bson:"cape_id"`

	SkinResourceHash          string `bson:"skin_resource_hash"`
	GeometryHash              string `bson:"geometry_hash"`
	GeometryDataEngineVersion string `bson:"geometry_data_version"`

	FullId      string `bson:"full_id"`
	PlayFabId   string `bson:"playfab_id"`
	PremiumSkin bool   `bson:"is_premium_skin"`
	PersonaSkin bool   `bson:"is_persona_skin"`
	SkinColour  string `bson:"skin_colour"`
	ArmSize     string `bson:"arm_size"`
}

type DBPlayerInfo struct {
	Id   primitive.ObjectID `bson:"_id,omitempty"`
	Xuid string             `bson:"xuid"`

	ObservedNamesIngame []DBNameListItem `bson:"observed_names_ingame"`
	ObservedNamesXbox   []DBNameListItem `bson:"observed_names_xbox"`
	ObservedSkins       []DBSkinListItem `bson:"observed_skins"`
}

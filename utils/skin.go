package utils

import (
	"encoding/base64"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type QueuedSkin struct {
	Username      string
	Xuid          string
	Skin          *JsonSkinData
	ServerAddress string
	Time          int64
}

type Skin struct {
	protocol.Skin
}

type Skin_anim struct {
	ImageWidth, ImageHeight uint32
	ImageData               string
	AnimationType           uint32
	FrameCount              float32
	ExpressionType          uint32
}

type JsonSkinData struct {
	SkinID                          string
	PlayFabID                       string
	SkinResourcePatch               string
	SkinImageWidth, SkinImageHeight uint32
	SkinData                        string
	Animations                      []Skin_anim
	CapeImageWidth, CapeImageHeight uint32
	CapeData                        string
	SkinGeometry                    string
	AnimationData                   string
	GeometryDataEngineVersion       string
	PremiumSkin                     bool
	PersonaSkin                     bool
	PersonaCapeOnClassicSkin        bool
	PrimaryUser                     bool
	CapeID                          string
	FullID                          string
	SkinColour                      string
	ArmSize                         string
	PersonaPieces                   []protocol.PersonaPiece
	PieceTintColours                []protocol.PersonaPieceTintColour
	Trusted                         bool
}

func (s *Skin) Json() *JsonSkinData {
	var skin_animations []Skin_anim
	for _, sa := range s.Animations {
		skin_animations = append(skin_animations, Skin_anim{
			ImageWidth:     sa.ImageWidth,
			ImageHeight:    sa.ImageHeight,
			ImageData:      base64.RawStdEncoding.EncodeToString(sa.ImageData),
			AnimationType:  sa.AnimationType,
			FrameCount:     sa.FrameCount,
			ExpressionType: sa.ExpressionType,
		})
	}
	return &JsonSkinData{
		SkinID:                    s.SkinID,
		PlayFabID:                 s.PlayFabID,
		SkinResourcePatch:         base64.RawStdEncoding.EncodeToString(s.SkinResourcePatch),
		SkinImageWidth:            s.SkinImageWidth,
		SkinImageHeight:           s.SkinImageHeight,
		SkinData:                  base64.RawStdEncoding.EncodeToString(s.SkinData),
		Animations:                skin_animations,
		CapeImageWidth:            s.CapeImageWidth,
		CapeImageHeight:           s.CapeImageHeight,
		CapeData:                  base64.RawStdEncoding.EncodeToString(s.CapeData),
		SkinGeometry:              base64.RawStdEncoding.EncodeToString(s.SkinGeometry),
		AnimationData:             base64.RawStdEncoding.EncodeToString(s.AnimationData),
		GeometryDataEngineVersion: string(s.GeometryDataEngineVersion),
		PremiumSkin:               s.PremiumSkin,
		PersonaSkin:               s.PersonaSkin,
		PersonaCapeOnClassicSkin:  s.PersonaCapeOnClassicSkin,
		PrimaryUser:               s.PrimaryUser,
		CapeID:                    s.CapeID,
		FullID:                    s.FullID,
		SkinColour:                s.SkinColour,
		ArmSize:                   s.ArmSize,
		PersonaPieces:             s.PersonaPieces,
		PieceTintColours:          s.PieceTintColours,
		Trusted:                   s.Trusted,
	}
}

func (j *JsonSkinData) ToSkin() *Skin {
	skin_resourcepatch, _ := base64.RawStdEncoding.DecodeString(j.SkinResourcePatch)
	skin_data, _ := base64.RawStdEncoding.DecodeString(j.SkinResourcePatch)
	cape_data, _ := base64.RawStdEncoding.DecodeString(j.CapeData)
	skin_geometry, _ := base64.RawStdEncoding.DecodeString(j.SkinGeometry)

	skin_animations := make([]protocol.SkinAnimation, len(j.Animations))
	for i, s2 := range j.Animations {
		image_data, _ := base64.RawStdEncoding.DecodeString(s2.ImageData)
		skin_animations[i] = protocol.SkinAnimation{
			ImageWidth:     s2.ImageWidth,
			ImageHeight:    s2.ImageHeight,
			ImageData:      image_data,
			AnimationType:  s2.AnimationType,
			FrameCount:     s2.FrameCount,
			ExpressionType: s2.ExpressionType,
		}
	}

	animation_data, _ := base64.RawStdEncoding.DecodeString(j.AnimationData)

	s := Skin{
		Skin: protocol.Skin{
			SkinID:                    j.SkinID,
			PlayFabID:                 j.PlayFabID,
			SkinResourcePatch:         skin_resourcepatch,
			SkinImageWidth:            j.SkinImageWidth,
			SkinImageHeight:           j.SkinImageHeight,
			SkinData:                  skin_data,
			Animations:                skin_animations,
			CapeImageWidth:            j.CapeImageWidth,
			CapeImageHeight:           j.CapeImageHeight,
			CapeData:                  cape_data,
			SkinGeometry:              skin_geometry,
			AnimationData:             animation_data,
			GeometryDataEngineVersion: []byte(j.GeometryDataEngineVersion),
			PremiumSkin:               j.PremiumSkin,
			PersonaSkin:               j.PersonaSkin,
			PersonaCapeOnClassicSkin:  j.PersonaCapeOnClassicSkin,
			PrimaryUser:               j.PrimaryUser,
			CapeID:                    j.CapeID,
			FullID:                    j.FullID,
			SkinColour:                j.SkinColour,
			ArmSize:                   j.ArmSize,
			PersonaPieces:             j.PersonaPieces,
			PieceTintColours:          j.PieceTintColours,
			Trusted:                   j.Trusted,
		},
	}

	return &s
}

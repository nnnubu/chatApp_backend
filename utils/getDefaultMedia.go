package utils

import (
	"math/rand"
)

func GetRandomDefaultMedia() (avatarRelPath string, bgRelPath string) {
	avatarPool := []string{"df_avatar1.jpg", "df_avatar2.jpg", "df_avatar3.jpg"}
	pickAvatarName := avatarPool[rand.Intn(len(avatarPool))]
	avatarRelPath = "/static/default/avatar/" + pickAvatarName
	switch pickAvatarName {
	case "df_avatar1.jpg", "df_avatar2.jpg":
		bgRelPath = "/static/default/bgImg/df_bgImg1.jpg"
	case "df_avatar3.jpg":
		bgRelPath = "/static/default/bgImg/df_bgImg.jpg"
	}
	return
}

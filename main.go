package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/na0chan-go/spl-random-bot/app"
	"github.com/na0chan-go/spl-random-bot/app/model"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	usermap = map[string]*model.UserState{}
	// discord   *discordgo.Session
	stopBot   = make(chan bool)
	vcsession *discordgo.VoiceConnection
)

// init 初期処理
func init() {
	loadEnv()
}

// loadEnv envファイルを読み込む
func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("error loading .env")
	}
	fmt.Println("load .env")
}

func main() {
	// Discordのクライアントを生成
	discord, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		fmt.Println("error discord client")
		fmt.Println(err)
	}

	discord.AddHandler(onMessageCreateHandler)
	discord.AddHandler(onVoiceStateUpdateHandler)
	err = discord.Open()
	if err != nil {
		fmt.Println(err)
	}

	defer discord.Close()

	fmt.Println("Listening...")
	<-stopBot
}

// onMessageCreateHandler メッセージが送られたときに動作する
func onMessageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// チャットを送ったユーザーログ
	log.Printf("%20s %20s %20s > %s\n", m.ChannelID, time.Now().Format(time.Stamp), m.Author.Username, m.Content)
	switch {
	// command: !join
	// botをVCに追加する
	case strings.HasPrefix(m.Content, app.CommandChannelVoiceJoin):
		var user *model.UserState
		if usermap[m.Author.ID] != nil {
			user = usermap[m.Author.ID]
		}
		// チャンネル取得
		channel, err := s.State.Channel(m.ChannelID)
		if err != nil {
			log.Println("error get channel")
		}
		// vcsession生成
		vcsession, err = s.ChannelVoiceJoin(channel.GuildID, user.CurrentVC, false, false)
		if err != nil {
			log.Println(err)
		}
		sendMessage(s, user.CurrentVC, "ボイスチャンネルに参加しました")
	// command: !leave
	// botをVCから退出させる
	case strings.HasPrefix(m.Content, app.CommandChannelVoiceLeave):
		if vcsession != nil {
			vcsession.Disconnect()
		} else {
			log.Println("vcsessionが存在しません")
		}
	// command: !random
	// スプラトゥーンの武器をVCに参加しているユーザーに割り当てる
	case strings.HasPrefix(m.Content, app.CommandRandom):
		weapons, err := fetchWeapon()
		if err != nil {
			log.Println(err)
		}
		for _, user := range usermap {
			randomWeapon := weapons[rand.Intn(len(weapons))]
			u, err := s.User(user.ID)
			if err != nil {
				log.Println(err)
			}
			// botでないとき
			if !u.Bot {
				log.Printf("%vさん:%v\n", user.Name, randomWeapon.WeaponName.JPName)
				sendMessage(s, user.CurrentVC, fmt.Sprintf("%vさん:%v\n", user.Name, randomWeapon.WeaponName.JPName))
			}
		}

	// command: !guild
	case strings.HasPrefix(m.Content, app.CommandGuild):
		guild, err := s.Guild(m.GuildID)
		if err != nil {
			log.Println(err)
		}
		log.Printf("%+v", guild)
		log.Println(guild.VoiceStates)

	// command: !users
	// 同じVCにいるユーザー一覧を出力する
	case strings.HasPrefix(m.Content, app.CommandUsers):
		// コマンドを送ったユーザーがいるCurrentVC
		CurrentVC := usermap[m.Author.ID].CurrentVC
		log.Printf("チャットを送信したユーザー:%vのカレントVC:%v\n", usermap[m.Author.ID].Name, CurrentVC)
		for _, user := range usermap {
			log.Printf("%+v\n", user)
			if user.CurrentVC == CurrentVC {
				log.Printf("%vさんが参加してます\n", user.Name)
				sendMessage(s, user.CurrentVC, fmt.Sprintf("%vさんが参加してます\n", user.Name))
			}
		}
	}
}

// onVoiceStateUpdateHandler ボイスチャンネルの状態が変わったとき動作する
func onVoiceStateUpdateHandler(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// Userが存在しないとき、Userを新規に設定する
	_, ok := usermap[v.UserID]
	if !ok {
		usermap[v.UserID] = new(model.UserState)
		user, err := s.User(v.UserID)
		if err != nil {
			log.Printf("err s.User: %v\n", err)
		}
		usermap[v.UserID].ID = user.ID
		usermap[v.UserID].Name = user.Username
		log.Printf("new user: %v\n", user.Username)
	}

	// チャンネルIDが存在するかつ、UsermapのカレントボイスチャンネルとチャンネルIDが一致しないとき
	if len(v.ChannelID) > 0 && usermap[v.UserID].CurrentVC != v.ChannelID {
		channel, _ := s.Channel(v.ChannelID)
		log.Printf("%vさんが%vに参加しました\n", usermap[v.UserID].Name, channel.Name)
	}
	// Usermapのカレントボイスチャンネルに現在のチャンネルIDを設定する
	usermap[v.UserID].CurrentVC = v.ChannelID
	if len(v.ChannelID) > 0 {
		log.Printf("現在%vさんは%vに参加しています\n", usermap[v.UserID].Name, usermap[v.UserID].CurrentVC)
	}
}

// fetchWeapon スプラ３の武器情報を取得する
func fetchWeapon() ([]model.Weapon, error) {
	// APIから武器情報を取得
	res, err := http.Get("https://stat.ink/api/v3/weapon")
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("http status %v", res.StatusCode)
	}

	// jsonを読み込む
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var weapons []model.Weapon
	if err := json.Unmarshal(body, &weapons); err != nil {
		return nil, err
	}
	return weapons, nil
}

// sendMessage メッセージを送信する関数
func sendMessage(s *discordgo.Session, channelID string, msg string) {
	_, err := s.ChannelMessageSend(channelID, msg)

	log.Println(">>> " + msg)
	if err != nil {
		log.Println("Error sending message: ", err)
	}
}

// sendReply リプライを送る
func sendReply(s *discordgo.Session, channelID string, msg string, reference *discordgo.MessageReference) {
	_, err := s.ChannelMessageSendReply(channelID, msg, reference)
	if err != nil {
		log.Println("Error sending message: ", err)
	}
}

// SendEmbedWithField 埋め込みメッセージを指定されたチャンネルに投稿します
func SendEmbedWithField(s *discordgo.Session, channelID, title, desc string, field []*discordgo.MessageEmbedField) error {
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       0xFF0000,
		Title:       title,
		Description: desc,
		Fields:      field,
	}
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	return err
}

// SendEmbed 埋め込みメッセージを指定されたチャンネルに投稿します
func SendEmbed(s *discordgo.Session, channelID, title, desc string) error {
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       0xFF0000,
		Title:       title,
		Description: desc,
	}
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	return err
}

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bitly/go-simplejson"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func discord_get_token() string {
	env_var := os.Getenv("DISCORD_BOT_TOKEN")
	if env_var != "" {
		return env_var
	} else {
		log.Println("Couldn't get environement variable DISCORD_BOT_TOKEN, checking if there's a .env file")
	}

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Couldn't load .env")
	}

	env_var = os.Getenv("DISCORD_BOT_TOKEN")
	if env_var == "" {
		log.Fatal("Couldn't get DISCORD_BOT_TOKEN in .env")
	}

	return env_var
}

func discord_prepare_video(url string, user_id string, json *simplejson.Json) *discordgo.MessageSend {
	vid := tiktok_extract_video(json)

	var requested_by string
	var msg *discordgo.MessageSend

	if user_id != "" {
		requested_by = fmt.Sprintf("**Requested by <@%s>**\n", user_id)
	} else {
		requested_by = user_id
	}

	video_reader := vid.get_video_reader()
	video_file := discordgo.File{
		Name:        fmt.Sprintf("%s.mp4", vid.id),
		ContentType: "video/mp4",
		Reader:      video_reader,
	}

	msg = &discordgo.MessageSend{
		Content: fmt.Sprintf("%s**Author: **%s\n**Desc:** %s\n[Tiktok link](<%s>)\n[Raw video link](<%s>)", requested_by, vid.author, vid.desc, url, vid.url),
		Files: []*discordgo.File{
			&video_file,
		},
	}

	return msg
}

func discord_prepare_images(url string, user_id string, json *simplejson.Json) *discordgo.MessageSend {
	msg := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{},
	}
	imgs := tiktok_extract_images(json)
	var fulldesc string
	if user_id != "" {
		fulldesc = fmt.Sprintf("Requested by <@%s>\n%s\n[Music link](%s)", user_id, imgs.desc, imgs.music_url)
	} else {
		fulldesc = imgs.desc
	}

	for n, img := range imgs.urls {
		if n == 8 {
			break
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Gothik | %s", imgs.author),
			Description: fulldesc,
			URL:         url,
			Image: &discordgo.MessageEmbedImage{
				URL: img,
			},
		}

		msg.Embeds = append(msg.Embeds, embed)
	}

	return msg
}

func discord_tiktok_message(raw_url string, user_id string) *discordgo.MessageSend {
	url := tiktok_is_valid(raw_url)
	if url == "" {
		return nil
	}

	if tiktok_is_shortened(url) {
		url = tiktok_get_full_url(url)
	}

	id := tiktok_extract_id(url)
	json, is_image, err := tiktok_extract_json(id)

	if err != nil {
		log.Printf("Couldn't retrieve TikTok: %s\n", err)

		return nil
	} else {
		if is_image == true {
			return discord_prepare_images(url, user_id, json)
		} else {
			return discord_prepare_video(url, user_id, json)
		}
	}
}

func discord_autodetect_link(s *discordgo.Session, m *discordgo.MessageCreate) {
	/* Ignoring the bot's own messages */
	if m.Author.ID == s.State.User.ID {
		return
	}

	link := tiktok_is_valid(m.Content)
	if link == "" {
		return
	}

	log.Println("Detected a tiktok link in a message, replying to it")
	message := discord_tiktok_message(link, m.Author.ID)

	var err error
	if message != nil {
		_, err = s.ChannelMessageSendComplex(m.ChannelID, message)
	}
	if err != nil {
		log.Printf("Coudln't send message %s\n", err)
	}

	err = s.ChannelMessageDelete(m.ChannelID, m.ID)

	if err != nil {
		log.Printf("Couldn't delete message, %s", err)
	}
}

func discord_tiktok_slash_command(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	message := discord_tiktok_message(optionMap["link"].StringValue(), "")

	if message != nil {
		interaction := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: message.Content,
			},
		}
		s.InteractionRespond(i.Interaction, interaction)
	}
}

func discord_init_bot(token string) (*discordgo.Session, *discordgo.ApplicationCommand) {
	bot, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Couldn't initialize Discord bot")
	}

	command := discordgo.ApplicationCommand{
		Name:        "tiktok",
		Description: "Embeds a TikTok video with a direct link to it",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "link",
				Description: "Video link",
				Required:    true,
			},
		},
	}

	bot.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as %s\n", s.State.User.Username)
	})

	err = bot.Open()
	if err != nil {
		log.Fatalf("Couldn't open session: %s", err)
	}

	log.Println("Adding command")
	bot.AddHandler(discord_tiktok_slash_command)
	bot.AddHandler(discord_autodetect_link)
	cmd_tiktok, err := bot.ApplicationCommandCreate(bot.State.User.ID, "", &command)
	if err != nil {
		log.Fatalf("Couldn't add command %s: %s", cmd_tiktok.Name, err)
	}

	return bot, cmd_tiktok
}

func discord_run_bot(bot *discordgo.Session, command *discordgo.ApplicationCommand) {
	defer bot.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	log.Println("Removing command")
	bot.ApplicationCommandDelete(bot.State.User.ID, "", command.ID)
}

package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-plugin-msteams-sync/server/markdown"
	"github.com/mattermost/mattermost-plugin-msteams-sync/server/msteams"
	"github.com/mattermost/mattermost-server/v6/model"
	"golang.org/x/net/html"
)

func (ah *ActivityHandler) GetAvatarURL(userID string) string {
	defaultAvatarURL := ah.plugin.GetURL() + "/public/msteams-sync-icon.svg"
	resp, err := http.Get(ah.plugin.GetURL() + "/avatar/" + userID)
	if err != nil {
		ah.plugin.GetAPI().LogDebug("Unable to get user avatar", "Error", err.Error())
		return defaultAvatarURL
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if strings.Contains(resp.Header.Get("Content-Type"), "image") {
		return ah.plugin.GetURL() + "/avatar/" + userID
	}

	return defaultAvatarURL
}

func (ah *ActivityHandler) msgToPost(userID, channelID string, msg *msteams.Message, senderID string) (*model.Post, error) {
	text := ah.handleMentions(msg)
	text = ah.handleEmojis(text)
	text = markdown.ConvertToMD(text)
	props := make(map[string]interface{})
	rootID := ""

	if msg.ReplyToID != "" {
		rootInfo, _ := ah.plugin.GetStore().GetPostInfoByMSTeamsID(msg.ChatID+msg.ChannelID, msg.ReplyToID)
		if rootInfo != nil {
			rootID = rootInfo.MattermostID
		}
	}

	newText, attachments, parentID := ah.handleAttachments(userID, channelID, text, msg)
	text = newText
	if parentID != "" {
		rootID = parentID
	}

	if rootID == "" && msg.Subject != "" {
		text = "## " + msg.Subject + "\n" + text
	}

	post := &model.Post{UserId: senderID, ChannelId: channelID, Message: text, Props: props, RootId: rootID, FileIds: attachments}
	post.AddProp("msteams_sync_"+ah.plugin.GetBotUserID(), true)

	if senderID == ah.plugin.GetBotUserID() {
		post.AddProp("override_username", msg.UserDisplayName)
		post.AddProp("from_webhook", "true")
		post.AddProp("override_icon_url", ah.GetAvatarURL(msg.UserID))
	}
	return post, nil
}

func (ah *ActivityHandler) handleMentions(msg *msteams.Message) string {
	for _, mention := range msg.Mentions {
		mmMention := ""
		if mention.UserID != "" {
			mmUserID, err := ah.plugin.GetStore().TeamsToMattermostUserID(mention.UserID)
			if err != nil {
				ah.plugin.GetAPI().LogDebug("Unable to get mm UserID", "Error", err.Error())
				continue
			}

			mmUser, getErr := ah.plugin.GetAPI().GetUser(mmUserID)
			if getErr != nil {
				ah.plugin.GetAPI().LogDebug("Unable to get mm user details", "Error", getErr.Error())
				continue
			}

			mmMention = fmt.Sprintf("@%s ", mmUser.Username)
		} else {
			if mention.MentionedText == "Everyone" {
				mmMention = "@all"
			} else {
				mmMention = "@channel"
			}
		}

		msg.Text = strings.Replace(msg.Text, fmt.Sprintf("<at id=\"%s\">%s</at>", fmt.Sprint(mention.ID), mention.MentionedText), mmMention, 1)
	}

	return msg.Text
}

func (ah *ActivityHandler) handleEmojis(text string) string {
	emojisData := strings.Split(text, "</emoji>")

	for idx, emojiData := range emojisData {
		if idx == len(emojisData)-1 {
			break
		}

		emojiIdx := strings.Index(emojiData, "<emoji")
		if emojiIdx != -1 {
			emojiData = emojiData[emojiIdx:] + "</emoji>"
			doc, err := html.Parse(strings.NewReader(emojiData))
			if err != nil {
				ah.plugin.GetAPI().LogWarn("Unable to parse emoji data", "EmojiData", emojiData, "Error", err.Error())
				continue
			}

			for _, a := range doc.FirstChild.FirstChild.NextSibling.FirstChild.Attr {
				if a.Key == "alt" {
					text = strings.Replace(text, emojiData, a.Val, 1)
					break
				}
			}
		}
	}

	return text
}

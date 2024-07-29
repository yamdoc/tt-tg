package internal

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/heilkit/tg"
	"github.com/heilkit/tg/tgvideo"
	"github.com/yamdoc/tt/tt"
)

func (manager *Manager) HandlePost(p *tt.Post, threadId int) error {
	temp, err := os.MkdirTemp("", "*_tt-tg")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(temp)

	if p.IsVideo() {
		post, files, err := tt.Download(p.Id, &tt.DownloadOpt{
			Directory: temp,
			Retries:   4,
			Log:       slog.Default(),
		})
		defer func() {
			for _, file := range files {
				if err := os.Remove(file); err != nil {
					slog.Error("failed to remove temporary file", "file", file, "error", err)
				}
			}
		}()
		if err != nil {
			if _, err := manager.tg.Send(manager.chat, fmt.Sprintf("#e %s", p.Id), &tg.SendOptions{ThreadID: threadId}); err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}
			return nil
		}
		vid := tg.Video{
			File:     tg.FromDisk(files[0]),
			FileName: fmt.Sprintf("%s_%s_%s.mp4", p.Author.UniqueId, time.Unix(p.CreateTime, 0).Format(time.DateOnly), post.Id),
		}.
			With(tgvideo.ThumbnailAt("0.05")).
			With(tgvideo.EmbedMetadata(post))
		if _, err := manager.tg.Send(manager.chat, vid, &tg.SendOptions{ThreadID: threadId}); err != nil {
			return fmt.Errorf("failed to send video: %w", err)
		}

		return nil
	}

	_, files, err := tt.Download(p.Id, &tt.DownloadOpt{
		Directory: temp,
		FilenameFormat: func(post *tt.Post, i int) string {
			return fmt.Sprintf("@%s_%s_%d.jpg", post.Author.UniqueId, time.Unix(post.CreateTime, 0).Format(time.DateOnly), i)
		},
		Retries: 4,
		Log:     slog.Default(),
	})
	if err != nil {
		if _, err := manager.tg.Send(manager.chat, fmt.Sprintf("#e %s", p.Id), &tg.SendOptions{ThreadID: threadId}); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
		return nil
	}
	for _, file := range files {
		if _, err := manager.tg.Send(manager.chat, &tg.Document{File: tg.FromDisk(file), FileName: filepath.Base(file)}, &tg.SendOptions{ThreadID: threadId}); err != nil {
			return fmt.Errorf("failed to send document: %w", err)
		}
	}

	return nil
}

func (manager *Manager) Profile(profile *Profile) error {
	if profile.UserId == "" {
		slog.Info("no user id, getting it", "profile", profile.Tag)
		info, err := tt.GetUserDetail(profile.Username)
		if err != nil {
			return fmt.Errorf("failed to get user info: %w", err)
		}
		profile.UserId = info.User.Id
		if err := manager.Config.Update(); err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}
	}

	if profile.Thread == 0 {
		slog.Info("no thread id, creating a thread", "profile", profile.Tag)
		topic, err := manager.tg.CreateTopic(manager.chat, &tg.Topic{Name: profile.Tag})
		if err != nil {
			return fmt.Errorf("failed to create topic: %w", err)
		}
		profile.Thread = topic.ThreadID
		if err := manager.Config.Update(); err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}
	}

	postChan, expectedCount, err := tt.GetUserFeed(profile.Username, tt.FeedOpt{
		While: tt.WhileAfter(profile.LastUpload),
		OnError: func(err error) {
			if err != nil {
				slog.Error("failed to get user feed", "err", err)
			}
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get user feed: %w", err)
	}
	if expectedCount == 0 {
		slog.Info("No updates", "user", profile.Tag)
		return nil
	}

	i := 0
	var post tt.Post
	for post = range postChan {
		i += 1
		slog.Info(fmt.Sprintf("Getting post [%d/%d]", i, expectedCount), "user", profile.Tag)
		if err := manager.HandlePost(&post, profile.Thread); err != nil {
			return fmt.Errorf("failed to handle post: %w (%s)", err, post.Id)
		}
	}
	profile.LastUpload = time.Unix(post.CreateTime, 0)
	if err := manager.Config.Update(); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	return nil
}

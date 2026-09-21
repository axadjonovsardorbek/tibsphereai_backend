package push

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type Client struct {
	msg *messaging.Client
}

// NewClientFromServiceAccountFile: serviceAccount.json fayl orqali init
func NewClientFromServiceAccountFile(ctx context.Context, serviceAccountPath string) (*Client, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(serviceAccountPath))
	if err != nil {
		fmt.Println("Error initializing firebase app:", err)
		return nil, fmt.Errorf("firebase.NewApp: %w", err)
	}
	msg, err := app.Messaging(ctx)
	if err != nil {
		fmt.Println("Error initializing firebase messaging client:", err)
		return nil, fmt.Errorf("app.Messaging: %w", err)
	}
	return &Client{msg: msg}, nil
}

// SendToToken: bitta userga push (token bitta)
func (c *Client) SendToToken(ctx context.Context, token, title, body string, data map[string]string) error {
	if c == nil || c.msg == nil {
		return fmt.Errorf("push client not initialized")
	}
	if token == "" {
		return fmt.Errorf("empty fcm token")
	}

	_, err := c.msg.Send(ctx, &messaging.Message{
		Token: token,
		Data:  data,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	})
	if err != nil {
		return fmt.Errorf("fcm send: %w", err)
	}
	return nil
}

// SendToTopic - "all" broadcast uchun (til bo'yicha topic: all_uz/all_ru/all_en,
// pkg/models.BroadcastTopic). Devicelar shu topicga Register paytida obuna bo'ladi.
func (c *Client) SendToTopic(ctx context.Context, topic, title, body string, data map[string]string) error {
	if c == nil || c.msg == nil {
		return fmt.Errorf("push client not initialized")
	}
	if topic == "" {
		return fmt.Errorf("empty topic")
	}

	_, err := c.msg.Send(ctx, &messaging.Message{
		Topic: topic,
		Data:  data,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	})
	if err != nil {
		return fmt.Errorf("fcm send to topic: %w", err)
	}
	return nil
}

// SubscribeToTopic - device Register bo'lganda foydalanuvchi tiliga mos topicga obuna qiladi.
func (c *Client) SubscribeToTopic(ctx context.Context, tokens []string, topic string) error {
	if c == nil || c.msg == nil {
		return fmt.Errorf("push client not initialized")
	}
	if len(tokens) == 0 || topic == "" {
		return nil
	}
	if _, err := c.msg.SubscribeToTopic(ctx, tokens, topic); err != nil {
		return fmt.Errorf("fcm subscribe to topic: %w", err)
	}
	return nil
}

// UnsubscribeFromTopic - device Unregister yoki til o'zgarganda eski topicdan chiqaradi.
func (c *Client) UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error {
	if c == nil || c.msg == nil {
		return fmt.Errorf("push client not initialized")
	}
	if len(tokens) == 0 || topic == "" {
		return nil
	}
	if _, err := c.msg.UnsubscribeFromTopic(ctx, tokens, topic); err != nil {
		return fmt.Errorf("fcm unsubscribe from topic: %w", err)
	}
	return nil
}

// MulticastResult: multi-user yuborishda umumiy natija
type MulticastResult struct {
	Total     int
	Successes int
	Failures  int
	// InvalidTokens: tokenlar “endilikda yaroqsiz” bo‘lsa (DB’dan tozalash uchun)
	InvalidTokens []string
}

// SendToManyTokens: ko‘p userga push (tokenlar list)
// Batching (500) bilan yuboradi. Invalid tokenlarni qaytaradi.
func (c *Client) SendToManyTokens(ctx context.Context, tokens []string, title, body string, data map[string]string) (MulticastResult, error) {
	if c == nil || c.msg == nil {
		return MulticastResult{}, fmt.Errorf("push client not initialized")
	}

	// bo‘sh tokenlarni olib tashlaymiz
	filtered := make([]string, 0, len(tokens))
	seen := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		filtered = append(filtered, t)
	}

	res := MulticastResult{Total: len(filtered)}
	if res.Total == 0 {
		return res, nil
	}

	const batchSize = 500
	for i := 0; i < len(filtered); i += batchSize {
		j := i + batchSize
		if j > len(filtered) {
			j = len(filtered)
		}
		batch := filtered[i:j]

		br, err := c.msg.SendEachForMulticast(ctx, &messaging.MulticastMessage{
			Tokens: batch,
			Data:   data,
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
		})
		if err != nil {
			// butun batch fail bo‘lsa ham umumiy error qaytaramiz
			return res, fmt.Errorf("fcm multicast batch send: %w", err)
		}

		res.Successes += br.SuccessCount
		res.Failures += br.FailureCount

		for idx, r := range br.Responses {
			if r.Success {
				continue
			}
			if messaging.IsUnregistered(r.Error) || messaging.IsInvalidArgument(r.Error) {
				res.InvalidTokens = append(res.InvalidTokens, batch[idx])
			}
		}
	}

	return res, nil
}

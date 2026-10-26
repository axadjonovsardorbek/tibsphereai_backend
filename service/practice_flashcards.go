package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"tibsphereai/internal/ai"
	m "tibsphereai/pkg/models"
	"time"
)

const (
	maxMistakesForCards = 8
	mistakesDeckTitle   = "Xatolarim"
)

// mistakesFrom - sessiyadagi xato yoki qisman javoblar (fleshkarta yaratish uchun asos).
func mistakesFrom(items []*m.PracticeItemRow) ([]m.FlashcardMistake, []string, []string) {
	var (
		out      []m.FlashcardMistake
		chunkIds []string
		topics   []string
	)
	seenChunk := map[string]bool{}
	for i := len(items) - 1; i >= 0 && len(out) < maxMistakesForCards; i-- { // oxirgilaridan boshlab
		it := items[i]
		if it.IsCorrect == nil || *it.IsCorrect {
			continue
		}
		var mk m.FlashcardMistake
		mk.Kind = it.Kind
		var ans map[string]any
		_ = json.Unmarshal(it.UserAnswer, &ans)
		if it.Kind == m.PracticeModeCase {
			var c m.PracticeCaseGen
			if json.Unmarshal(it.Payload, &c) != nil {
				continue
			}
			pj, _ := json.Marshal(c.Patient)
			mk.Prompt = c.ChiefComplaint + " | " + truncateRunes(string(pj), 400)
			mk.CorrectAnswer, mk.Explanation = c.ExpectedAnswer, c.Explanation
			mk.UserAnswer, _ = ans["text"].(string)
		} else {
			var t m.PracticeTestGen
			if json.Unmarshal(it.Payload, &t) != nil {
				continue
			}
			mk.Prompt = t.Question
			mk.Explanation = t.Explanation
			if t.CorrectIndex >= 0 && t.CorrectIndex < len(t.Options) {
				mk.CorrectAnswer = t.Options[t.CorrectIndex]
			}
			if key, _ := ans["key"].(string); key != "" {
				if idx := optionIndex(key); idx >= 0 && idx < len(t.Options) {
					mk.UserAnswer = t.Options[idx]
				}
			}
		}
		out = append(out, mk)
		for _, c := range it.SourceChunkIds {
			if !seenChunk[c] && len(chunkIds) < 8 {
				seenChunk[c] = true
				chunkIds = append(chunkIds, c)
			}
		}
		if it.TopicId != "" {
			topics = append(topics, it.TopicId)
		}
	}
	return out, chunkIds, topics
}

// dominant - eng ko'p uchragan qiymat (bo'sh bo'lsa "").
func dominant(vals []string) string {
	cnt := map[string]int{}
	best, bestN := "", 0
	for _, v := range vals {
		cnt[v]++
		if cnt[v] > bestN {
			best, bestN = v, cnt[v]
		}
	}
	return best
}

// startSessionFlashcards - sessiya yakunlangach xatolar asosida fleshkartalarni fonda yaratadi
// (TZ_MVP: "har test va keysdan keyin AI xatolarni tahlil qilib yangi kartalar yaratib boradi").
// Xato bo'lmasa hech narsa qilinmaydi (status none). Narx sessiya tokenlariga qo'shiladi.
func (s *PracticeService) startSessionFlashcards(sess *m.PracticeSessionRow) {
	items, err := s.storage.PracticeS.GetItems(sess.Id)
	if err != nil {
		return
	}
	mistakes, _, _ := mistakesFrom(items)
	if len(mistakes) == 0 || s.ai == nil || s.ai.Flashcards == nil || s.storage.FlashcardS == nil {
		return
	}
	_ = s.storage.PracticeS.SetFlashcards(sess.Id, "pending", 0)
	job := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		n, err := s.generateSessionFlashcards(ctx, sess, items)
		if err != nil {
			fmt.Println("session flashcards failed:", sess.Id, err)
			_ = s.storage.PracticeS.SetFlashcards(sess.Id, "failed", 0)
			return
		}
		_ = s.storage.PracticeS.SetFlashcards(sess.Id, "done", n)
	}
	if s.asyncFlashcards {
		go job()
	} else {
		job()
	}
}

func (s *PracticeService) generateSessionFlashcards(ctx context.Context, sess *m.PracticeSessionRow, items []*m.PracticeItemRow) (int, error) {
	mistakes, chunkIds, topics := mistakesFrom(items)
	in := &m.FlashcardGenInput{
		Mode: m.FlashcardKindMistakes, Language: sess.Language, Mistakes: mistakes,
		Count: min(m.FlashcardMaxPerSession, 2*len(mistakes)),
	}
	if s.storage.SourceS != nil {
		in.Chunks, _ = s.storage.SourceS.GetChunks(chunkIds)
	}
	in.Existing, _ = s.storage.FlashcardS.ExistingFronts(sess.UserId, existingFrontsForPrompt)

	var (
		gen    []m.FlashcardGen
		usage  *ai.Usage
		genErr error
	)
	for attempt := 0; attempt < 3; attempt++ {
		var u *ai.Usage
		gen, u, genErr = s.ai.Flashcards.GenerateFlashcards(ctx, in)
		usage = addUsage(usage, u)
		if genErr == nil {
			break
		}
		fmt.Println("session flashcard generation failed (attempt", attempt+1, "):", genErr)
	}
	s.recordUsage(ctx, sess.Id, sess.UserId, "flashcard_generate", usage)
	if genErr != nil {
		return 0, genErr
	}

	topicId := dominant(topics)
	if len(sess.TopicIds) == 1 {
		topicId = sess.TopicIds[0]
	}
	deckId, err := s.mistakesDeck(sess)
	if err != nil {
		return 0, err
	}
	rows := make([]*m.FlashcardRow, 0, len(gen))
	for _, g := range gen {
		refs, _ := BuildSourceRefs(in.Chunks, g.UsedSources)
		rows = append(rows, &m.FlashcardRow{
			UserId: sess.UserId, DeckId: deckId, Front: g.Front, Back: g.Back, Sources: refs,
			OriginSessionId: sess.Id, CategoryId: sess.CategoryId, TopicId: topicId, Language: sess.Language,
		})
	}
	return s.storage.FlashcardS.AddCards(rows)
}

// mistakesDeck - foydalanuvchining doimiy "Xatolarim" to'plami (bo'lmasa yaratadi).
func (s *PracticeService) mistakesDeck(sess *m.PracticeSessionRow) (string, error) {
	decks, err := s.storage.FlashcardS.ListDecks(sess.UserId)
	if err != nil {
		return "", err
	}
	for _, d := range decks {
		if d.Kind == m.FlashcardKindMistakes {
			return d.Id, nil
		}
	}
	return s.storage.FlashcardS.CreateDeck(&m.FlashcardDeckRow{
		UserId: sess.UserId, Title: mistakesDeckTitle, Kind: m.FlashcardKindMistakes, Language: strings.ToLower(sess.Language),
	})
}

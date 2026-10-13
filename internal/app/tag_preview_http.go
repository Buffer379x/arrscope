package app

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"time"
)

func (s *Server) tagPreviewHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	var in struct {
		Service string        `json:"service"`
		Config  ServiceConfig `json:"config"`
		Token   string        `json:"token"`
		TagID   int           `json:"tagId"`
		Page    int           `json:"page"`
	}
	if e := decode(w, r, &in); e != nil {
		failure(w, r, 400, e)
		return
	}
	if !serviceOK(in.Service) {
		failure(w, r, 400, errors.New("Invalid service"))
		return
	}
	if s.Demo {
		failure(w, r, 403, errors.New("Demo connections cannot be changed"))
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(130 * time.Second))
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	select {
	case s.Sync.gate <- struct{}{}:
		defer func() { <-s.Sync.gate }()
	case <-ctx.Done():
		failure(w, r, 503, errors.New("Sync is busy. Try again after it completes"))
		return
	}
	cfg := s.Config()
	before := cfg.Radarr
	if in.Service == "sonarr" {
		before = cfg.Sonarr
	}
	next := in.Config
	if r.URL.Path == "/api/tag-approval" {
		next = before
	} else {
		if e := mergeConnection(&next, before); e != nil {
			failure(w, r, 400, e)
			return
		}
		if e := validateTagPolicy(before, next); e != nil {
			failure(w, r, 400, e)
			return
		}
	}
	var old *ServiceConfig
	if r.URL.Path != "/api/tag-approval" {
		old = &before
	}
	p, e := s.Store.BuildTagPlan(ctx, in.Service, next, old)
	if e != nil {
		failure(w, r, 400, e)
		return
	}
	if p.Waiting {
		failure(w, r, 409, errors.New("Synchronize file data before reviewing outbound tags"))
		return
	}
	if in.Token != "" && in.Token != p.Preview.Token {
		failure(w, r, 409, ErrTagPreviewChanged)
		return
	}
	if r.URL.Path == "/api/tag-approval" {
		if in.Token == "" {
			failure(w, r, 400, errors.New("Review the tag preview before confirming"))
			return
		}
		if !reflect.DeepEqual(s.Config(), cfg) {
			failure(w, r, 409, ErrTagPreviewChanged)
			return
		}
		if e = s.Store.ApproveTagPlan(ctx, p); e != nil {
			failure(w, r, 500, errors.New("Could not save tag approval"))
			return
		}
		s.Tags.Queue(in.Service)
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	if in.TagID > 0 {
		page := in.Page
		if page < 1 {
			page = 1
		}
		targets := []TagTarget{}
		for _, op := range p.Operations {
			for _, id := range op.Remove {
				if id == in.TagID {
					targets = append(targets, TagTarget{op.ID, op.Title})
					break
				}
			}
		}
		maxPage := (len(targets) + 99) / 100
		if maxPage < 1 {
			maxPage = 1
		}
		if page > maxPage {
			page = maxPage
		}
		p.Preview.TargetTotal = len(targets)
		p.Preview.TargetPage = page
		start := (page - 1) * 100
		if start > len(targets) {
			start = len(targets)
		}
		end := start + 100
		if end > len(targets) {
			end = len(targets)
		}
		p.Preview.Targets = targets[start:end]
	}
	jsonOut(w, 200, p.Preview)
}

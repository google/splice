/*
Copyright 2018 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package endpoints

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/datastore"
	"splice/models"
)

func TestStartTx(t *testing.T) {
	existingTx := Client{tx: &datastore.Transaction{}}
	if err := existingTx.StartTx(context.Background()); err == nil {
		t.Errorf("StartTx() = %v, want err", err)
	}
}

func TestCommitTx(t *testing.T) {
	missingTx := Client{}
	if err := missingTx.CommitTx(); err == nil {
		t.Errorf("CommitTx() = %v, want err", err)
	}
}

func TestRollbackTx(t *testing.T) {
	missingTx := Client{}
	if err := missingTx.RollbackTx(); err != nil {
		t.Errorf("RollbackTx() = %v, want nil", err)
	}
}

func TestSave(t *testing.T) {
	tests := []struct {
		name string
		in   Client
	}{
		{"Empty Client", Client{}},
		{"Missing Request", Client{Keys: []*datastore.Key{&datastore.Key{}}}},
		{"Empty Tx", Client{tx: &datastore.Transaction{}, Req: &models.Request{}}},
	}

	for _, tt := range tests {
		if _, got := tt.in.Save(context.Background()); got == nil {
			t.Errorf("save() = %v, want err", got)
			continue
		}
	}
}

func TestFind(t *testing.T) {
	tests := []struct {
		name  string
		reqID string
		in    Client
	}{
		{"Empty Client", "abc", Client{}},
		{"Blank ReqID", "", Client{Req: &models.Request{}}},
		{"Empty Tx", "abc", Client{Req: &models.Request{}}},
	}

	for _, tt := range tests {
		if _, got := tt.in.Find(context.Background(), tt.reqID); got == nil {
			t.Errorf("find(%s) = %v, want: err", tt.reqID, got)
			continue
		}
	}
}

func TestFindOrphans(t *testing.T) {
	tests := []struct {
		name string
		days time.Duration
		kind string
		in   Client
	}{
		{"Empty Client", 1 * 24 * time.Hour, models.RequestStatusAccepted, Client{}},
		{"Negative Duration", -1 * 24 * time.Hour, models.RequestStatusAccepted, Client{Req: &models.Request{}}},
		{"Invalid Duration", 0, models.RequestStatusAccepted, Client{Req: &models.Request{}}},
		{"Empty Tx", 1 * 24 * time.Hour, models.RequestStatusAccepted, Client{Req: &models.Request{}}},
	}

	for _, tt := range tests {
		if _, _, got := tt.in.FindOrphans(context.Background(), tt.days, tt.kind); got == nil {
			t.Errorf("FindOrphans(%d,%s) = %v, want: err", tt.days, tt.kind, got)
			continue
		}
	}
}

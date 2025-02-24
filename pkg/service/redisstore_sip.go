// Copyright 2023 LiveKit, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"

	"github.com/a4to/protocol/conkit"
)

const (
	SIPTrunkKey         = "sip_trunk"
	SIPInboundTrunkKey  = "sip_inbound_trunk"
	SIPOutboundTrunkKey = "sip_outbound_trunk"
	SIPDispatchRuleKey  = "sip_dispatch_rule"
)

func (s *RedisStore) StoreSIPTrunk(ctx context.Context, info *conkit.SIPTrunkInfo) error {
	return redisStoreOne(s.ctx, s, SIPTrunkKey, info.SipTrunkId, info)
}

func (s *RedisStore) StoreSIPInboundTrunk(ctx context.Context, info *conkit.SIPInboundTrunkInfo) error {
	return redisStoreOne(s.ctx, s, SIPInboundTrunkKey, info.SipTrunkId, info)
}

func (s *RedisStore) StoreSIPOutboundTrunk(ctx context.Context, info *conkit.SIPOutboundTrunkInfo) error {
	return redisStoreOne(s.ctx, s, SIPOutboundTrunkKey, info.SipTrunkId, info)
}

func (s *RedisStore) loadSIPLegacyTrunk(ctx context.Context, id string) (*conkit.SIPTrunkInfo, error) {
	return redisLoadOne[conkit.SIPTrunkInfo](ctx, s, SIPTrunkKey, id, ErrSIPTrunkNotFound)
}

func (s *RedisStore) loadSIPInboundTrunk(ctx context.Context, id string) (*conkit.SIPInboundTrunkInfo, error) {
	return redisLoadOne[conkit.SIPInboundTrunkInfo](ctx, s, SIPInboundTrunkKey, id, ErrSIPTrunkNotFound)
}

func (s *RedisStore) loadSIPOutboundTrunk(ctx context.Context, id string) (*conkit.SIPOutboundTrunkInfo, error) {
	return redisLoadOne[conkit.SIPOutboundTrunkInfo](ctx, s, SIPOutboundTrunkKey, id, ErrSIPTrunkNotFound)
}

func (s *RedisStore) LoadSIPTrunk(ctx context.Context, id string) (*conkit.SIPTrunkInfo, error) {
	tr, err := s.loadSIPLegacyTrunk(ctx, id)
	if err == nil {
		return tr, nil
	} else if err != ErrSIPTrunkNotFound {
		return nil, err
	}
	in, err := s.loadSIPInboundTrunk(ctx, id)
	if err == nil {
		return in.AsTrunkInfo(), nil
	} else if err != ErrSIPTrunkNotFound {
		return nil, err
	}
	out, err := s.loadSIPOutboundTrunk(ctx, id)
	if err == nil {
		return out.AsTrunkInfo(), nil
	} else if err != ErrSIPTrunkNotFound {
		return nil, err
	}
	return nil, ErrSIPTrunkNotFound
}

func (s *RedisStore) LoadSIPInboundTrunk(ctx context.Context, id string) (*conkit.SIPInboundTrunkInfo, error) {
	in, err := s.loadSIPInboundTrunk(ctx, id)
	if err == nil {
		return in, nil
	} else if err != ErrSIPTrunkNotFound {
		return nil, err
	}
	tr, err := s.loadSIPLegacyTrunk(ctx, id)
	if err == nil {
		return tr.AsInbound(), nil
	} else if err != ErrSIPTrunkNotFound {
		return nil, err
	}
	return nil, ErrSIPTrunkNotFound
}

func (s *RedisStore) LoadSIPOutboundTrunk(ctx context.Context, id string) (*conkit.SIPOutboundTrunkInfo, error) {
	in, err := s.loadSIPOutboundTrunk(ctx, id)
	if err == nil {
		return in, nil
	} else if err != ErrSIPTrunkNotFound {
		return nil, err
	}
	tr, err := s.loadSIPLegacyTrunk(ctx, id)
	if err == nil {
		return tr.AsOutbound(), nil
	} else if err != ErrSIPTrunkNotFound {
		return nil, err
	}
	return nil, ErrSIPTrunkNotFound
}

func (s *RedisStore) DeleteSIPTrunk(ctx context.Context, id string) error {
	tx := s.rc.TxPipeline()
	tx.HDel(s.ctx, SIPTrunkKey, id)
	tx.HDel(s.ctx, SIPInboundTrunkKey, id)
	tx.HDel(s.ctx, SIPOutboundTrunkKey, id)
	_, err := tx.Exec(ctx)
	return err
}

func (s *RedisStore) listSIPLegacyTrunk(ctx context.Context, page *conkit.Pagination) ([]*conkit.SIPTrunkInfo, error) {
	return redisIterPage[conkit.SIPTrunkInfo](ctx, s, SIPTrunkKey, page)
}

func (s *RedisStore) listSIPInboundTrunk(ctx context.Context, page *conkit.Pagination) ([]*conkit.SIPInboundTrunkInfo, error) {
	return redisIterPage[conkit.SIPInboundTrunkInfo](ctx, s, SIPInboundTrunkKey, page)
}

func (s *RedisStore) listSIPOutboundTrunk(ctx context.Context, page *conkit.Pagination) ([]*conkit.SIPOutboundTrunkInfo, error) {
	return redisIterPage[conkit.SIPOutboundTrunkInfo](ctx, s, SIPOutboundTrunkKey, page)
}

func (s *RedisStore) listSIPDispatchRule(ctx context.Context, page *conkit.Pagination) ([]*conkit.SIPDispatchRuleInfo, error) {
	return redisIterPage[conkit.SIPDispatchRuleInfo](ctx, s, SIPDispatchRuleKey, page)
}

func (s *RedisStore) ListSIPTrunk(ctx context.Context, req *conkit.ListSIPTrunkRequest) (*conkit.ListSIPTrunkResponse, error) {
	var items []*conkit.SIPTrunkInfo
	old, err := s.listSIPLegacyTrunk(ctx, req.Page)
	if err != nil {
		return nil, err
	}
	for _, t := range old {
		v := t
		if req.Filter(v) && req.Page.Filter(v) {
			items = append(items, v)
		}
	}
	in, err := s.listSIPInboundTrunk(ctx, req.Page)
	if err != nil {
		return nil, err
	}
	for _, t := range in {
		v := t.AsTrunkInfo()
		if req.Filter(v) && req.Page.Filter(v) {
			items = append(items, v)
		}
	}
	out, err := s.listSIPOutboundTrunk(ctx, req.Page)
	if err != nil {
		return nil, err
	}
	for _, t := range out {
		v := t.AsTrunkInfo()
		if req.Filter(v) && req.Page.Filter(v) {
			items = append(items, v)
		}
	}
	items = sortPage(items, req.Page)
	return &conkit.ListSIPTrunkResponse{Items: items}, nil
}

func (s *RedisStore) ListSIPInboundTrunk(ctx context.Context, req *conkit.ListSIPInboundTrunkRequest) (*conkit.ListSIPInboundTrunkResponse, error) {
	var items []*conkit.SIPInboundTrunkInfo
	in, err := s.listSIPInboundTrunk(ctx, req.Page)
	if err != nil {
		return nil, err
	}
	for _, t := range in {
		v := t
		if req.Filter(v) && req.Page.Filter(v) {
			items = append(items, v)
		}
	}
	old, err := s.listSIPLegacyTrunk(ctx, req.Page)
	if err != nil {
		return nil, err
	}
	for _, t := range old {
		v := t.AsInbound()
		if req.Filter(v) && req.Page.Filter(v) {
			items = append(items, v)
		}
	}
	items = sortPage(items, req.Page)
	return &conkit.ListSIPInboundTrunkResponse{Items: items}, nil
}

func (s *RedisStore) ListSIPOutboundTrunk(ctx context.Context, req *conkit.ListSIPOutboundTrunkRequest) (*conkit.ListSIPOutboundTrunkResponse, error) {
	var items []*conkit.SIPOutboundTrunkInfo
	out, err := s.listSIPOutboundTrunk(ctx, req.Page)
	if err != nil {
		return nil, err
	}
	for _, t := range out {
		v := t
		if req.Filter(v) && req.Page.Filter(v) {
			items = append(items, v)
		}
	}
	old, err := s.listSIPLegacyTrunk(ctx, req.Page)
	if err != nil {
		return nil, err
	}
	for _, t := range old {
		v := t.AsOutbound()
		if req.Filter(v) && req.Page.Filter(v) {
			items = append(items, v)
		}
	}
	items = sortPage(items, req.Page)
	return &conkit.ListSIPOutboundTrunkResponse{Items: items}, nil
}

func (s *RedisStore) StoreSIPDispatchRule(ctx context.Context, info *conkit.SIPDispatchRuleInfo) error {
	return redisStoreOne(ctx, s, SIPDispatchRuleKey, info.SipDispatchRuleId, info)
}

func (s *RedisStore) LoadSIPDispatchRule(ctx context.Context, sipDispatchRuleId string) (*conkit.SIPDispatchRuleInfo, error) {
	return redisLoadOne[conkit.SIPDispatchRuleInfo](ctx, s, SIPDispatchRuleKey, sipDispatchRuleId, ErrSIPDispatchRuleNotFound)
}

func (s *RedisStore) DeleteSIPDispatchRule(ctx context.Context, sipDispatchRuleId string) error {
	return s.rc.HDel(s.ctx, SIPDispatchRuleKey, sipDispatchRuleId).Err()
}

func (s *RedisStore) ListSIPDispatchRule(ctx context.Context, req *conkit.ListSIPDispatchRuleRequest) (*conkit.ListSIPDispatchRuleResponse, error) {
	var items []*conkit.SIPDispatchRuleInfo
	out, err := s.listSIPDispatchRule(ctx, req.Page)
	if err != nil {
		return nil, err
	}
	for _, t := range out {
		v := t
		if req.Filter(v) && req.Page.Filter(v) {
			items = append(items, v)
		}
	}
	items = sortPage(items, req.Page)
	return &conkit.ListSIPDispatchRuleResponse{Items: items}, nil
}

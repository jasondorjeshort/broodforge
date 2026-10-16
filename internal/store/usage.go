// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"encoding/json"
	"strings"
)

// UsageFilter narrows a usage query (05-debug-interface.md, Usage and audit queries). At most
// one of Session, User (the requester), Credential (the payer), Channel and Project is set;
// none is the instance total (02-domain.md, Usage record). Since is a stored time; rows
// recorded at or after it count.
type UsageFilter struct {
	Session    string
	User       string
	Credential string
	Channel    string
	Project    string
	Since      string
}

func (f UsageFilter) where() (string, []Value) {
	var conds []string
	var args []Value
	add := func(col, v string) {
		if v != "" {
			conds = append(conds, col+" = ?")
			args = append(args, Text(v))
		}
	}
	add("session_id", f.Session)
	add("requester_id", f.User)
	add("credential_id", f.Credential)
	add("channel_id", f.Channel)
	add("project_id", f.Project)
	if f.Since != "" {
		conds = append(conds, "recorded_at >= ?")
		args = append(args, Text(f.Since))
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// UsageTotal is one model's totals, or the total over every model.
type UsageTotal struct {
	Model       string `json:"model"`
	Turns       int64  `json:"turns"`
	Input       int64  `json:"input"`
	Output      int64  `json:"output"`
	CacheRead   int64  `json:"cacheRead"`
	CacheWrite  int64  `json:"cacheWrite"`
	Interrupted int64  `json:"interruptedTurns"`
}

// TotalModel is the model column of the total line.
const TotalModel = "total"

const usageSums = `count(DISTINCT session_id || ' ' || turn_seq),
    coalesce(sum(input_tokens), 0), coalesce(sum(output_tokens), 0),
    coalesce(sum(cache_read_tokens), 0), coalesce(sum(cache_write_tokens), 0),
    count(DISTINCT CASE WHEN interrupted THEN session_id || ' ' || turn_seq END)`

func scanTotal(model string, r *Row, off int) UsageTotal {
	return UsageTotal{Model: model, Turns: r.Int(off), Input: r.Int(off + 1), Output: r.Int(off + 2),
		CacheRead: r.Int(off + 3), CacheWrite: r.Int(off + 4), Interrupted: r.Int(off + 5)}
}

// UsageByModel returns one total per model, by model name, and the total over all of them.
// A turn counts once per model it used and once in the total.
func UsageByModel(q Querier, f UsageFilter) ([]UsageTotal, UsageTotal, error) {
	where, args := f.where()
	var rows []UsageTotal
	err := q.Query("SELECT model, "+usageSums+" FROM usage_record"+where+" GROUP BY model ORDER BY model", args,
		func(r *Row) error {
			rows = append(rows, scanTotal(r.Text(0), r, 1))
			return nil
		})
	if err != nil {
		return nil, UsageTotal{}, err
	}
	total := UsageTotal{Model: TotalModel}
	err = q.Query("SELECT "+usageSums+" FROM usage_record"+where, args, func(r *Row) error {
		total = scanTotal(TotalModel, r, 0)
		return nil
	})
	return rows, total, err
}

// TurnUsage is one turn's usage for one model, with the harness's raw report.
type TurnUsage struct {
	Session     string          `json:"session"`
	Turn        int64           `json:"turn"`
	Model       string          `json:"model"`
	Input       int64           `json:"input"`
	Output      int64           `json:"output"`
	CacheRead   int64           `json:"cacheRead"`
	CacheWrite  int64           `json:"cacheWrite"`
	Interrupted bool            `json:"interrupted"`
	Time        string          `json:"time"`
	Raw         json.RawMessage `json:"raw"`
}

// UsageByTurn returns one row per turn and model, in session and turn order.
func UsageByTurn(q Querier, f UsageFilter) ([]TurnUsage, error) {
	where, args := f.where()
	var out []TurnUsage
	err := q.Query(`SELECT session_id, turn_seq, model, sum(input_tokens), sum(output_tokens),
    sum(cache_read_tokens), sum(cache_write_tokens), max(interrupted), max(recorded_at),
    json_group_array(json(raw))
FROM usage_record`+where+` GROUP BY session_id, turn_seq, model ORDER BY session_id, turn_seq, model`, args,
		func(r *Row) error {
			raw := json.RawMessage(r.Text(9))
			if !json.Valid(raw) {
				raw = json.RawMessage("[]")
			}
			out = append(out, TurnUsage{Session: r.Text(0), Turn: r.Int(1), Model: r.Text(2), Input: r.Int(3),
				Output: r.Int(4), CacheRead: r.Int(5), CacheWrite: r.Int(6), Interrupted: r.Int(7) != 0,
				Time: r.Text(8), Raw: raw})
			return nil
		})
	return out, err
}

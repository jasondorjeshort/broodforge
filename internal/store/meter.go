// SPDX-License-Identifier: GPL-3.0-or-later

package store

// MeterRow is one credential's plan meter (migration 0004, ADR 0021 Amendments 2026-10-30):
// the latest rate-limit event of a turn it paid for, as read, zero where a field was absent.
// Utilizations are fractions, the resets unix seconds, SeenAt a stored time.
type MeterRow struct {
	Credential          string
	SeenAt              string
	Session             string
	Status              string
	RateLimitType       string
	FiveHourUtilization float64
	FiveHourResetsAt    int64
	SevenDayUtilization float64
	SevenDayResetsAt    int64
	OverageStatus       string
	IsUsingOverage      bool
}

// SaveMeter records a credential's meter in its own short transaction, replacing the row it
// had (an upsert): the bot's writer, on a rate-limit event in a turn's stream.
func SaveMeter(s *Store, m MeterRow) error {
	return s.Write(func(tx *Tx) error {
		using := int64(0)
		if m.IsUsingOverage {
			using = 1
		}
		_, err := tx.Exec(`INSERT INTO credential_meter (credential_id, seen_at, session_id, status, rate_limit_type,
    five_hour_utilization, five_hour_resets_at, seven_day_utilization, seven_day_resets_at, overage_status, is_using_overage)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (credential_id) DO UPDATE SET seen_at = excluded.seen_at, session_id = excluded.session_id,
    status = excluded.status, rate_limit_type = excluded.rate_limit_type,
    five_hour_utilization = excluded.five_hour_utilization, five_hour_resets_at = excluded.five_hour_resets_at,
    seven_day_utilization = excluded.seven_day_utilization, seven_day_resets_at = excluded.seven_day_resets_at,
    overage_status = excluded.overage_status, is_using_overage = excluded.is_using_overage`,
			Text(m.Credential), Text(m.SeenAt), Text(m.Session), Text(m.Status), Text(m.RateLimitType),
			Real(m.FiveHourUtilization), Int(m.FiveHourResetsAt), Real(m.SevenDayUtilization), Int(m.SevenDayResetsAt),
			Text(m.OverageStatus), Int(using))
		return err
	})
}

// Meter reads one credential's meter; nil, no error, before any rate-limit event for it.
func Meter(q Querier, credential string) (*MeterRow, error) {
	var out *MeterRow
	err := q.Query(`SELECT credential_id, seen_at, session_id, status, rate_limit_type, five_hour_utilization,
    five_hour_resets_at, seven_day_utilization, seven_day_resets_at, overage_status, is_using_overage
FROM credential_meter WHERE credential_id = ?`, []Value{Text(credential)},
		func(r *Row) error {
			out = &MeterRow{Credential: r.Text(0), SeenAt: r.Text(1), Session: r.Text(2), Status: r.Text(3),
				RateLimitType: r.Text(4), FiveHourUtilization: r.Float(5), FiveHourResetsAt: r.Int(6),
				SevenDayUtilization: r.Float(7), SevenDayResetsAt: r.Int(8), OverageStatus: r.Text(9),
				IsUsingOverage: r.Int(10) != 0}
			return nil
		})
	return out, err
}

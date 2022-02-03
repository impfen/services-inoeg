// Kiebitz - Privacy-Friendly Appointment Scheduling
// Copyright (C) 2021-2021 The Kiebitz Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version. Additional terms
// as defined in section 7 of the license (e.g. regarding attribution)
// are specified at https://kiebitz.eu/en/docs/open-source/additional-terms.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package meters

import (
	"encoding/binary"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/impfen/services-inoeg"
	"github.com/impfen/services-inoeg/databases"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Redis struct {
	*databases.Redis
}

func MakeRedisShards(settings interface{}) (services.Meter, error) {

	redisClient, err := databases.MakeRedisShards(settings)
	if err != nil {
		return nil, err
	}
	meter := &Redis{
		redisClient,
	}

	return meter, nil
}

func MakeRedis(settings interface{}) (services.Meter, error) {

	redisClient, err := databases.MakeRedis(settings)
	if err != nil {
		return nil, err
	}

	meter := &Redis{
		redisClient,
	}

	return meter, nil
}

var paramsRegex = regexp.MustCompile(`^([^\()]+)\((.*)\)$`)

func decodeData(value string) (map[string]string, string) {
	matches := paramsRegex.FindStringSubmatch(value)
	if matches == nil {
		return nil, value
	} else {
		m := make(map[string]string)
		parens := matches[2]
		eqns := strings.Split(parens, ",")
		for _, eqn := range eqns {
			kv := strings.SplitN(eqn, "=", 2)
			if len(kv) < 2 {
				return nil, matches[1]
			}
			m[kv[0]] = kv[1]
		}
		return m, matches[1]
	}
}

func encodeData(name string, data map[string]string) (string, error) {
	s := name + "("
	keys := make([]string, len(data))
	i := 0
	for k, v := range data {
		if strings.Contains(k, "=") || strings.Contains(v, "=") {
			return "", fmt.Errorf("keys/values should not contain '=' characters. Encountered one in string '%s' or '%s'", k, v)
		}
		keys[i] = k
		i++
	}
	sort.Sort(sort.StringSlice(keys))
	for i, k := range keys {
		v := data[k]
		s += k + "=" + v
		if i < len(keys)-1 {
			s += ","
		}
	}
	return s + ")", nil
}

func (r *Redis) getKey(name string, data map[string]string, tw services.TimeWindow) (string, error) {
	ed, err := encodeData(name, data)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s:%d:%d", strings.Replace(ed, ":", "::", -1), tw.Type, tw.From, tw.To), nil
}

func (r *Redis) getTimeId(t int64, twType string) int64 {
	tm := time.Unix(t/1e9, t%1e9).UTC()
	day := time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, time.UTC)
	switch twType {
	case "second":
		// we return the current minute
		return day.Add(time.Minute*time.Duration(tm.Minute()) + time.Hour*time.Duration(tm.Hour())).Unix()
	case "minute":
		// we return the current hour
		return day.Add(time.Hour * time.Duration(tm.Hour())).Unix()
	case "quarterHour":
		fallthrough
	case "hour":
		// we return the first day of the current week
		return day.AddDate(0, 0, -(int(day.Weekday())-1)%7).Unix()
	case "day":
		// we return the first day of the current quarter
		return time.Date(tm.Year(), tm.Month()-(tm.Month()-1)%3, 1, 0, 0, 0, 0, time.UTC).Unix()
	case "week":
		// we return the first day of the current year
		return time.Date(tm.Year(), 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	case "month":
		// we return the first day of the current year modulo 4
		return time.Date(tm.Year()-tm.Year()%4, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	}
	panic("unsupported time window")
}

func (r *Redis) increaseTimeId(tId, n int64, twType string) int64 {
	t := r.getTimeFromId(tId, twType)
	tm := time.Unix(t/1e9, t%1e9).UTC()
	switch twType {
	case "second":
		// we store one minute per interval
		return tm.Add(time.Duration(n) * time.Minute).Unix()
	case "minute":
		// we store an entire hour (60 minutes) per interval
		return tm.Add(time.Duration(n) * time.Hour).Unix()
	case "quarterHour":
		// we store an entire week
		return tm.AddDate(0, 0, 7*int(n)).Unix()
	case "hour":
		// we store an entire week (168 hours)
		return tm.AddDate(0, 0, 7*int(n)).Unix()
	case "day":
		// we store three entire months (90 days)
		return tm.AddDate(0, 3*int(n), 0).Unix()
	case "week":
		// we store an entire year (around 48 weeks)
		return tm.AddDate(int(n), 0, 0).Unix()
	case "month":
		// we store 4 years (48 months)
		return tm.AddDate(int(n*4), 0, 0).Unix()

	}
	panic("unsupported type")
}

// Returns the 'from'
func (r *Redis) getTimeFromId(tId int64, twType string) int64 {
	return tId * 1e9
}

func (r *Redis) getTimeWindowFromTimeId(timeId int64, twType string) services.TimeWindow {
	return services.TimeWindow{
		From: r.getTimeFromId(timeId, twType),
		To:   r.getTimeFromId(r.increaseTimeId(timeId, 1, twType), twType),
		Type: "custom",
	}
}

func (r *Redis) getFullId(id string, tw services.TimeWindow) string {
	// we group meter values for a given ID by day
	return r.getFullIdByTimeId(id, r.getTimeId(tw.From, tw.Type), tw.Type)
}

func (r *Redis) getFullIdByTimeId(id string, tId int64, twType string) string {
	// we group meter values for a given ID by day
	return fmt.Sprintf("%s:%s:%d", id, twType, tId)
}

// Adds the maximum value from a given UID to a given statistic
func (r *Redis) AddMax(id string, name string, uid string, data map[string]string, tw services.TimeWindow, value int64) error {

	key, err := r.getKey(name, data, tw)

	if err != nil {
		return err
	}

	fullId := r.getFullId(id, tw)
	fullKey := fmt.Sprintf("addMax:%s:%s", fullId, key)

	c := r.Client(fullKey)
	if oldValueBytes, err := c.HGet(r.Ctx, fullKey, uid).Result(); err != nil {
		if err != redis.Nil {
			return err
		} else {
			// the value doesn't exist yet, we store the current value
			bs := make([]byte, 8)
			binary.LittleEndian.PutUint64(bs, uint64(value))
			if err := c.HSet(r.Ctx, fullKey, uid, bs).Err(); err != nil {
				return err
			}

		}
	} else {
		var oldValue uint64
		if err == nil {
			oldValue = binary.LittleEndian.Uint64([]byte(oldValueBytes))
		}
		if int64(oldValue) >= value {
			// the old value is larger than the current value, we do nothing
			return nil
		} else {
			// the new value is larger than the old one, we store it as the
			// new maximum and add the difference
			bs := make([]byte, 8)
			binary.LittleEndian.PutUint64(bs, uint64(value))
			if err := c.HSet(r.Ctx, fullKey, uid, bs).Err(); err != nil {
				return err
			}
			// we subtract the old value
			value = value - int64(oldValue)
		}
	}

	// we set the expiration time of the control structure
	tId := r.getTimeId(tw.From, tw.Type)
	maxTw := r.getTimeWindowFromTimeId(r.increaseTimeId(tId, 10, tw.Type), tw.Type)

	if _, err := c.ExpireAt(r.Ctx, fullKey, time.Unix(maxTw.To/1e9, 0)).Result(); err != nil {
		return err
	}

	// twe add the difference to the maximum value to the statistic
	return r.Add(id, name, data, tw, value)

}

// Adds a value from a UID to the statistic, but only once
func (r *Redis) AddOnce(id string, name string, uid string, data map[string]string, tw services.TimeWindow, value int64) error {

	key, err := r.getKey(name, data, tw)

	if err != nil {
		return err
	}

	fullId := r.getFullId(id, tw)
	fullKey := fmt.Sprintf("addOnce:%s:%s", fullId, key)

	c := r.Client(fullKey)
	if ok, err := c.SIsMember(r.Ctx, fullKey, uid).Result(); err != nil {
		return err
	} else if ok {
		// the UID has already been counted
		return nil
	}

	tId := r.getTimeId(tw.From, tw.Type)
	maxTw := r.getTimeWindowFromTimeId(r.increaseTimeId(tId, 10, tw.Type), tw.Type)

	if _, err := c.ExpireAt(r.Ctx, fullKey, time.Unix(maxTw.To/1e9, 0)).Result(); err != nil {
		return err
	}

	if err := c.SAdd(r.Ctx, fullKey, uid).Err(); err != nil {
		return err
	}
	// the UID hasn't been counted yet, we add it
	return r.Add(id, name, data, tw, value)

}

func (r *Redis) Add(id string, name string, data map[string]string, tw services.TimeWindow, value int64) error {
	key, err := r.getKey(name, data, tw)
	if err != nil {
		return err
	}
	fullKey := r.getFullId(id, tw)

	c := r.Client(fullKey)

	res, err := c.HIncrBy(r.Ctx, fullKey, key, value).Result()
	if err != nil {
		return err
	}
	if res == value {
		// we set the expiration date of the key
		tId := r.getTimeId(tw.From, tw.Type)
		// we keep n intervals at most
		maxTw := r.getTimeWindowFromTimeId(r.increaseTimeId(tId, 10, tw.Type), tw.Type)
		_, err = c.ExpireAt(r.Ctx, fullKey, time.Unix(maxTw.To/1e9, 0)).Result()
	}
	return err
}

var pattern = regexp.MustCompile(`^((?:[^:]*(?:::)?)+):(\w+):(\d+):(\d+)$`)

func parseMetric(key string, value string) (*services.Metric, error) {
	i, err := strconv.ParseInt(value, 10, 64)
	matches := pattern.FindStringSubmatch(key)
	if matches == nil {
		return nil, fmt.Errorf("key did not match")
	}
	var tw services.TimeWindow
	tw.Type = matches[2]
	if tw.From, err = strconv.ParseInt(matches[3], 10, 64); err != nil {
		return nil, err
	}
	if tw.To, err = strconv.ParseInt(matches[4], 10, 64); err != nil {
		return nil, err
	}
	data, name := decodeData(strings.Replace(matches[1], "::", ":", -1))
	return &services.Metric{
		TimeWindow: tw,
		Name:       name,
		Value:      i,
		Data:       data,
	}, nil
}

type ByNameAndWindow []*services.Metric

func (b ByNameAndWindow) Len() int      { return len(b) }
func (b ByNameAndWindow) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByNameAndWindow) Less(i, j int) bool {
	return b[i].Name < b[j].Name || (b[i].Name == b[j].Name && b[i].TimeWindow.From > b[j].TimeWindow.From)
}

func (r *Redis) N(id string, to, n int64, name, twType string) ([]*services.Metric, error) {

	toTw := services.MakeTimeWindow(to, twType)
	fromTw := toTw.Copy()
	fromTw.IncreaseBy(-n + 1)

	maxTId := r.getTimeId(toTw.To, twType)
	tId := r.getTimeId(fromTw.From, twType)

	return r.GetByTimeIds(id, fromTw.From, toTw.To, tId, maxTId, name, twType)

}

func (r *Redis) Range(id string, from, to int64, name, twType string) ([]*services.Metric, error) {

	toTw := services.MakeTimeWindow(to, twType)
	fromTw := services.MakeTimeWindow(from, twType)

	maxTId := r.getTimeId(toTw.To, twType)
	tId := r.getTimeId(fromTw.From, twType)

	return r.GetByTimeIds(id, from, to, tId, maxTId, name, twType)

}

func (r *Redis) GetByTimeIds(id string, from, to int64, tId, maxTId int64, name, twType string) ([]*services.Metric, error) {
	metrics := make([]*services.Metric, 0)

	// we measure by how much a time ID increases (on average)
	incr := r.increaseTimeId(tId, 1, twType) - tId

	// we cancel if there are too many time IDs that we need to iterate over
	if tId > maxTId || (maxTId-tId)/incr > 30 {
		return nil, fmt.Errorf("too many time windows to iterate over")
	}

	for tId <= maxTId {
		fullKey := r.getFullIdByTimeId(id, tId, twType)
		c := r.Client(fullKey)

		result, err := c.HGetAll(r.Ctx, fullKey).Result()
		if err != nil {
			return nil, err
		}
		for k, v := range result {
			metric, err := parseMetric(k, v)
			if err != nil {
				continue
			}
			if metric.TimeWindow.To <= from || metric.TimeWindow.From >= to {
				continue
			}
			if name != "" && metric.Name != name {
				continue
			}
			metrics = append(metrics, metric)
		}
		tId = r.increaseTimeId(tId, 1, twType)
	}
	sort.Sort(ByNameAndWindow(metrics))
	return metrics, nil
}

func (r *Redis) Get(id string, name string, data map[string]string, tw services.TimeWindow) (*services.Metric, error) {
	key, err := r.getKey(name, data, tw)
	if err != nil {
		return nil, err
	}
	fullKey := r.getFullId(id, tw)
	c := r.Client(fullKey)
	res, err := c.HGet(r.Ctx, fullKey, key).Int64()
	if err != nil {
		if err == redis.Nil {
			res = 0
		} else {
			return nil, err
		}
	}
	return &services.Metric{
		Value:      res,
		TimeWindow: tw,
		Name:       name,
	}, nil
}

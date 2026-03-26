// Copyright 2021 E99p1ant. All rights reserved.

package dbutil

import (
	"time"
)

func Now() time.Time {
	// 统一截断到微秒，减少不同数据库时间精度带来的比较误差。
	return time.Now().Truncate(time.Microsecond)
}

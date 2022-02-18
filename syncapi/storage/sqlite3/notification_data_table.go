// Copyright 2020 The Matrix.org Foundation C.I.C.
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

package sqlite3

import (
	"database/sql"

	"github.com/matrix-org/dendrite/syncapi/storage/shared"
	"github.com/matrix-org/dendrite/syncapi/storage/tables"
)

const notificationDataSchema = `
CREATE TABLE IF NOT EXISTS syncapi_notification_data (
	id INTEGER PRIMARY KEY,
	user_id TEXT NOT NULL,
	room_id TEXT NOT NULL,
	notification_count BIGINT NOT NULL DEFAULT 0,
	highlight_count BIGINT NOT NULL DEFAULT 0,
	CONSTRAINT syncapi_notifications_unique UNIQUE (user_id, room_id)
);`

func NewSqliteNotificationDataTable(db *sql.DB) (tables.NotificationData, error) {
	_, err := db.Exec(notificationDataSchema)
	if err != nil {
		return nil, err
	}
	return shared.NewNotificationDataTable(db)
}
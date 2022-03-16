// Copyright 2021 The Matrix.org Foundation C.I.C.
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

package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/matrix-org/dendrite/internal/sqlutil"
	"github.com/matrix-org/dendrite/syncapi/storage/tables"
	"github.com/matrix-org/dendrite/syncapi/types"
	"github.com/matrix-org/gomatrixserverlib"
)

// The memberships table is designed to track the last time that
// the user was a given state. This allows us to find out the
// most recent time that a user was invited to, joined or left
// a room, either by choice or otherwise. This is important for
// building history visibility.

const membershipsSchema = `
CREATE TABLE IF NOT EXISTS syncapi_memberships (
    -- The 'room_id' key for the state event.
    room_id TEXT NOT NULL,
    -- The state event ID
	user_id TEXT NOT NULL,
	-- The status of the membership
	membership TEXT NOT NULL,
	-- The event ID that last changed the membership
	event_id TEXT NOT NULL,
	-- The stream position of the change
	stream_pos BIGINT NOT NULL,
	-- The topological position of the change in the room
	topological_pos BIGINT NOT NULL,
	-- Unique index
	CONSTRAINT syncapi_memberships_unique UNIQUE (room_id, user_id, membership)
);
`

const upsertMembershipSQL = "" +
	"INSERT INTO syncapi_memberships (room_id, user_id, membership, event_id, stream_pos, topological_pos)" +
	" VALUES ($1, $2, $3, $4, $5, $6)" +
	" ON CONFLICT ON CONSTRAINT syncapi_memberships_unique" +
	" DO UPDATE SET event_id = $4, stream_pos = $5, topological_pos = $6"

const selectMembershipSQL = "" +
	"SELECT event_id, stream_pos, topological_pos FROM syncapi_memberships" +
	" WHERE room_id = $1 AND user_id = $2 AND membership = ANY($3)" +
	" ORDER BY stream_pos DESC" +
	" LIMIT 1"

const purgeMembershipForRoomSQL = "DELETE FROM syncapi_memberships WHERE room_id = $1"

type membershipsStatements struct {
	upsertMembershipStmt       *sql.Stmt
	selectMembershipStmt       *sql.Stmt
	purgeMembershipForRoomStmt *sql.Stmt
}

func NewPostgresMembershipsTable(db *sql.DB) (tables.Memberships, error) {
	s := &membershipsStatements{}
	_, err := db.Exec(membershipsSchema)
	if err != nil {
		return nil, err
	}
	if s.upsertMembershipStmt, err = db.Prepare(upsertMembershipSQL); err != nil {
		return nil, err
	}
	if s.selectMembershipStmt, err = db.Prepare(selectMembershipSQL); err != nil {
		return nil, err
	}
	if s.purgeMembershipForRoomStmt, err = db.Prepare(purgeMembershipForRoomSQL); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *membershipsStatements) UpsertMembership(
	ctx context.Context, txn *sql.Tx, event *gomatrixserverlib.HeaderedEvent,
	streamPos, topologicalPos types.StreamPosition,
) error {
	membership, err := event.Membership()
	if err != nil {
		return fmt.Errorf("event.Membership: %w", err)
	}
	_, err = sqlutil.TxStmt(txn, s.upsertMembershipStmt).ExecContext(
		ctx,
		event.RoomID(),
		*event.StateKey(),
		membership,
		event.EventID(),
		streamPos,
		topologicalPos,
	)
	return err
}

func (s *membershipsStatements) SelectMembership(
	ctx context.Context, txn *sql.Tx, roomID, userID, memberships []string,
) (eventID string, streamPos, topologyPos types.StreamPosition, err error) {
	stmt := sqlutil.TxStmt(txn, s.selectMembershipStmt)
	err = stmt.QueryRowContext(ctx, roomID, userID, memberships).Scan(&eventID, &streamPos, &topologyPos)
	return
}

func (s *membershipsStatements) PurgeRoom(
	ctx context.Context, txn *sql.Tx, roomID string,
) error {
	stmt := sqlutil.TxStmt(txn, s.purgeMembershipForRoomStmt)
	_, err := stmt.ExecContext(ctx, roomID)
	return err
}

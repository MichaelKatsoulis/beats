// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mongodb

// Represent a mongodb message being parsed
import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type mongodbMessage struct {
	ts time.Time

	tcpTuple     common.TCPTuple
	cmdlineTuple *common.ProcessTuple
	direction    uint8

	isResponse bool

	// Standard message header fields from mongodb wire protocol
	// see http://docs.mongodb.org/meta-driver/latest/legacy/mongodb-wire-protocol/#standard-message-header
	messageLength int32
	requestID     int32
	responseTo    int32
	opCode        opCode

	// decoded flagBits
	checkSumPresent bool
	moreToCome      bool
	exhaustAllowed  bool

	// deduced from content. Either an operation from the original wire protocol or the name of a command (passed through a query)
	// List of commands: http://docs.mongodb.org/manual/reference/command/
	// List of original protocol operations: http://docs.mongodb.org/meta-driver/latest/legacy/mongodb-wire-protocol/#request-opcodes
	method    string
	error     string
	resource  string
	documents []interface{}
	params    map[string]interface{}

	// Other fields vary very much depending on operation type
	// lets just put them in a map
	event mapstr.M
}

func (m *mongodbMessage) SetFlagBits(flagBits int32) {
	m.checkSumPresent = flagBits&0x1 != 0    // 0 bit
	m.moreToCome = flagBits&0x2 != 0         // 1 bit
	m.exhaustAllowed = flagBits&0x10000 != 0 // 16 bit
}

// Represent a stream being parsed that contains a mongodb message
type stream struct {
	tcptuple *common.TCPTuple

	data    []byte
	message *mongodbMessage
}

// Parser moves to next message in stream
func (st *stream) PrepareForNewMessage() {
	st.data = st.data[st.message.messageLength:]
	st.message = nil
}

// The private data of a parser instance
// is composed of 2 potentially active streams: incoming, outgoing
type mongodbConnectionData struct {
	streams [2]*stream
}

// Represent a full mongodb transaction (request/reply)
// These transactions are the end product of this parser
type transaction struct {
	cmdline  *common.ProcessTuple
	src      common.Endpoint
	dst      common.Endpoint
	ts       time.Time
	endTime  time.Time
	bytesOut int
	bytesIn  int

	mongodb mapstr.M

	event            mapstr.M
	method           string
	resource         string
	error            string
	params           map[string]interface{}
	requestDocuments []interface{}
	documents        []interface{}
}

type msgKind byte

const (
	msgKindBody             msgKind = 0
	msgKindDocumentSequence msgKind = 1
	msgKindInternal         msgKind = 2
)

type opCode int32

const (
	opReply      opCode = 1
	opMsgLegacy  opCode = 1000
	opUpdate     opCode = 2001
	opInsert     opCode = 2002
	opReserved   opCode = 2003
	opQuery      opCode = 2004
	opGetMore    opCode = 2005
	opDelete     opCode = 2006
	opKillCursor opCode = 2007
	opMsg        opCode = 2013
)

// List of valid mongodb wire protocol operation codes
// see http://docs.mongodb.org/meta-driver/latest/legacy/mongodb-wire-protocol/#request-opcodes
var opCodeNames = map[opCode]string{
	1:    "OP_REPLY",
	1000: "OP_MSG",
	2001: "OP_UPDATE",
	2002: "OP_INSERT",
	2003: "RESERVED",
	2004: "OP_QUERY",
	2005: "OP_GET_MORE",
	2006: "OP_DELETE",
	2007: "OP_KILL_CURSORS",
	2013: "OP_MSG",
}

func validOpcode(o opCode) bool {
	_, found := opCodeNames[o]
	return found
}

func (o opCode) String() string {
	if name, found := opCodeNames[o]; found {
		return name
	}
	return fmt.Sprintf("(value=%d)", int32(o))
}

func awaitsReply(msg *mongodbMessage) bool {
	opCode := msg.opCode
	// The request of opMsg type doesn't get response if moreToCome is set
	// From documentation: https://www.mongodb.com/docs/manual/reference/mongodb-wire-protocol
	// "Requests with the moreToCome bit set will not receive a reply"
	if !msg.isResponse && opCode == opMsg && !msg.moreToCome {
		return true
	}
	return opCode == opQuery || opCode == opGetMore
}

// List of mongodb user commands (send through a query of the legacy protocol)
// see http://docs.mongodb.org/manual/reference/command/
//
// This list was obtained by calling db.listCommands() and some grepping.
// They are compared cased insensitive
var databaseCommands = []string{
	"getLastError",
	"connPoolSync",
	"top",
	"dropIndexes",
	"explain",
	"grantRolesToRole",
	"dropRole",
	"dropAllRolesFromDatabase",
	"listCommands",
	"replSetReconfig",
	"replSetFresh",
	"writebacklisten",
	"setParameter",
	"update",
	"replSetGetStatus",
	"find",
	"resync",
	"appendOplogNote",
	"revokeRolesFromRole",
	"compact",
	"createUser",
	"replSetElect",
	"getPrevError",
	"serverStatus",
	"getShardVersion",
	"updateRole",
	"replSetFreeze",
	"getCmdLineOpts",
	"applyOps",
	"count",
	"aggregate",
	"copydbsaslstart",
	"distinct",
	"repairDatabase",
	"profile",
	"replSetStepDown",
	"findAndModify",
	"_transferMods",
	"filemd5",
	"forceerror",
	"getnonce",
	"saslContinue",
	"clone",
	"saslStart",
	"_getUserCacheGeneration",
	"_recvChunkCommit",
	"whatsmyuri",
	"repairCursor",
	"validate",
	"dbHash",
	"planCacheListFilters",
	"touch",
	"mergeChunks",
	"cursorInfo",
	"_recvChunkStart",
	"unsetSharding",
	"revokePrivilegesFromRole",
	"logout",
	"group",
	"shardConnPoolStats",
	"listDatabases",
	"buildInfo",
	"availableQueryOptions",
	"_isSelf",
	"splitVector",
	"geoSearch",
	"dbStats",
	"connectionStatus",
	"currentOpCtx",
	"copydb",
	"insert",
	"reIndex",
	"moveChunk",
	"cleanupOrphaned",
	"driverOIDTest",
	"isMaster",
	"getParameter",
	"replSetHeartbeat",
	"ping",
	"listIndexes",
	"dropUser",
	"dropDatabase",
	"dataSize",
	"convertToCapped",
	"planCacheSetFilter",
	"usersInfo",
	"grantPrivilegesToRole",
	"handshake",
	"_mergeAuthzCollections",
	"mapreduce.shardedfinish",
	"_recvChunkAbort",
	"authSchemaUpgrade",
	"replSetGetConfig",
	"replSetSyncFrom",
	"collStats",
	"replSetMaintenance",
	"createRole",
	"copydbgetnonce",
	"cloneCollectionAsCapped",
	"_migrateClone",
	"parallelCollectionScan",
	"connPoolStats",
	"revokeRolesFromUser",
	"authenticate",
	"create",
	"shutdown",
	"invalidateUserCache",
	"shardingState",
	"renameCollection",
	"replSetGetRBID",
	"splitChunk",
	"createIndexes",
	"updateUser",
	"cloneCollection",
	"logRotate",
	"planCacheListPlans",
	"medianKey",
	"hostInfo",
	"geoNear",
	"fsync",
	"checkShardingIndex",
	"getShardMap",
	"planCacheClear",
	"listCollections",
	"collMod",
	"_recvChunkStatus",
	"planCacheListQueryShapes",
	"delete",
	"planCacheClearFilters",
	"mapReduce",
	"rolesInfo",
	"eval",
	"drop",
	"grantRolesToUser",
	"resetError",
	"getLog",
	"dropAllUsersFromDatabase",
	"diagLogging",
	"replSetUpdatePosition",
	"setShardVersion",
	"replSetInitiate",
}

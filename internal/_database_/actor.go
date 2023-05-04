package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/google/uuid"
	"github.com/looplab/fsm"
	"go.etcd.io/bbolt"
)

type dbActor struct {
	behavior actor.Behavior
	db       *bbolt.DB
	fm       *fsm.FSM
	pathDB   string
	// mux      sync.Mutex
	ctx     actor.Context
	rootctx *actor.RootContext
	pid     *actor.PID
}

type DB interface {
	PID() *actor.PID
	RootContext() *actor.RootContext
}

func (db *dbActor) PID() *actor.PID {
	return db.pid
}
func (db *dbActor) RootContext() *actor.RootContext {
	return db.rootctx
}

func Open(ctx *actor.RootContext, pathdb string) (DB, error) {

	instance := &dbActor{}
	instance.pathDB = pathdb

	instance.behavior = make(actor.Behavior, 0)
	instance.behavior.Become(instance.CloseState)
	instance.fm = instance.initFSM()

	props := actor.PropsFromFunc(instance.Receive)

	if ctx == nil {
		ctx = actor.NewActorSystem().Root
	}
	instance.rootctx = ctx

	pid, err := ctx.SpawnNamed(props, fmt.Sprintf("db-actor-%d", time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	instance.pid = pid

	time.Sleep(1 * time.Second)
	return instance, nil
}

func (a *dbActor) Receive(ctx actor.Context) {

	a.ctx = ctx
	a.behavior.Receive(ctx)
}

func (a *dbActor) CloseState(ctx actor.Context) {
	logs.LogBuild.Printf("Message arrive in datab (CloseState): %s, %T, %s", ctx.Message(), ctx.Message(), ctx.Sender())
	switch ctx.Message().(type) {
	case *actor.Started:
		a.fm.Event(eOpenCmd)
	case *MsgErrorDB:
		a.fm.Event(eError)
	case *MsgOpenDB:
		a.fm.Event(eOpenCmd)
		if ctx.Sender() != nil {
			ctx.Respond(&MsgOpenDB{})
		}
	case *MsgOpenedDB:
		a.behavior.Become(a.WaitState)
		a.fm.Event(eOpened)
	}
}

func (a *dbActor) WaitState(ctx actor.Context) {
	logs.LogBuild.Printf("Message arrive in datab (WaitState): %T, %s",
		ctx.Message(), ctx.Sender())
	// logs.LogBuild.Printf("Message arrive in datab (WaitState): %s, %T, %s", ctx.Message(), ctx.Message(), ctx.Sender())
	switch msg := ctx.Message().(type) {
	case *MsgFlushFilesystem:
		if a.db != nil {
			if err := a.db.Sync(); err != nil {
				logs.LogError.Printf("error file db: %s", err)
			}
		}
	case *actor.Stopping:
		a.db.Close()
		a.fm.Event(eClosed)
	case *MsgOpenDB:
		if ctx.Sender() != nil {
			ctx.Respond(&MsgOpenDB{})
		}
	case *MsgErrorDB:
		a.fm.Event(eError)
	case *MsgInsertData:

		if err := func() error {
			var id string
			if len(msg.ID) <= 0 {
				if uid, err := uuid.NewUUID(); err != nil {
					// ctx.Respond(&MsgNoAckPersistData{Error: err.Error()})
					return err
				} else {
					id = uid.String()
				}
			} else {
				id = msg.ID
			}

			if err := a.db.Update(PersistData(id, msg.Data, false, msg.Buckets...)); err != nil {
				// ctx.Respond(&MsgNoAckPersistData{Error: err.Error()})
				return err
			}
			if ctx.Sender() != nil {
				ctx.Respond(&MsgAckPersistData{
					ID:      id,
					Bucktes: msg.Buckets,
				})
			}
			// logs.LogBuild.Printf("STEP 6_00000: %s", ctx.Sender())
			return nil
		}(); err != nil {
			logs.LogError.Println(err)
			if ctx.Sender() != nil {
				ctx.Respond(&MsgNoAckPersistData{Error: err.Error()})
			}
			switch {
			case errors.Is(err, bbolt.ErrDatabaseNotOpen):
				a.fm.Event(eError)
			case errors.Is(err, bbolt.ErrDatabaseOpen):
				a.fm.Event(eError)
			}
		}

	case *MsgUpdateData:
		if err := func(ctx actor.Context) error {
			var id string
			if len(msg.ID) <= 0 {
				if uid, err := uuid.NewUUID(); err != nil {
					return err
				} else {
					id = uid.String()
				}
			} else {
				id = msg.ID
			}

			// logs.LogBuild.Printf("STEP 6_0000: %s", ctx.Sender())
			if err := a.db.Update(PersistData(id, msg.Data, true, msg.Buckets...)); err != nil {
				return err
			}
			if ctx.Sender() != nil {
				ctx.Respond(&MsgAckPersistData{
					ID:      id,
					Bucktes: msg.Buckets,
				})
			}
			// logs.LogBuild.Printf("STEP 6_1111: %s", ctx.Sender())
			//TODO:
			//time.Sleep(1 * time.Second)
			return nil
		}(ctx); err != nil {
			logs.LogError.Println(err)
			if ctx.Sender() != nil {
				ctx.Respond(&MsgNoAckPersistData{Error: err.Error()})
			}
			switch {
			case errors.Is(err, bbolt.ErrDatabaseNotOpen):
				a.fm.Event(eError)
			case errors.Is(err, bbolt.ErrDatabaseOpen):
				a.fm.Event(eError)
			case errors.Is(err, bbolt.ErrDatabaseReadOnly):
				a.fm.Event(eError)
			case errors.Is(err, bbolt.ErrTxNotWritable):
				a.fm.Event(eError)
			}
		}

	case *MsgDeleteData:
		if err := func() error {
			id := msg.ID

			if err := a.db.Update(RemoveData(id, msg.Buckets...)); err != nil {
				return err
			}
			if ctx.Sender() != nil {
				ctx.Respond(&MsgAckDeleteData{
					ID:      id,
					Buckets: msg.Buckets,
				})
			}
			return nil
		}(); err != nil {
			if ctx.Sender() != nil {
				ctx.Respond(&MsgNoAckDeleteData{Error: err.Error()})
			}
			logs.LogError.Println(err)
			switch {
			case errors.Is(err, bbolt.ErrDatabaseNotOpen):
				a.fm.Event(eError)
			case errors.Is(err, bbolt.ErrDatabaseOpen):
				a.fm.Event(eError)
			}
		}

	case *MsgGetData:
		if err := func() error {
			id := msg.ID
			data := make([]byte, 0)
			callabck := func(v []byte) {
				data = append(data, v...)
			}
			if err := a.db.View(GetData(callabck, id, msg.Buckets...)); err != nil {
				return err
			}
			if ctx.Sender() != nil {
				ctx.Respond(&MsgAckGetData{Data: data})
			}
			return nil
		}(); err != nil {
			logs.LogError.Println(err)
			if ctx.Sender() != nil {
				ctx.Respond(&MsgNoAckGetData{Error: err.Error()})
			}
			switch {
			case errors.Is(err, bbolt.ErrDatabaseNotOpen):
				a.fm.Event(eError)
			case errors.Is(err, bbolt.ErrDatabaseOpen):
				a.fm.Event(eError)
			}
		}
	case *MsgQueryData:
		if err := func() error {
			prefix := []byte(msg.PrefixID)

			contxt, cancel := context.WithCancel(context.TODO())

			callback := func(v *QueryType) {
				// log.Printf("data in channel: %s, %s", v.ID, pid)
				data := make([]byte, len(v.Data))
				copy(data, v.Data)
				if err := ctx.RequestFuture(ctx.Sender(), &MsgQueryResponse{
					Data:    data,
					ID:      v.ID,
					Buckets: msg.Buckets,
				}, 3*time.Second).Wait(); err != nil {
					logs.LogWarn.Printf("error send datadb: %s, %s", err, ctx.Sender())
					cancel()
					return
				}
			}
			if err := a.db.View(QueryData(contxt, callback, prefix, msg.Reverse, msg.Buckets...)); err != nil {
				return err
			}
			if ctx.Sender() != nil {
				ctx.Respond(&MsgAckGetData{Data: nil})
			}
			return nil
		}(); err != nil {
			logs.LogError.Println(err)
			if ctx.Sender() != nil {
				ctx.Respond(&MsgNoAckGetData{Error: err.Error()})
			}
			switch {
			case errors.Is(err, bbolt.ErrDatabaseNotOpen):
				a.fm.Event(eError)
			case errors.Is(err, bbolt.ErrDatabaseOpen):
				a.fm.Event(eError)
			}
		}
	case *MsgList:
		if err := func() error {
			data := make(map[string][]byte, 0)
			callback := func(result map[string][]byte) {
				for k, v := range result {
					copydata := make([]byte, len(v))
					copy(copydata, v)
					data[k] = copydata
				}
			}
			if err := a.db.View(List(callback)); err != nil {
				return err
			}
			if ctx.Sender() != nil {
				ctx.Respond(&MsgAckList{Data: data})
			}
			return nil
		}(); err != nil {
			logs.LogError.Println(err)
			if ctx.Sender() != nil {
				ctx.Respond(&MsgNoAckList{Error: err.Error()})
			}
			switch {
			case errors.Is(err, bbolt.ErrDatabaseNotOpen):
				a.fm.Event(eError)
			case errors.Is(err, bbolt.ErrDatabaseOpen):
				a.fm.Event(eError)
			}
		}
	case *MsgListKeys:
		if err := func() error {
			data := make([][]byte, 0)
			callback := func(list [][]byte) {
				data = append(data, list...)
			}
			if err := a.db.View(ListKeys(callback, msg.Buckets...)); err != nil {

				return err
			}
			if ctx.Sender() != nil {
				ctx.Respond(&MsgAckListKeys{Data: data})
			}
			return nil
		}(); err != nil {
			logs.LogError.Println(err)
			if ctx.Sender() != nil {
				ctx.Respond(&MsgNoAckListKyes{Error: err.Error()})
			}
			switch {
			case errors.Is(err, bbolt.ErrDatabaseNotOpen):
				a.fm.Event(eError)
			case errors.Is(err, bbolt.ErrDatabaseOpen):
				a.fm.Event(eError)
			}
		}
	case *MsgCloseDB:
		a.db.Close()
		a.fm.Event(eClosed)
	}
}

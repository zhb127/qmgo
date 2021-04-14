/*
 Copyright 2020 The Qmgo Authors.
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

package qmgo

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zhb127/qmgo/operator"
	"github.com/zhb127/qmgo/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserHook struct {
	Name string `bson:"name"`
	Age  int    `bson:"age"`

	beforeCount int
	afterCount  int
}

func (u *UserHook) BeforeUpsert() error {
	u.beforeCount++
	return nil
}

func (u *UserHook) AfterUpsert() error {
	u.afterCount++
	return nil
}

func (u *UserHook) BeforeUpdate() error {
	u.beforeCount++
	return nil
}

func (u *UserHook) AfterUpdate() error {
	u.afterCount++
	return nil
}

func (u *UserHook) BeforeInsert() error {
	if u.Name == "Lucas" || u.Name == "xm" {
		u.Age = 17
	}
	return nil
}

var afterInsertCount = 0

func (u *UserHook) AfterInsert() error {
	afterInsertCount++
	return nil
}

type MyQueryHook struct {
	beforeCount int
	afterCount  int
}

func (q *MyQueryHook) BeforeQuery() error {
	q.beforeCount++
	return nil
}

func (q *MyQueryHook) AfterQuery() error {
	q.afterCount++
	return nil
}

func TestInsertHook(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	ctx := context.Background()
	defer cli.Close(ctx)
	defer cli.DropCollection(ctx)

	afterInsertCount = 0
	u := &UserHook{Name: "Lucas", Age: 7}
	_, err := cli.InsertOne(context.Background(), u, options.InsertOneOptions{
		InsertHook: u,
	})
	ast.NoError(err)

	uc := bson.M{"name": "Lucas"}
	ur := &UserHook{}
	uk := &MyQueryHook{}
	err = cli.Find(ctx, uc, options.FindOptions{
		QueryHook: uk,
	}).One(ur)
	ast.NoError(err)

	ast.Equal(17, ur.Age)

	ast.Equal(1, afterInsertCount)
	ast.Equal(1, uk.beforeCount)
	ast.Equal(1, uk.afterCount)
}

func TestInsertManyHook(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	ctx := context.Background()
	defer cli.Close(ctx)
	defer cli.DropCollection(ctx)

	afterInsertCount = 0
	u1 := &UserHook{Name: "Lucas", Age: 7}
	u2 := &UserHook{Name: "xm", Age: 7}
	us := []*UserHook{u1, u2}
	_, err := cli.InsertMany(ctx, us, options.InsertManyOptions{
		InsertHook: us,
	})
	ast.NoError(err)

	uc := bson.M{"name": "Lucas"}
	ur := []UserHook{}
	qh := &MyQueryHook{}
	err = cli.Find(ctx, uc, options.FindOptions{
		QueryHook: qh,
	}).All(&ur)
	ast.NoError(err)

	ast.Equal(17, ur[0].Age)

	ast.Equal(2, afterInsertCount)
	ast.Equal(1, qh.afterCount)
	ast.Equal(1, qh.beforeCount)

}

type MyUpdateHook struct {
	beforeUpdateCount int
	afterUpdateCount  int
}

func (u *MyUpdateHook) BeforeUpdate() error {
	u.beforeUpdateCount++
	return nil
}

func (u *MyUpdateHook) AfterUpdate() error {
	u.afterUpdateCount++
	return nil
}

func TestUpdateHook(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	ctx := context.Background()
	defer cli.Close(ctx)
	defer cli.DropCollection(ctx)

	u := UserHook{Name: "Lucas", Age: 7}
	uh := &MyUpdateHook{}
	res, err := cli.InsertOne(context.Background(), u)
	ast.NoError(err)

	err = cli.UpdateOne(ctx, bson.M{"name": "Lucas"}, bson.M{operator.Set: bson.M{"age": 27}}, options.UpdateOptions{
		UpdateHook: uh,
	})
	ast.NoError(err)
	ast.Equal(1, uh.beforeUpdateCount)
	ast.Equal(1, uh.afterUpdateCount)

	err = cli.UpdateId(ctx, res.InsertedID, bson.M{operator.Set: bson.M{"age": 27}}, options.UpdateOptions{
		UpdateHook: uh,
	})
	ast.NoError(err)
	ast.Equal(2, uh.beforeUpdateCount)
	ast.Equal(2, uh.afterUpdateCount)

	err = cli.ReplaceOne(ctx, bson.M{"name": "Lucas"}, &u)
	ast.NoError(err)
	ast.Equal(1, u.beforeCount)
	ast.Equal(1, u.afterCount)

	err = cli.ReplaceOne(ctx, bson.M{"name": "Lucas"}, &u, options.ReplaceOptions{
		UpdateHook: &u,
	})
	ast.NoError(err)
	ast.Equal(2, u.beforeCount)
	ast.Equal(2, u.afterCount)

	cli.UpdateAll(ctx, bson.M{"name": "Lucas"}, bson.M{operator.Set: bson.M{"age": 27}}, options.UpdateOptions{
		UpdateHook: uh,
	})
	ast.NoError(err)
	ast.Equal(3, uh.beforeUpdateCount)
	ast.Equal(3, uh.afterUpdateCount)
}

type MyRemoveHook struct {
	beforeCount int
	afterCount  int
}

func (m *MyRemoveHook) BeforeRemove() error {
	m.beforeCount++
	return nil
}

func (m *MyRemoveHook) AfterRemove() error {
	m.afterCount++
	return nil
}

func TestRemoveHook(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	ctx := context.Background()
	defer cli.Close(ctx)
	defer cli.DropCollection(ctx)

	u := []*UserHook{&UserHook{Name: "Lucas", Age: 7}, &UserHook{Name: "xm", Age: 7},
		&UserHook{Name: "wxy", Age: 7}, &UserHook{Name: "zp", Age: 7}}
	rlt, err := cli.InsertMany(context.Background(), u)
	ast.NoError(err)

	rh := &MyRemoveHook{}
	err = cli.RemoveId(ctx, rlt.InsertedIDs[0].(primitive.ObjectID), options.RemoveOptions{
		RemoveHook: rh,
	})
	ast.NoError(err)
	ast.Equal(1, rh.afterCount)
	ast.Equal(1, rh.beforeCount)

	rh = &MyRemoveHook{}
	err = cli.Remove(ctx, bson.M{"age": 17}, options.RemoveOptions{
		RemoveHook: rh,
	})
	ast.NoError(err)
	ast.Equal(1, rh.afterCount)
	ast.Equal(1, rh.beforeCount)

	rh = &MyRemoveHook{}
	_, err = cli.RemoveAll(ctx, bson.M{"age": "7"}, options.RemoveOptions{
		RemoveHook: rh,
	})
	ast.NoError(err)
	ast.Equal(1, rh.afterCount)
	ast.Equal(1, rh.beforeCount)

}

func TestUpsertHook(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	ctx := context.Background()
	defer cli.Close(ctx)
	defer cli.DropCollection(ctx)

	afterInsertCount = 0
	u := &UserHook{Name: "Lucas", Age: 7}
	res, err := cli.InsertOne(context.Background(), u, options.InsertOneOptions{
		InsertHook: u,
	})
	ast.NoError(err)

	u.Age = 17
	_, err = cli.Upsert(context.Background(), bson.M{"name": "Lucas"}, u)
	ast.NoError(err)

	ast.Equal(1, u.beforeCount)
	ast.Equal(1, u.afterCount)

	_, err = cli.UpsertId(context.Background(), res.InsertedID, u)
	ast.NoError(err)

	ast.Equal(2, u.beforeCount)
	ast.Equal(2, u.afterCount)
}

type MyErrorHook struct {
	beforeQCount  int
	afterQCount   int
	beforeRCount  int
	afterRCount   int
	beforeUCount  int
	afterUCount   int
	beforeICount  int
	afterICount   int
	beforeUsCount int
	afterUsCount  int
}

func (m *MyErrorHook) BeforeUpsert() error {
	if m.beforeUsCount == 0 {
		m.beforeUsCount++
		return errors.New("error")
	}
	m.beforeUsCount++
	return nil
}

func (m *MyErrorHook) AfterUpsert() error {
	if m.afterUsCount == 0 {
		m.afterUsCount++
		return errors.New("error")
	}
	m.afterUsCount++
	return nil
}

func (m *MyErrorHook) BeforeRemove() error {
	if m.beforeRCount == 0 {
		m.beforeRCount++
		return errors.New("error")
	}
	m.beforeRCount++
	return nil
}

func (m *MyErrorHook) AfterRemove() error {
	m.afterRCount++
	return errors.New("error")
}

func (m *MyErrorHook) BeforeQuery() error {
	if m.beforeQCount == 0 {
		m.beforeQCount++
		return errors.New("error")
	}
	m.beforeQCount++

	return nil
}

func (m *MyErrorHook) AfterQuery() error {
	m.afterQCount++
	return errors.New("error")
}

func (m *MyErrorHook) BeforeInsert() error {
	if m.beforeICount == 0 {
		m.beforeICount++
		return errors.New("error")
	}
	m.beforeICount++

	return nil
}

func (m *MyErrorHook) AfterInsert() error {
	m.afterICount++
	return errors.New("error")
}

func (m *MyErrorHook) BeforeUpdate() error {
	if m.beforeUCount == 0 {
		m.beforeUCount++
		return errors.New("error")
	}
	m.beforeUCount++
	return nil
}

func (m *MyErrorHook) AfterUpdate() error {
	m.afterUCount++
	return errors.New("error")
}

func TestHookErr(t *testing.T) {
	ast := require.New(t)
	cli := initClient("test")
	ctx := context.Background()
	defer cli.Close(ctx)
	defer cli.DropCollection(ctx)

	u := &UserHook{Name: "Lucas", Age: 7}
	myHook := &MyErrorHook{}
	_, err := cli.InsertOne(context.Background(), u, options.InsertOneOptions{
		InsertHook: myHook,
	})
	ast.Error(err)
	ast.Equal(1, myHook.beforeICount)
	ast.Equal(0, myHook.afterICount)

	_, err = cli.InsertOne(context.Background(), u, options.InsertOneOptions{
		InsertHook: myHook,
	})
	ast.Error(err)
	ast.Equal(2, myHook.beforeICount)
	ast.Equal(1, myHook.afterICount)

	err = cli.UpdateOne(ctx, bson.M{"name": "Lucas"}, bson.M{operator.Set: bson.M{"age": 27}}, options.UpdateOptions{
		UpdateHook: myHook,
	})
	ast.Error(err)
	ast.Equal(1, myHook.beforeUCount)
	ast.Equal(0, myHook.afterUCount)

	err = cli.UpdateOne(ctx, bson.M{"name": "Lucas"}, bson.M{operator.Set: bson.M{"age": 27}}, options.UpdateOptions{
		UpdateHook: myHook,
	})
	ast.Error(err)
	ast.Equal(2, myHook.beforeUCount)
	ast.Equal(1, myHook.afterUCount)

	err = cli.UpdateId(ctx, bson.M{"name": "Lucas"}, bson.M{operator.Set: bson.M{"age": 27}}, options.UpdateOptions{
		UpdateHook: myHook,
	})
	ast.Error(err)

	err = cli.Find(ctx, bson.M{"age": 27}, options.FindOptions{
		QueryHook: myHook,
	}).One(u)
	ast.Error(err)
	ast.Equal(1, myHook.beforeQCount)
	ast.Equal(0, myHook.afterQCount)

	err = cli.Find(ctx, bson.M{"age": 27}, options.FindOptions{
		QueryHook: myHook,
	}).One(u)
	ast.Error(err)
	ast.Equal(2, myHook.beforeQCount)
	ast.Equal(1, myHook.afterQCount)

	err = cli.Remove(ctx, bson.M{"age": 27}, options.RemoveOptions{
		RemoveHook: myHook,
	})
	ast.Error(err)
	ast.Equal(1, myHook.beforeRCount)
	ast.Equal(0, myHook.afterRCount)

	err = cli.Remove(ctx, bson.M{"age": 27}, options.RemoveOptions{
		RemoveHook: myHook,
	})
	ast.Error(err)
	ast.Equal(2, myHook.beforeRCount)
	ast.Equal(1, myHook.afterRCount)

	_, err = cli.Upsert(ctx, bson.M{"name": "Lucas"}, u, options.UpsertOptions{
		UpsertHook: myHook,
	})
	ast.Error(err)
	ast.Equal(1, myHook.beforeUsCount)
	ast.Equal(0, myHook.afterUsCount)

	_, err = cli.Upsert(ctx, bson.M{"name": "Lucas"}, u, options.UpsertOptions{
		UpsertHook: myHook,
	})
	ast.Error(err)
	ast.Equal(2, myHook.beforeUsCount)
	ast.Equal(1, myHook.afterUsCount)

	myUpsertHook := &MyErrorHook{}
	_, err = cli.UpsertId(ctx, bson.M{"name": "Lucas"}, u, options.UpsertOptions{
		UpsertHook: myUpsertHook,
	})
	ast.Error(err)

}

/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package distribution

import (
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/core/cc/api"
	"configcenter/src/scene_server/event_server/types"
	"encoding/json"
	"fmt"
	redis "gopkg.in/redis.v5"
	"runtime/debug"
	"strconv"
	"time"
)

var (
	timeout    = time.Second * 10
	waitperiod = time.Second
)

var (
	ERR_WAIT_TIMEOUT   = fmt.Errorf("wait timeout")
	ERR_PROCESS_EXISTS = fmt.Errorf("process exists")
)

// StartHandleInsts distribute events to distribute queue
func StartHandleInsts() (err error) {
	defer func() {
		if err == nil {
			syserror := recover()
			if syserror != nil {
				err = fmt.Errorf("system error: %v", syserror)
			}
		}
		if err != nil {
			blog.Info("event inst handle process stoped by %v", err)
			debug.PrintStack()
		}
	}()

	blog.Info("event inst handle process started")
	for {
		// pod one event from cache
		event := popEventInst()
		if event == nil {
			continue
		}
		if err := handleInst(event); err != nil {
			blog.Errorf("error handle dist: %v, %v", err, event)
		}
	}
}

func handleInst(event *types.EventInstCtx) (err error) {
	blog.Info("handling event inst : %v", event.Raw)
	defer blog.Info("done event inst : %v", event.ID)
	if err = saveRunning(types.EventCacheEventRunningPrefix+fmt.Sprint(event.ID), timeout); err != nil {
		if ERR_PROCESS_EXISTS == err {
			blog.Infof("%v process exist, continue", event.ID)
			return nil
		}
		blog.Infof("save runtime error: %v, raw = %s", err, event.Raw)
		return err
	}

	// check previout done
	previousID := fmt.Sprint(event.ID - 1)
	priviousRunningkey := types.EventCacheEventRunningPrefix + previousID
	done, err := checkFromDone(types.EventCacheEventDoneKey, previousID)
	if err != nil {
		return err
	}
	if !done {
		running, checkErr := checkFromRunning(priviousRunningkey)
		if checkErr != nil {
			return checkErr
		}
		if !running {
			// wait a second to ensure previous not in trouble
			time.Sleep(time.Second * 3)
			running, checkErr = checkFromRunning(priviousRunningkey)
			if checkErr != nil {
				return checkErr
			}
		}
		if running {
			// if previous running, wait it
			if checkErr = waitPreviousDone(types.EventCacheEventDoneKey, previousID, timeout); checkErr != nil {
				if checkErr == ERR_WAIT_TIMEOUT {
					return nil
				}
				return checkErr
			}
		}
	}

	defer func() {
		if err != nil {
			blog.Errorf("prepare dist event error:%v", err)
		}
		err = SaveEventDone(event)
	}()

	// selete members

	origindists := GetDistInst(&event.EventInst)

	for _, origindist := range origindists {
		subscribers := findEventTypeSubscribers(origindist.GetType())
		if len(subscribers) <= 0 || "nil" == subscribers[0] {
			blog.Infof("%v no subscriber，continue", origindist.GetType())
			return SaveEventDone(event)
		}
		// prepare dist event
		for _, subscriber := range subscribers {
			var dstbID, subscribeID int64
			distinst := origindist
			dstbID, err = nextDistID(subscriber)
			if err != nil {
				return err
			}
			subscribeID, err = strconv.ParseInt(subscriber, 10, 64)
			if err != nil {
				return err
			}
			distinst.DstbID = dstbID
			distinst.SubscriptionID = subscribeID
			distByte, _ := json.Marshal(distinst)
			pushToQueue(types.EventCacheDistQueuePrefix+subscriber, string(distByte))
		}
	}

	return
}

var hostIndentDiffFiels = map[string][]string{
	common.BKInnerObjIDApp:    {common.BKAppNameField},
	common.BKInnerObjIDSet:    {common.BKSetNameField, "bk_service_status", "bk_set_env"},
	common.BKInnerObjIDModule: {common.BKModuleNameField},
	common.BKInnerObjIDPlat:   {common.BKCloudNameField},
	common.BKInnerObjIDHost: {common.BKHostNameField,
		common.BKCloudIDField, common.BKHostInnerIPField, common.BKHostOuterIPField,
		common.BKOSTypeField, common.BKOSNameField,
		"bk_mem", "bk_cpu", "bk_disk"},
}

func GetDistInst(e *types.EventInst) []types.DistInst {
	distinst := types.DistInst{
		EventInst: *e,
	}
	distinst.ID = 0
	var ds []types.DistInst
	var m map[string]interface{}
	if e.EventType == types.EventTypeInstData && e.ObjType == common.BKINnerObjIDObject {
		var ok bool

		if e.Action == "delete" && len(e.Data) > 0 {
			m, ok = e.Data[0].PreData.(map[string]interface{})
		} else {
			m, ok = e.Data[0].CurData.(map[string]interface{})
		}
		if !ok {
			return nil
		}

		if m[common.BKObjIDField] != nil {
			distinst.ObjType = m[common.BKObjIDField].(string)
		}

	}
	ds = append(ds, distinst)

	// add new dist if event belong to hostidentifier
	if diffFields, ok := hostIndentDiffFiels[e.ObjType]; ok && e.Action == types.EventActionUpdate && e.EventType == types.EventTypeInstData {
		for dataIndex := range e.Data {
			curdata := e.Data[dataIndex].CurData.(map[string]interface{})
			predata := e.Data[dataIndex].PreData.(map[string]interface{})
			if checkDifferent(curdata, predata, diffFields...) {
				hostIdentify := types.DistInst{
					EventInst: *e,
				}
				hostIdentify.Data = nil
				hostIdentify.EventType = types.EventTypeRelation
				hostIdentify.ObjType = "hostidentifier"

				instID, _ := curdata[common.GetInstIDField(e.ObjType)].(int)
				if instID == 0 {
					// this should wound happen -_-
					blog.Errorf("conver instID faile the raw is %v", curdata[common.GetInstIDField(e.ObjType)])
					continue
				}
				count := 0
				identifiers := GetIdentifierCache().getCache(e.ObjType, instID)
				total := len(identifiers)
				// pack identifiers into 1 distribution to prevent send too many messages
				for ident := range identifiers {
					count++
					d := types.EventData{PreData: *ident}
					for _, field := range diffFields {
						ident.Set(field, curdata[field])
					}
					d.CurData = *ident
					hostIdentify.Data = append(hostIdentify.Data, d)
					// each group is divided into 1000 units in order to limit the message size
					if count%1000 == 0 || count == total {
						ds = append(ds, hostIdentify)
						hostIdentify.Data = nil
					}
				}
			}
		}
	} else if e.EventType == types.EventTypeRelation && distinst.ObjType == "moduletransfer" {
		
	}

	return ds
}

func checkDifferent(curdata, predata map[string]interface{}, fields ...string) (isDifferent bool) {
	for _, field := range fields {
		if curdata[field] != predata[field] {
			return true
		}
	}
	return false
}

func GetIdentifierCache() *HostIdenCache {
	return &HostIdenCache{}
}

func pushToQueue(key, value string) (err error) {
	cacheValue := common.KvMap{
		"key":    key,
		"values": []string{string(value)},
	}
	_, err = api.GetAPIResource().CacheCli.Insert("rpush", cacheValue)
	blog.Infof("pushed to queue:%v", key)
	return
}

func nextDistID(eventtype string) (nextid int64, err error) {
	eventIDseletor := common.KvMap{
		"key": types.EventCacheDistIDPrefix + eventtype,
	}
	var id int
	id, err = api.GetAPIResource().CacheCli.Insert("incr", eventIDseletor)
	if err != nil {
		return
	}
	return int64(id), nil
}

func SaveEventDone(event *types.EventInstCtx) (err error) {
	redisCli := api.GetAPIResource().CacheCli.GetSession().(*redis.Client)
	if err = redisCli.HSet(types.EventCacheEventDoneKey, fmt.Sprint(event.ID), event.Raw).Err(); err != nil {
		return
	}
	if err = redisCli.Del(types.EventCacheEventRunningPrefix + fmt.Sprint(event.ID)).Err(); err != nil {
		return
	}
	return
}

func waitPreviousDone(key string, id string, timeout time.Duration) (err error) {
	var done bool
	timer := time.NewTimer(timeout)
	for !done {
		select {
		case <-timer.C:
			timer.Stop()
			return ERR_WAIT_TIMEOUT
		default:
			done, err = checkFromDone(key, id)
			if err != nil {
				return
			}
		}
		time.Sleep(waitperiod)
	}
	return
}

func checkFromDone(key string, id string) (bool, error) {
	if id == "0" {
		return true, nil
	}
	redisCli := api.GetAPIResource().CacheCli.GetSession().(*redis.Client)
	return redisCli.HExists(key, fmt.Sprint(id)).Result()
}

func checkFromRunning(key string) (bool, error) {
	redisCli := api.GetAPIResource().CacheCli.GetSession().(*redis.Client)
	return redisCli.Exists(key).Result()
}

func saveRunning(key string, timeout time.Duration) (err error) {
	// prevent other process handle the same event
	redisCli := api.GetAPIResource().CacheCli.GetSession().(*redis.Client)
	set, err := redisCli.SetNX(key, time.Now().UTC().Format(time.RFC3339), timeout).Result()
	if !set {
		return ERR_PROCESS_EXISTS
	}
	return err
}

func findEventTypeSubscribers(eventtype string) []string {
	redisCli := api.GetAPIResource().CacheCli.GetSession().(*redis.Client)
	return redisCli.SMembers(types.EventCacheSubscribeformKey + eventtype).Val()
}

func popEventInst() *types.EventInstCtx {
	var eventstr string

	redisCli := api.GetAPIResource().CacheCli.GetSession().(*redis.Client)
	redisCli.BRPopLPush(types.EventCacheEventQueueKey, types.EventCacheEventQueueDuplicateKey, time.Second*60).Scan(&eventstr)

	if eventstr == "" {
		return nil
	}

	// Unmarshal event
	eventbytes := []byte(eventstr)
	event := types.EventInst{}
	if err := json.Unmarshal(eventbytes, &event); err != nil {
		blog.Errorf("event distribute fail, unmarshal error: %v, date=[%s]", err, eventbytes)
		return nil
	}

	return &types.EventInstCtx{EventInst: event, Raw: eventstr}
}

package main

import (
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/nezha/pkg/log"
	"github.com/q191201771/nezha/pkg/unique"
	"sync"
)

// TODO chef: 没有sub了一定时间后，停止pull

type GroupManager struct {
	config     *Config
	appName    string
	streamName string

	exitChan     chan struct{}
	rtmpGroup    *rtmp.Group
	httpFlvGroup *httpflv.Group
	mutex        sync.Mutex

	UniqueKey string
}

func NewGroupManager(appName string, streamName string, config *Config) *GroupManager {
	uk := unique.GenUniqueKey("GROUPMANAGER")
	log.Infof("lifecycle new lal.GroupManager. [%s] appName=%s streamName=%s", uk, appName, streamName)

	return &GroupManager{
		config:     config,
		appName:    appName,
		streamName: streamName,
		exitChan:   make(chan struct{}),
		UniqueKey:  uk,
	}
}

func (gm *GroupManager) RunLoop() {
	<- gm.exitChan
}

func (gm *GroupManager) Dispose(err error) {
	log.Infof("lifecycle dispose lal.GroupManager. [%s] reason=%v", gm.UniqueKey, err)
	gm.exitChan <- struct{}{}
}

// 返回true则允许推流，返回false则关闭连接
func (gm *GroupManager) AddRTMPPubSession(session *rtmp.ServerSession, rtmpGroup *rtmp.Group) bool {
	gm.attachRTMPGroup(rtmpGroup)

	return !gm.isInExist()
}

func (gm *GroupManager) AddRTMPSubSession(session *rtmp.ServerSession, rtmpGroup *rtmp.Group) {
	gm.attachRTMPGroup(rtmpGroup)

	gm.pullIfNeeded()
}

func (gm *GroupManager) AddHTTPFlvSubSession(session *httpflv.SubSession, httpFlvGroup *httpflv.Group) {
	gm.attachHTTPFlvGroup(httpFlvGroup)

	gm.pullIfNeeded()
}

func (gm *GroupManager) IsTotalEmpty() bool {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	return (gm.rtmpGroup == nil || gm.rtmpGroup.IsTotalEmpty()) &&
		(gm.httpFlvGroup == nil || gm.httpFlvGroup.IsTotalEmpty())
}

// GroupObserver of rtmp.Group
func (gm *GroupManager) ReadRTMPAVMsgCB(header rtmp.Header, timestampAbs uint32, message []byte) {

	// TODO chef: broadcast to httpflv.Group
}

// GroupObserver of httpflv.Group
func (gm *GroupManager) ReadHTTPRespHeaderCB() {
	// noop
}

// GroupObserver of httpflv.Group
func (gm *GroupManager) ReadFlvHeaderCB(flvHeader []byte) {
	// noop
}

// GroupObserver of httpflv.Group
func (gm *GroupManager) ReadFlvTagCB(tag *httpflv.Tag) {
	log.Info("ReadFlvTagCB")

	// TODO chef: broadcast to rtmp.Group
}

func (gm *GroupManager) attachRTMPGroup(rtmpGroup *rtmp.Group) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	if gm.rtmpGroup != nil && gm.rtmpGroup != rtmpGroup {
		log.Warnf("duplicate rtmp group in group manager. %+v %+v", gm.rtmpGroup, rtmpGroup)
	}
	gm.rtmpGroup = rtmpGroup
	rtmpGroup.SetObserver(gm)
}

func (gm *GroupManager) attachHTTPFlvGroup(httpFlvGroup *httpflv.Group) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	if gm.httpFlvGroup != nil && gm.httpFlvGroup != httpFlvGroup {
		log.Warnf("duplicate http flv group in group manager. %+v %+v", gm.httpFlvGroup, httpFlvGroup)
	}
	gm.httpFlvGroup = httpFlvGroup
	httpFlvGroup.SetObserver(gm)
}

func (gm *GroupManager) pullIfNeeded() {
	if !gm.isInExist() {
		switch gm.config.Pull.Type {
		case "httpflv":
			go gm.httpFlvGroup.Pull(gm.config.Pull.Addr, gm.config.Pull.ConnectTimeout, gm.config.Pull.ReadTimeout)
		case "rtmp":
			go gm.rtmpGroup.Pull(gm.config.Pull.Addr, gm.config.Pull.ConnectTimeout)
		}
	}
}

func (gm *GroupManager) isInExist() bool {
	return (gm.rtmpGroup != nil && gm.rtmpGroup.IsInExist()) ||
		(gm.httpFlvGroup != nil && gm.httpFlvGroup.IsInExist())
}

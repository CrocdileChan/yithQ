package zero

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"yithQ/meta"
	"yithQ/util/logger"
	"yithQ/util/router"
)

type Zero struct {
	weightQueue      *WeightQueue
	cfg              *Config
	metadataVersion  uint32
	nodeTimer        *sync.Map //map[string]*time.Timer
	heartbeatTimeout time.Duration
}

func NewZero(cfg *Config) *Zero {
	timeout, err := time.ParseDuration(cfg.HeartbeatTimeout)
	if err != nil {
		logger.Lg.Fatalf("parse heartbeatTimeout(%s) to duration error : %v", cfg.HeartbeatTimeout, err)
	}
	return &Zero{
		weightQueue:      NewWeightQueue(),
		cfg:              cfg,
		metadataVersion:  0,
		nodeTimer:        &sync.Map{},
		heartbeatTimeout: timeout,
	}
}

func (z *Zero) Run() {
	logger.Lg.Info("zero start running ...")
	go z.ListenYith()
	select {}
}

func (z *Zero) ListenYith() {
	logger.Lg.Infof("zero listen yith nodes by port %s ", z.cfg.ListenPort)
	logger.Lg.Infof("nortify yith nodes by port %s", z.cfg.YithWatchPort)

	r := router.NewRouter()
	r.HandleFunc(http.MethodGet, "/"+meta.HeartbeatStr, z.ReceiveHeartbeat)
	r.HandleFunc(http.MethodPost, "/"+meta.TopicReplicaAddChangeStr, z.AddTopicReplica)
	r.HandleFunc(http.MethodGet, "/"+meta.FetchMetadataStr, z.ForFetchMetadata)
	r.HandleFunc(http.MethodPost, "/"+meta.TopicPartitionDeleteChangeStr, z.DeleteTopicPartition)
	r.HandleFunc(http.MethodPost, "/"+meta.PickupStr, z.YithPickup)
	http.ListenAndServe(z.cfg.ListenPort, r)

}

func (z *Zero) NortifyAllYiths() error {
	topicNodeMap := z.weightQueue.TopicNode()
	nodes := z.weightQueue.AllNodes()
	newVersion := atomic.AddUint32(&z.metadataVersion, 1)
	byt, err := meta.NewMetadata().Marshal(topicNodeMap, nodes, newVersion)
	if err != nil {
		return err
	}
	for _, nodeIp := range topicNodeMap {
		go func(nodeIp string) {
			node := strings.Split(nodeIp, ":")[0] + z.cfg.YithWatchPort
			resp, err := http.Post(node, "application/json", bytes.NewBuffer(byt))
			if err != nil {
				return
			}
			resp.Body.Close()
		}(nodeIp)

	}
	return nil
}

func (z *Zero) AddTopicReplica(w http.ResponseWriter, req *http.Request) {
	byt, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Lg.Errorf("yith(%s) add topic replica [read http body] error : %v", req.RemoteAddr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var topic meta.TopicMetadata
	err = json.Unmarshal(byt, &topic)
	if err != nil {
		logger.Lg.Errorf("yith(%s) add topic(%s) replica [json decode]  error : %v", req.RemoteAddr, topic.Topic, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	z.addTopicReplica(req.RemoteAddr, topic)
}

func (z *Zero) DeleteTopicPartition(w http.ResponseWriter, req *http.Request) {
	byt, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Lg.Errorf("yith(%s) delete topic  [read http body] error : %v", req.RemoteAddr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var topic meta.TopicMetadata
	err = json.Unmarshal(byt, &topic)
	if err != nil {
		logger.Lg.Errorf("yith(%s) delete topic(%s)  [json decode]  error : %v", req.RemoteAddr, topic.Topic, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	z.deleteTopicPartition(req.RemoteAddr, topic)
}

func (z *Zero) ForFetchMetadata(w http.ResponseWriter, req *http.Request) {
	topicNodeMap := z.weightQueue.TopicNode()
	nodes := z.weightQueue.AllNodes()
	version := atomic.LoadUint32(&z.metadataVersion)
	byt, err := meta.NewMetadata().Marshal(topicNodeMap, nodes, version)
	if err != nil {
		logger.Lg.Errorf("yith(%s) fetch metadata  error :%v", req.RemoteAddr, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(byt)
}

func (z *Zero) YithPickup(w http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Lg.Errorf("read yith(%s) pickup data error : %v", req.RemoteAddr, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	var topicMetadata []*meta.TopicMetadata
	json.Unmarshal(data, &topicMetadata)
	if err != nil {
		logger.Lg.Errorf("decode yith(%s) pickup data error : %v", req.RemoteAddr, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	for _, tm := range topicMetadata {
		for t, _ := range z.weightQueue.TopicNode() {
			if tm.Topic == t.Topic && tm.PartitionID == t.PartitionID {
				tm.IsReplica = t.IsReplica
			}
		}
	}

	byt, err := json.Marshal(topicMetadata)
	if err != nil {
		logger.Lg.Errorf("encode yith(%s) pickup data error : %v", req.RemoteAddr, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(byt)
}

func (z *Zero) ReceiveHeartbeat(w http.ResponseWriter, req *http.Request) {
	timer, ok := z.nodeTimer.Load(req.RemoteAddr)
	if !ok {
		f := func() {
			z.yithNodeExpire(req.RemoteAddr)
		}
		z.nodeTimer.Store(req.RemoteAddr, time.AfterFunc(z.heartbeatTimeout, f))
		z.weightQueue.AddNode(req.RemoteAddr)
		z.NortifyAllYiths()
		logger.Lg.Infof("New Yith node (%s) join in Zero", req.RemoteAddr)
	} else {
		timer.(*time.Timer).Reset(z.heartbeatTimeout)
	}
}

func (z *Zero) addTopicReplica(yithNode string, topic meta.TopicMetadata) {
	nodes := z.weightQueue.PopNodesWithout(topic.ReplicaFactory, yithNode)
	for i, node := range nodes {
		z.weightQueue.Put(node, meta.TopicMetadata{
			Topic:          topic.Topic,
			PartitionID:    topic.PartitionID*100 + i,
			IsReplica:      true,
			ReplicaFactory: topic.ReplicaFactory,
		})
	}
}

func (z *Zero) deleteTopicPartition(yithNode string, topic meta.TopicMetadata) {
	z.weightQueue.DeleteTopicPartition(topic)
}

func (z *Zero) yithNodeExpire(yithAddr string) {
	logger.Lg.Warnf("yith_node(%s) expired!", yithAddr)
	z.weightQueue.DeleteNode(yithAddr)
	z.nodeTimer.Delete(yithAddr)
}

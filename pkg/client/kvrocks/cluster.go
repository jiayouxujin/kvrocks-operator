package kvrocks

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const (
	ClusterNotInitErr     = "CLUSTERDOWN The cluster is not initialized"
	ClusterAlreadyMigrate = "Can't migrate slot which has been migrated"
	ClusterSlotInvalid    = "Can't migrate slot which doesn't belong to me"
	ClusterVersionInvalid = "Invalid cluster version"
	ClusterInvalidVersion = "Invalid version of cluster"
)

type ClusterOptions struct {
	Name     string   `json:"name"`
	Nodes    []string `json:"nodes"`
	Replicas int      `json:"replicas"`
	Password string   `json:"password"`
}

func (s *Client) CreateNamespace(endpoint string, namespace string) error {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := client.Post(
		endpoint+"namespaces",
		"application/json",
		strings.NewReader(`{"name": "`+namespace+`"}`),
	)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return err
	}
	return nil
}

func (s *Client) CreateCluster(endpoint string, nodes []string, namespace string, clusterName string) error {
	clusterOptions := &ClusterOptions{
		Name:     clusterName,
		Nodes:    nodes,
		Replicas: 2,
		Password: "123456",
	}
	clusterOptionsJson, err := json.Marshal(clusterOptions)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := client.Post(
		endpoint+"namespaces/"+namespace+"/clusters",
		"application/json",
		strings.NewReader(string(clusterOptionsJson)),
	)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return err
	}
	return nil
}

func (s *Client) SetClusterID(ip, password, nodeID string) error {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.Do(ctx, "CLUSTERX", "SETNODEID", nodeID).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("set cluster nodeID successfully", "ip", ip, "nodeId", nodeID)
	return nil
}

func (s *Client) SetTopoMsg(ip, password, topoMsg string, version int) error {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.Do(ctx, "CLUSTERX", "SETNODES", topoMsg, version).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("clusterx setnodes successfully", "ip", ip)
	return nil
}

func (s *Client) MoveSlots(ip, password string, slot int, dstNodeId string) bool {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.Do(ctx, "CLUSTERX", "MIGRATE", slot, dstNodeId).Err(); err != nil && (err.Error() == ClusterAlreadyMigrate || err.Error() == ClusterSlotInvalid) {
		return true
	}
	return false
}

func (s *Client) ResetSlot(ip, password string, slot, version int, dstNodeId string) error {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.Do(ctx, "CLUSTERX", "SETSLOT", slot, "NODE", dstNodeId, version).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("clusterx setslot successfully", "ip", ip, "node", dstNodeId, "slot", slot, "version", version)
	return nil
}

func (s *Client) ClusterVersion(ip, password string) (int, error) {
	c := kvrocksClient(ip, password)
	defer c.Close()
	result, err := c.Do(ctx, "CLUSTERX", "VERSION").Int()
	if err != nil {
		return -1, err
	}
	return result, nil
}

func (s *Client) ClusterNodeInfo(ip, password string) (*Node, error) {
	c := kvrocksClient(ip, password)
	defer c.Close()
	info, err := c.ClusterNodes(ctx).Result()
	if err != nil {
		return nil, err
	}
	return parseNodeInfo(info)
}

func parseNodeInfo(info string) (*Node, error) {
	node := &Node{}
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		fields := strings.Split(line, " ")
		if len(fields) < 8 {
			// last line is always empty
			continue
		}
		id := fields[0]
		flags := fields[2]
		if strings.Contains(flags, "myself") {
			node.NodeId = id
			node.IP = strings.Split(fields[1], ":")[0]
			if strings.Contains(flags, RoleMaster) {
				node.Role = RoleMaster
				node.Slots = SlotsToInt(fields[8:])
			} else if strings.Contains(flags, RoleSlaver) {
				node.Role = RoleSlaver
				node.Master = fields[3]
			}
		}
	}
	return node, nil
}

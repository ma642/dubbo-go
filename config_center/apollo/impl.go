/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package apollo

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	perrors "github.com/pkg/errors"

	"github.com/zouyx/agollo/v3"
	"github.com/zouyx/agollo/v3/env/config"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	cc "dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
)

const (
	apolloProtocolPrefix = "http://"
)

type apolloConfiguration struct {
	cc.BaseDynamicConfiguration
	url *common.URL

	listeners sync.Map
	appConf   *config.AppConfig
	parser    parser.ConfigurationParser
}

func newApolloConfiguration(url *common.URL) (*apolloConfiguration, error) {
	c := &apolloConfiguration{
		url: url,
	}
	c.appConf = &config.AppConfig{
		AppID:            url.GetParam(constant.CONFIG_APP_ID_KEY, ""),
		Cluster:          url.GetParam(constant.CONFIG_CLUSTER_KEY, ""),
		NamespaceName:    url.GetParam(constant.CONFIG_NAMESPACE_KEY, cc.DEFAULT_GROUP),
		IP:               c.getAddressWithProtocolPrefix(url),
		Secret:           url.GetParam(constant.CONFIG_SECRET_KEY, ""),
		IsBackupConfig:   url.GetParamBool(constant.CONFIG_BACKUP_CONFIG_KEY, true),
		BackupConfigPath: url.GetParam(constant.CONFIG_BACKUP_CONFIG_PATH_KEY, ""),
	}
	agollo.InitCustomConfig(func() (*config.AppConfig, error) {
		return c.appConf, nil
	})
	return c, agollo.Start()
}

func (c *apolloConfiguration) AddListener(key string, listener cc.ConfigurationListener, opts ...cc.Option) {
	k := &cc.Options{}
	for _, opt := range opts {
		opt(k)
	}

	key = k.Group + key
	l, _ := c.listeners.LoadOrStore(key, newApolloListener())
	l.(*apolloListener).AddListener(listener)
}

func (c *apolloConfiguration) RemoveListener(key string, listener cc.ConfigurationListener, opts ...cc.Option) {
	k := &cc.Options{}
	for _, opt := range opts {
		opt(k)
	}

	key = k.Group + key
	l, ok := c.listeners.Load(key)
	if ok {
		l.(*apolloListener).RemoveListener(listener)
	}
}

func (c *apolloConfiguration) GetInternalProperty(key string, opts ...cc.Option) (string, error) {
	newConfig := agollo.GetConfig(c.appConf.NamespaceName)
	if newConfig == nil {
		return "", perrors.New(fmt.Sprintf("nothing in namespace:%s ", key))
	}
	return newConfig.GetStringValue(key, ""), nil
}

func (c *apolloConfiguration) GetRule(key string, opts ...cc.Option) (string, error) {
	return c.GetInternalProperty(key, opts...)
}

// PublishConfig will publish the config with the (key, group, value) pair
func (c *apolloConfiguration) PublishConfig(string, string, string) error {
	return perrors.New("unsupport operation")
}

// GetConfigKeysByGroup will return all keys with the group
func (c *apolloConfiguration) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	return nil, perrors.New("unsupport operation")
}

func (c *apolloConfiguration) GetProperties(key string, opts ...cc.Option) (string, error) {
	/**
	 * when group is not null, we are getting startup configs(config file) from ShutdownConfig Center, for example:
	 * key=dubbo.propertie
	 */
	if key == "" {
		key = c.appConf.NamespaceName
	}
	tmpConfig := agollo.GetConfig(key)
	if tmpConfig == nil {
		return "", perrors.New(fmt.Sprintf("nothing in namespace:%s ", key))
	}

	content := tmpConfig.GetContent()
	b := []byte(content)
	if len(b) == 0 {
		return "", perrors.New(fmt.Sprintf("nothing in namespace:%s ", key))
	}

	content = string(b[8:]) //remove defalut content= prefix
	return content, nil
}

func (c *apolloConfiguration) getAddressWithProtocolPrefix(url *common.URL) string {
	address := url.Location
	converted := address
	if len(address) != 0 {
		addr := regexp.MustCompile(`\s+`).ReplaceAllString(address, "")
		parts := strings.Split(addr, ",")
		addrs := make([]string, 0)
		for _, part := range parts {
			addr := part
			if !strings.HasPrefix(part, apolloProtocolPrefix) {
				addr = apolloProtocolPrefix + part
			}
			addrs = append(addrs, addr)
		}
		converted = strings.Join(addrs, ",")
	}
	return converted
}

func (c *apolloConfiguration) Parser() parser.ConfigurationParser {
	return c.parser
}

func (c *apolloConfiguration) SetParser(p parser.ConfigurationParser) {
	c.parser = p
}

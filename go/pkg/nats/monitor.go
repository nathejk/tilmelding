package nats

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type Monitor struct {
	Url string
}

type Channel struct {
	LastSequence int64
}

func (m *Monitor) StanUrl() string {
	// Fetch port number to nats streaming server from "varz" endpoint.
	res, err := http.Get(m.Url + "/varz")
	if err != nil {
		panic(err.Error())
	}
	resBody, err := ioutil.ReadAll(res.Body)
	var varz struct {
		Port json.Number `json:"port,Number"`
	}
	err = json.Unmarshal(resBody, &varz)

	stanUrl, _ := url.Parse(m.Url)
	stanUrl.Scheme = "stan"
	stanUrl.Host = stanUrl.Hostname() + ":" + string(varz.Port)
	return stanUrl.String()
}

func (m *Monitor) ClusterId() string {
	res, err := http.Get(m.Url + "/streaming/channelsz?subs=1")
	if err != nil {
		panic(err.Error())
	}
	resBody, err := ioutil.ReadAll(res.Body)
	var channelszsubs struct {
		ClusterId string `json:"cluster_id"`
	}
	err = json.Unmarshal(resBody, &channelszsubs)
	return channelszsubs.ClusterId
}

func (m *Monitor) Channels() map[string]Channel {
	res, err := http.Get(m.Url + "/streaming/channelsz?subs=1")
	if err != nil {
		panic(err.Error())
	}
	resBody, err := ioutil.ReadAll(res.Body)

	var channelsz struct {
		Channels []struct {
			Name         string      `json:"name"`
			LastSequence json.Number `json:"last_seq,Number"`
		} `json:"channels"`
	}
	err = json.Unmarshal(resBody, &channelsz)
	if err != nil {
		log.Println(string(resBody), m.Url+"/streaming/channelsz?subs=1")
		panic(err)
	}

	channels := make(map[string]Channel)
	for _, channelInfo := range channelsz.Channels {
		lastSequence, err := channelInfo.LastSequence.Int64()
		if err != nil {
			panic(err)
		}
		channels[channelInfo.Name] = Channel{LastSequence: lastSequence}
	}

	return channels
}

func (m *Monitor) LastSequence(channel string) int64 {
	channels := m.Channels()
	if channel, exists := channels[channel]; exists {
		return channel.LastSequence
	}
	return 0
}

type natsMonitor struct {
	Url string
}

type monitorChannel struct {
	LastSequence int64
}

func (m *natsMonitor) StanUrl() string {
	// Fetch port number to nats streaming server from "varz" endpoint.
	res, err := http.Get(m.Url + "/varz")
	if err != nil {
		panic(err.Error())
	}
	resBody, err := ioutil.ReadAll(res.Body)
	var varz struct {
		Port json.Number `json:"port,Number"`
	}
	err = json.Unmarshal(resBody, &varz)

	stanUrl, _ := url.Parse(m.Url)
	stanUrl.Scheme = "stan"
	stanUrl.Host = stanUrl.Hostname() + ":" + string(varz.Port)
	return stanUrl.String()
}

func (m *natsMonitor) ClusterId() string {
	res, err := http.Get(m.Url + "/streaming/channelsz?subs=1")
	if err != nil {
		panic(err.Error())
	}
	resBody, err := ioutil.ReadAll(res.Body)
	var channelszsubs struct {
		ClusterId string `json:"cluster_id"`
	}
	err = json.Unmarshal(resBody, &channelszsubs)
	return channelszsubs.ClusterId
}

func (m *natsMonitor) Channels() map[string]monitorChannel {
	res, err := http.Get(m.Url + "/streaming/channelsz?subs=1")
	if err != nil {
		panic(err.Error())
	}
	resBody, err := ioutil.ReadAll(res.Body)

	var channelsz struct {
		Channels []struct {
			Name         string      `json:"name"`
			LastSequence json.Number `json:"last_seq,Number"`
		} `json:"channels"`
	}
	err = json.Unmarshal(resBody, &channelsz)
	if err != nil {
		log.Println(string(resBody), m.Url+"/streaming/channelsz?subs=1")
		panic(err)
	}

	channels := make(map[string]monitorChannel)
	for _, channelInfo := range channelsz.Channels {
		lastSequence, err := channelInfo.LastSequence.Int64()
		if err != nil {
			panic(err)
		}
		channels[channelInfo.Name] = monitorChannel{LastSequence: lastSequence}
	}

	return channels
}

func (m *natsMonitor) LastSequence(subject string) int64 {
	channels := m.Channels()
	if ch, exists := channels[subject]; exists {
		return ch.LastSequence
	}
	return 0
}

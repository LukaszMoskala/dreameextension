package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/klauspost/compress/zstd"
	"github.com/radovskyb/watcher"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sync"
	"time"
)

// Generated by goland when I pasted valetudo config file. I did not write this by hand :)

type ValetudoConfigT struct {
	Mqtt struct {
		Enabled    bool `json:"enabled"`
		Connection struct {
			Host string `json:"host"`
			Port int    `json:"port"`
			Tls  struct {
				Enabled                 bool   `json:"enabled"`
				Ca                      string `json:"ca"`
				IgnoreCertificateErrors bool   `json:"ignoreCertificateErrors"`
			} `json:"tls"`
			Authentication struct {
				Credentials struct {
					Enabled  bool   `json:"enabled"`
					Username string `json:"username"`
					Password string `json:"password"`
				} `json:"credentials"`
				ClientCertificate struct {
					Enabled     bool   `json:"enabled"`
					Certificate string `json:"certificate"`
					Key         string `json:"key"`
				} `json:"clientCertificate"`
			} `json:"authentication"`
		} `json:"connection"`
		Identity struct {
			Identifier string `json:"identifier"`
		} `json:"identity"`
		Interfaces struct {
			Homie struct {
				Enabled                   bool `json:"enabled"`
				AddICBINVMapProperty      bool `json:"addICBINVMapProperty"`
				CleanAttributesOnShutdown bool `json:"cleanAttributesOnShutdown"`
			} `json:"homie"`
			Homeassistant struct {
				Enabled                 bool `json:"enabled"`
				CleanAutoconfOnShutdown bool `json:"cleanAutoconfOnShutdown"`
			} `json:"homeassistant"`
		} `json:"interfaces"`
		Customizations struct {
			TopicPrefix    string `json:"topicPrefix"`
			ProvideMapData bool   `json:"provideMapData"`
		} `json:"customizations"`
		OptionalExposedCapabilities []string `json:"optionalExposedCapabilities"`
	} `json:"mqtt"`
	Valetudo struct {
		Customizations struct {
			FriendlyName string `json:"friendlyName"`
		} `json:"customizations"`
	} `json:"valetudo"`
}

var vconf ValetudoConfigT

// DiscoveryT is a minimal configuration for homeassistant mqtt discovery.
type DiscoveryT struct {
	Availability struct {
		PayloadAvailable    string `json:"payload_available"`
		PayloadNotAvailable string `json:"payload_not_available"`
		Topic               string `json:"topic"`
	} `json:"availability"`
	ContentType string `json:"content_type"`
	ImageTopic  string `json:"image_topic"`
	Name        string `json:"name"`
}

// make sure we don't try to play two sounds at once
// however we still may conflict with system trying to play something
var aplayMutex sync.Mutex

func playSound() {
	aplayMutex.Lock()
	defer aplayMutex.Unlock()
	files, err := os.ReadDir("/data/dreameextension/obstacleVoices")
	if err != nil {
		log.Printf("Could not read obstacle voices directory: %v", err)
		return
	}
	l := len(files)
	if l > 0 {
		x := rand.Int31n(int32(l))

		f := files[x]
		log.Printf("Obstacle voice file: %s", f.Name())
		c := exec.Command("aplay", "-Dhw:0,0", path.Join("/data/dreameextension/obstacleVoices", f.Name()))
		err = c.Run()
		if err != nil {
			log.Printf("Could not play obstacle voice file: %v", err)
		}

	}

}

func mqttSound(client mqtt.Client, message mqtt.Message) {
	zr, err := zstd.NewReader(bytes.NewReader(message.Payload()))
	if err != nil {
		log.Printf("Could not decompress message: %v", err)
	}
	defer zr.Close()
	aplayMutex.Lock()
	defer aplayMutex.Unlock()

	c := exec.Command("aplay", "-Dhw:0,0", "-")
	c.Stdin = zr
	err = c.Run()
	if err != nil {
		log.Printf("Could not run command: %v", err)
	}

}

func main() {
	func() {
		f, err := os.Open("/data/valetudo_config.json")
		if err != nil {
			log.Fatalf("Could not open valetudo config file: %v", err)
		}
		defer f.Close()
		err = json.NewDecoder(f).Decode(&vconf)
		if err != nil {
			log.Fatalf("Could not parse valetudo config file: %v", err)
		}
		if vconf.Mqtt.Customizations.TopicPrefix == "" {
			vconf.Mqtt.Customizations.TopicPrefix = "valetudo"
		}
		if vconf.Mqtt.Identity.Identifier == "" {
			log.Fatalf("Mqtt.Identity.Identifier is required. Please set it in valetudo.")
		}
		if vconf.Valetudo.Customizations.FriendlyName == "" {
			log.Fatalf("Please set friendlyName in valetudo.")
		}
	}()

	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", vconf.Mqtt.Connection.Host, vconf.Mqtt.Connection.Port)).SetClientID(fmt.Sprintf("dreameextension-%s", vconf.Mqtt.Identity.Identifier))
	opts.SetKeepAlive(10 * time.Second)
	opts.SetPingTimeout(4 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetWill(fmt.Sprintf("%s/%s/dreameextension/availability", vconf.Mqtt.Customizations.TopicPrefix, vconf.Mqtt.Identity.Identifier), "offline", 1, true)
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		t := client.Publish(fmt.Sprintf("%s/%s/dreameextension/availability", vconf.Mqtt.Customizations.TopicPrefix, vconf.Mqtt.Identity.Identifier), 1, true, "online")
		t.Wait()
		if t.Error() != nil {
			log.Printf("Could not publish availability status: %v", t.Error())
		} else {
			log.Printf("MQTT connected")
			client.Subscribe(fmt.Sprintf("%s/%s/dreameextension/play", vconf.Mqtt.Customizations.TopicPrefix, vconf.Mqtt.Identity.Identifier), 2, mqttSound)

			d := &DiscoveryT{}
			d.Availability.PayloadAvailable = "online"
			d.Availability.PayloadNotAvailable = "offline"
			d.Availability.Topic = fmt.Sprintf("%s/%s/dreameextension/availability", vconf.Mqtt.Customizations.TopicPrefix, vconf.Mqtt.Identity.Identifier)
			d.Name = fmt.Sprintf("%s obstacle", vconf.Valetudo.Customizations.FriendlyName)
			d.ImageTopic = fmt.Sprintf("%s/%s/dreameextension/obstacleImage", vconf.Mqtt.Customizations.TopicPrefix, vconf.Mqtt.Identity.Identifier)
			d.ContentType = "image/jpeg"
			b, _ := json.Marshal(d)
			//TODO: configurable discovery prefix
			t = client.Publish("homeassistant/image/dreameextension-"+vconf.Mqtt.Identity.Identifier+"/config", 1, true, string(b))
			t.Wait()
			if t.Error() != nil {
				log.Printf("Could not publish discovery: %v", t.Error())
			}
		}
	})
	if vconf.Mqtt.Connection.Authentication.Credentials.Enabled {
		opts.SetUsername(vconf.Mqtt.Connection.Authentication.Credentials.Username)
		opts.SetPassword(vconf.Mqtt.Connection.Authentication.Credentials.Password)
	}
	if vconf.Mqtt.Connection.Tls.Enabled {
		log.Fatalf("TLS connection not yet supported. Patches welcome :)")
	}

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Could not connect to mqtt server: %v", token.Error())
	}
	w := watcher.New()
	w.SetMaxEvents(1)
	w.FilterOps(watcher.Create)
	r := regexp.MustCompile("\\.jpg$")
	w.AddFilterHook(watcher.RegexFilterHook(r, false))
	go func() {
		for {
			select {
			case event := <-w.Event:
				go playSound()
				//we get notified when file is created, not when robot is finished writing to this file.
				//So we add sleep to allow robot to finish writing to make sure we have complete file
				//before attempting to read it
				time.Sleep(1 * time.Second)
				data, err := os.ReadFile(event.Path)
				if err != nil {
					log.Println(err)
				} else {
					token := c.Publish(fmt.Sprintf("%s/%s/dreameextension/obstacleImage", vconf.Mqtt.Customizations.TopicPrefix, vconf.Mqtt.Identity.Identifier), 0, false, data)
					token.Wait()
					if token.Error() != nil {
						log.Printf("Failed to publish image to mqtt: %s", token.Error())
					} else {
						log.Printf("Published")
					}
				}

			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()
	// Watch this folder for changes.
	if err := w.Add("/data/record"); err != nil {
		log.Fatalln(err)
	}

	// Start the watching process - it'll check for changes every 500ms.
	if err := w.Start(time.Millisecond * 500); err != nil {
		log.Fatalln(err)
	}
}

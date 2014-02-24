// Copyright 2013 Joyent Inc.
// Licensed under the AGPLv3, see LICENCE file for details.

package joyent

import (
	"fmt"
	"bytes"
	"strings"
	"text/template"

	"launchpad.net/juju-core/constraints"
	"launchpad.net/juju-core/environs"
	"launchpad.net/juju-core/environs/instances"
	"launchpad.net/juju-core/environs/jujutest"
	"launchpad.net/juju-core/environs/simplestreams"
	"launchpad.net/juju-core/environs/storage"

	"launchpad.net/gojoyent/jpc"
	//"launchpad.net/juju-core/errors"
)

var Provider environs.EnvironProvider = GetProviderInstance()

var indexData = `
		{
		 "index": {
		  "com.ubuntu.cloud:released:joyent": {
		   "updated": "Fri, 14 Feb 2014 13:39:35 +0000",
		   "clouds": [
			{
			 "region": "{{.Region}}",
			 "endpoint": "{{.SdcEndpoint.URL}}"
			}
		   ],
		   "cloudname": "joyent",
		   "datatype": "image-ids",
		   "format": "products:1.0",
		   "products": [
			"com.ubuntu.cloud:server:12.04:amd64",
			"com.ubuntu.cloud:server:12.10:amd64",
			"com.ubuntu.cloud:server:13.04:amd64"
		   ],
		   "path": "streams/v1/com.ubuntu.cloud:released:joyent.json"
		  }
		 },
		 "updated": "Fri, 14 Feb 2014 13:39:35 +0000",
		 "format": "index:1.0"
		}
`

var imagesData = `
{
  "content_id": "com.ubuntu.cloud:released:joyent",
  "format": "products:1.0",
  "updated": "Fri, 14 Feb 2014 13:39:35 +0000",
  "datatype": "image-ids",
  "products": {
    "com.ubuntu.cloud:server:12.04:amd64": {
      "release": "precise",
      "version": "12.04",
      "arch": "amd64",
      "versions": {
        "20140214": {
          "items": {
            "11223344-0a0a-ff99-11bb-0a1b2c3d4e5f": {
              "region": "some-region",
              "id": "11223344-0a0a-ff99-11bb-0a1b2c3d4e5f",
              "virt": "virtualmachine"
            }
          },
          "pubname": "ubuntu-precise-12.04-amd64-server-20140214",
          "label": "release"
        }
      }
    },
    "com.ubuntu.cloud:server:12.10:amd64": {
      "release": "quantal",
      "version": "12.10",
      "arch": "amd64",
      "versions": {
        "20140214": {
          "items": {
            "11223344-0a0a-ee88-22ab-00aa11bb22cc": {
              "region": "some-region",
              "id": "11223344-0a0a-ee88-22ab-00aa11bb22cc",
              "virt": "virtualmachine"
            }
          },
          "pubname": "ubuntu-quantal-12.10-amd64-server-20140214",
          "label": "release"
        }
      }
    },
    "com.ubuntu.cloud:server:13.04:amd64": {
      "release": "raring",
      "version": "13.04",
      "arch": "amd64",
      "versions": {
        "20140214": {
          "items": {
            "11223344-0a0a-dd77-33cd-abcd1234e5f6": {
              "region": "some-region",
              "id": "11223344-0a0a-dd77-33cd-abcd1234e5f6",
              "virt": "virtualmachine"
            }
          },
          "pubname": "ubuntu-raring-13.04-amd64-server-20140214",
          "label": "release"
        }
      }
    }
  }
}
`

const productMetadataFile = "streams/v1/com.ubuntu.cloud:released:joyent.json"

func parseIndexData(creds *jpc.Credentials) bytes.Buffer {
	var metadata bytes.Buffer

	t := template.Must(template.New("").Parse(indexData))
	if err := t.Execute(&metadata, creds); err != nil {
		panic(fmt.Errorf("cannot generate index metdata: %v", err))
	}

	return metadata
}

// This provides the content for code accessing test://host/... URLs. This allows
// us to set the responses for things like the Metadata server, by pointing
// metadata requests at test://host/...
var testRoundTripper = &jujutest.ProxyRoundTripper{}

func init() {
	testRoundTripper.RegisterForScheme("test")
}

// Set Metadata requests to be served by the filecontent supplied.
func UseExternalTestImageMetadata(creds *jpc.Credentials) {
	metadata := parseIndexData(creds)
	files := map[string]string{
		"/streams/v1/index.json": metadata.String(),
		"/streams/v1/com.ubuntu.cloud:released:joyent.json": imagesData,
	}
	testRoundTripper.Sub = jujutest.NewCannedRoundTripper(files, nil)
}

func UnregisterExternalTestImageMetadata() {
	testRoundTripper.Sub = nil
}

// MetadataStorage returns a Storage instance which is used to store simplestreams metadata for tests.
func MetadataStorage(e environs.Environ) storage.Storage {
	container := "juju-test-metadata"
	metadataStorage := NewStorage(e.(*JoyentEnviron), container)

	// Ensure the container exists.
	err := metadataStorage.(*JoyentStorage).CreateContainer()
	if err != nil {
		panic(fmt.Errorf("cannot create %s container: %v", container, err))
	}
	return metadataStorage
}

// ImageMetadataStorage returns a Storage object pointing where the gojoyent
// infrastructure sets up its entry for image metadata
func ImageMetadataStorage(e environs.Environ) storage.Storage {
	env := e.(*JoyentEnviron)
	return NewStorage(env, "juju-test-metadata")
}

func UseStorageTestImageMetadata(stor storage.Storage, creds *jpc.Credentials) {
	// Put some image metadata files into the public storage.
	metadata := parseIndexData(creds)
	data := metadata.Bytes()
	stor.Put("images/"+simplestreams.DefaultIndexPath+".json", bytes.NewReader(data), int64(len(data)))
	stor.Put("images/"+productMetadataFile, strings.NewReader(imagesData), int64(len(imagesData)))
}

func RemoveStorageTestImageMetadata(stor storage.Storage) {
	stor.Remove("images/"+simplestreams.DefaultIndexPath + ".json")
	stor.Remove("images/"+productMetadataFile)
}

func FindInstanceSpec(e environs.Environ, series, arch, cons string) (spec *instances.InstanceSpec, err error) {
	env := e.(*JoyentEnviron)
	spec, err = env.FindInstanceSpec(&instances.InstanceConstraint{
		Series:      series,
		Arches:      []string{arch},
		Region:      env.Ecfg().Region(),
		Constraints: constraints.MustParse(cons),
	})
	return
}

func ControlBucketName(e environs.Environ) string {
	env := e.(*JoyentEnviron)
	return env.Storage().(*JoyentStorage).GetContainerName()
}

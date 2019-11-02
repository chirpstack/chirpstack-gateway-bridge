---
title: GCP Cloud IoT Core
menu:
    main:
        parent: integrate
        weight: 3
description: Setting up the ChirpStack Gateway Bridge using the GCP Cloud IoT Core MQTT Bridge.
---

# Google Cloud Platform Cloud IoT Core

The Google Cloud Platform [Cloud IoT Core](https://cloud.google.com/iot-core/)
authentication type must be used when connecting to the
[Cloud IoT Core MQTT Bridge](https://cloud.google.com/iot/docs/how-tos/mqtt-bridge).
Cloud IoT Core will publish the received events from the LoRa<sup>&reg;</sup> Gateway to
a Google Cloud Platform [Cloud Pub/Sub topic](https://cloud.google.com/pubsub/).

## Limitations

* Please note that this authentication type is only available for the `json` or
  `protobuf` marshaler.
* As you need to setup the device ID (in this case the device is the gateway)
  when provisioning the device (LoRa Gateway) in Cloud IoT Core,
  this does not allow to connect multiple LoRa Gateways to a single ChirpStack Gateway
  Bridge instance.

## Conventions

### Device ID naming

The Cloud IoT Core device ID must equal `gw-[GATEWAY_ID]`. So when your gateway ID
equals to `0102030405060708`, then your Cloud IoT Core device ID equals to
`gw-0102030405060708`.

### MQTT topics

When the Google Cloud Platform Cloud IoT Core authentication type has been
configured, ChirpStack Gateway Bridge will use MQTT topics which are expected by
Cloud IoT Core and will ignore the MQTT topics as configured in the
`chirpstack-gateway-bridge.toml` configuration file.

#### Uplink topics

* `/devices/gw-[GATEWAY_ID]/events/up`: uplink frame
* `/devices/gw-[GATEWAY_ID]/events/stats`: gateway statistics
* `/devices/gw-[GATEWAY_ID]/events/ack`: downlink frame acknowledgements (scheduling)

#### Downlink topics

* `/devices/gw-[GATEWAY_ID]/commands/down`: scheduling downlink frame transmission
* `/devices/gw-[GATEWAY_ID]/commands/config`: gateway configuration

## Sending commands to the LoRa Gateway

For sending commands to the LoRa Gateway (e.g. scheduling a downlink transmission
or reconfigure the channel-plan), you can use the Cloud IoT Core
[sendCommandToDevice](https://cloud.google.com/iot/docs/reference/rest/) API method.

### Cloud Function

When you would like to use a Pub/Sub topic for sending commands to the
LoRa Gateway, you could use Cloud Function to trigger the `sendCommandToDevice`
API method. Example:

#### `index.js`

You need to replace:

* `REGION` with your GCP region
* `PROJECT_ID` with your GCP project ID
* `REGISTRY_ID` with your GCP Cloud IoT Core registry ID

{{<highlight js>}}
'use strict';

const {google} = require('googleapis');

// configuration options
const REGION = 'europe-west1';
const PROJECT_ID = 'example-project';
const REGISTRY_ID = 'eu868-gateways';


let client = null;
const API_VERSION = 'v1';
const DISCOVERY_API = 'https://cloudiot.googleapis.com/$discovery/rest';


// getClient returns the GCP API client.
// Note: after the first initialization, the client will be cached.
function getClient (cb) {
  if (client !== null) {
    cb(client);
    return;
  }

  google.auth.getClient({scopes: ['https://www.googleapis.com/auth/cloud-platform']}).then((authClient => {
    google.options({
      auth: authClient
    });

    const discoveryUrl = `${DISCOVERY_API}?version=${API_VERSION}`;
    google.discoverAPI(discoveryUrl).then((c, err) => {
      if (err) {
        console.log('Error during API discovery', err);
        return undefined;
      }
      client = c;
      cb(client);
    });
  }));
}


// sendMessage forwards the Pub/Sub message to the given device.
exports.sendMessage = (event, context, callback) => {
  const deviceId = event.attributes.deviceId;
  const subFolder = event.attributes.subFolder;
  const data = event.data;
  
  getClient((client) => {
    const parentName = `projects/${PROJECT_ID}/locations/${REGION}`;
    const registryName = `${parentName}/registries/${REGISTRY_ID}`;
    const request = {
      name: `${registryName}/devices/${deviceId}`,
      binaryData: data,
      subfolder: subFolder
    };
    
    console.log("start call sendCommandToDevice");
    client.projects.locations.registries.devices.sendCommandToDevice(request, (err, data) => {
      if (err) {
        console.log("Could not send command:", request, "Message:", err);
        callback(new Error(err));
      } else {
        callback();
      }
    });
  });
};
{{< /highlight >}}

#### `package.json`


{{<highlight json>}}
{
  "name": "gateway-commands",
  "version": "2.0.0",
  "dependencies": {
    "@google-cloud/pubsub": "0.20.1",
    "googleapis": "34.0.0"
  }
}
{{< /highlight >}}

#### Attributes

Besides the `json` or `protobuf` message, the above Cloud Function expects the
following attributes:

* `deviceId`:  the LoRa Gateway ID (e.g. `gw-0102030405060708`)
* `subFolder`: the command type
   * `down`: for scheduling downlink frames
   * `config`: for sending gateway configuration

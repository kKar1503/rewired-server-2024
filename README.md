# ReWired 2024 Hackathon ReRemote Server

This is the backend implementation to receive raw TCP packets from the ReRemote IoT devices, process the data, store into
SQLite database and expose the data through a WebSocket server.

There is a [client implementation](https://github.com/kKar1503/rewired-client-2024) made to showcase what this backend's 
WebSocket server exposes.

## Deployed Live URL \[Currently Active\]

Currently, the backend project's WebSocket server is being deployed on the following URL:

<wss://rewired-server.karlok.dev/ws>

There is also a debug WebSocket server that outputs the raw TCP packets sent to the server as JSON data for debugging purposes, 
deployed on the following URL:

<wss://rewired-server.karlok.dev/debug>

## Technical Implementations

This section will delve deeper into the details of the technical implementation used in the ReRemote server.

### TCP Server

The choice to use a TCP server instead of HTTP / [MQTT](https://mqtt.org/) is due to our implementation will require us to 
send data from the devices to the server at a high frequency, therefore, sending data in HTTP or MQTT protocol will require 
us to send much larger packets, and thus may not be able to fulfill our needs of high throughput. With our implementation,
we are able to keep the packet size of our data to be a maximum of 64 bits or 8 bytes.

The implementation of the TCP server always reads the first byte of the incoming packet to determine the type of the packet,
which for our service, we have 4 different type of packets:
- `0b00010001` - [Heartbeat](#heartbeat-packet)
- `0b00010010` - [Gate Status](#gate-status-packet)
- `0b00010011` - [Increment](#increment-packet)
- `0b00010100` - [Decrement](#decrement-packet)

The first byte is actually a combination of 2 x 4 bits value. The first 4 bits represent the `Version` number, which in our case
is always 1 (represented in binary as `0b0001`). The next 4 bits represent the `Type` number, which uniquely determines the type
of the packet.

After determine the type of the packet using the first byte, then the server will, based on the type of the packet, the TCP server
will read a selected number of bytes, depending on the packet type.

#### Heartbeat Packet

The Heartbeat Packet is used to monitor the status of the devices, to ensure that the devices are online and connected to the backend.
The packet is expected to be sent to the server at least once every 60 seconds to be considerd online.

The packet size is a total of 3 bytes with the structure as follows:

| Version |  Type  | Device ID |
|---------|--------|-----------|
| 4 bits  | 4 bits |  16 bits  |

#### Gate Status Packet

The Gate Status Packet is used to transmit the data that represents one out of four possible status that the device is in:
- `0b00000001` - Turn On
- `0b00000010` - Unblocked
- `0b00000011` - Blocked
- `0b00000100` - Faulty

Using the status, the server then can use to determine and then outputs as data to clients on the status of the devices and
determines the flow of population in and out of individual registered rooms.

The packet size is a total of 8 bytes with the structure as follows:

| Version |  Type  | Device ID | Status | Epoch Unix Time |
|---------|--------|-----------|--------| ----------------|
| 4 bits  | 4 bits |  16 bits  | 8 bits |     32 bits     |

#### Increment Packet

The Increment Packet is used to increment the population in the 
[_inner room_](#why-is-the-increment-and-decrement-packet-using-the-inner-room) of the device which the ID is provided. 

The packet size is a total of 3 bytes with the structure as follows:

| Version |  Type  | Device ID |
|---------|--------|-----------|
| 4 bits  | 4 bits |  16 bits  |

#### Decrement Packet

The Decrement Packet is used to decrement the population in the 
[_inner room_](#why-is-the-increment-and-decrement-packet-using-the-inner-room) of the device which the ID is provided. 

The packet size is a total of 3 bytes with the structure as follows:

| Version |  Type  | Device ID |
|---------|--------|-----------|
| 4 bits  | 4 bits |  16 bits  |

#### Why is the Increment and Decrement packet using the _inner room_?

As the ReRemote device is designed with 2 buttons, one to increment and another to decrement. They are used to specifically change
the population of one room to avoid confusion, therefore, in the default behaviour the increment the inner room, which the idea is
that the devices are _at the door leading into the room_.

### Processing Movements through the Devices

WIP

### SQLite

As in this project, we want to store the data in a separate place to minimize the data passing between the TCP Server and the 
WebSocket server, while keeping the implementations simplest possible, therefore, I chose the lightweight SQLite for this implementation.

### WebSocket Server

The WebSocket server is implemented with a central manager that manages all the connected clients, receives the processed data from 
the SQLite database and distributes the processed data to each socket. 

## Product: ReRemote

The product ReRemote is a product aim to repurpose IR sensors from remote controllers to act as gates that would help track
the number of persons that pass by.

The product is design to provide the ability to keep track of the number of people in a specific area based on people moving
through the ReRemote's gates. With that, we would have the ability to downstream this information to control room
temperature, air conditioning, and lighting in an efficient manner. For instance, if nobody is present in an area, one might
turn off the power of certain appliances to save energy.

However, the possibility for such integration is endless, as the product only provides a simple, elegent and yet infinitely
extensible interface that allows for all types of integration that can utilise the number of people in different section of
the house.

## Team Members

- [Yam Kar Lok](https://github.com/kKar1503)
- [Hong Yu]()
- [Thaddeus]()
- [Kenneth Chen]()
- [Rezky]()

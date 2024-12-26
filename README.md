# Ledis Report

## Overview

Ledis is a simplified, lightweight version of Redis, designed to handle basic data structures such as Strings and Sets, along with special features like data expiration and snapshots. It also includes a simple web CLI for interacting with the data store.
The demo here: https://ledis.onrender.com

## Architecture

The architecture of Ledis is designed to be thread-safe and efficient, utilizing Go's concurrency primitives and data structures. The main components of the architecture are:

1. **Data Structures**: Ledis supports two primary data structures - Strings and Sets.
2. **Key Management**: Keys are managed with expiration times and garbage collection.
3. **Garbage Collector**: A garbage collector is implemented to automatically remove expired keys.
4. **Snapshots**: The ability to save and restore the state of the database.
5. **Web CLI**: A simple command-line interface for interacting with Ledis.

### Data Structures

Ledis supports the following data structures:

- **Strings**: Simple key-value pairs where the value is a string.
- **Sets**: Unordered collections of unique string values.

### Key Management

Keys in Ledis are managed using a `Key` struct, which includes the following fields:

- `name`: The name of the key.
- `_type`: The type of the key (String or Set).
- `lastRenewed`: The last time the key was renewed.
- `expireTime`: The expiration time of the key.

The `Key` struct provides methods to renew the key, check if it is expired, and clone the key.

### Garbage Collector

The garbage collector is responsible for removing expired keys from the database. It uses an interval-based approach to periodically check for expired keys and remove them. The garbage collector is implemented in the `garbage_collector.go` file and uses a Left-Leaning Red-Black (LLRB) tree to store keys in order of their expiration times.

#### Key Methods

- `add(key *Key)`: Adds a key to the garbage collector.
- `remove(key *Key)`: Removes a key from the garbage collector.
- `clean(ctx context.Context)`: Periodically checks for expired keys and removes them.
- `clone(ledis *Ledis)`: Creates a clone of the garbage collector for snapshot purposes.
- `stop()`: Stops the garbage collector.

### Snapshots

Ledis supports saving the current state of the database and restoring it from a snapshot. This is useful for backup and recovery purposes. The `save` method creates a deep copy of the current Ledis instance and stores it in the `snapshots` field. The `restore` method restores the state from the last saved snapshot.

### Web CLI

A simple web CLI is provided to interact with Ledis. Commands are parsed and validated using utility functions, and the appropriate methods are called to handle each command. The supported commands include:

- **String Commands**: `SET`, `GET`
- **Set Commands**: `SADD`, `SREM`, `SMEMBERS`, `SINTER`
- **Key Management Commands**: `KEYS`, `DEL`, `EXPIRE`, `TTL`
- **Snapshot Commands**: `SAVE`, `RESTORE`

## Implementations

### Ledis Struct (Data Management)

The `Ledis` struct is the core of the data management system in Ledis. It contains maps for storing atomic data (strings) and set data, as well as a map for managing keys and their metadata. The `Ledis` struct also includes a mutex for thread safety, a reference to the garbage collector, and a field for snapshots.

#### Fields

- `atomicData`: A map that stores string values, with keys as strings and values as pointers to strings.
- `setData`: A map that stores sets, with keys as strings and values as pointers to maps of string keys with boolean values.
- `keys`: A map that stores key metadata, with keys as strings and values as pointers to `Key` structs.
- `mu`: A read-write mutex (`sync.RWMutex`) for synchronizing access to the data store.
- `snapshots`: A reference to another `Ledis` instance used for storing snapshots.
- `gc`: A reference to the garbage collector.

#### Methods

- `HandleCommand(cmd string)`: Parses and validates commands, and calls the appropriate method to handle each command.
- `deleteKey(key *Key)`: Deletes a key from the database.
- `set(key *Key, value string)`: Sets a string value for a key.
- `get(key *Key)`: Gets the string value of a key.
- `sadd(key *Key, values ...string)`: Adds values to a set.
- `srem(key *Key, values ...string)`: Removes values from a set.
- `smembers(key *Key)`: Returns all members of a set.
- `sinter(keys ...*Key)`: Returns the intersection of multiple sets.
- `listKeys()`: Lists all available keys.
- `setExpireKey(key *Key, expireTime time.Duration)`: Sets an expiration time for a key.
- `getTTL(key *Key)`: Gets the time-to-live (TTL) of a key.
- `save()`: Saves the current state of the database to a snapshot.
- `restore()`: Restores the database from the last snapshot.
- `convertKeys(keys ...string)`: Converts key names to `Key` objects.
- `renewKey(key *Key)`: Renews a key's expiration time.

### Key Struct (Key Management)

The `Key` struct is used to manage metadata for each key in the database. It includes fields for the key name, type, last renewed time, and expiration time. The `Key` struct provides methods to renew the key, check if it is expired, and clone the key.

#### Fields

- `name`: The name of the key.
- `_type`: The type of the key (String or Set).
- `lastRenewed`: The last time the key was renewed.
- `expireTime`: The expiration time of the key.

#### Methods

- `renew()`: Updates the `lastRenewed` time to the current time.
- `getExpireTime()`: Calculates the expiration time by adding the `expireTime` to the `lastRenewed` time.
- `isExpired()`: Checks if the key is expired by comparing the current time with the expiration time.
- `clone()`: Creates a deep copy of the key.

### garbageCollector Struct (Cleaning Data Management)

The `garbageCollector` struct is responsible for cleaning up expired keys from the database. It uses an LLRB tree to store keys in order of their expiration times and periodically checks for expired keys to remove them.

#### Fields

- `tree`: An LLRB tree that stores keys in order of their expiration times.
- `ledis`: A reference to the `Ledis` instance.
- `cancel`: A cancel function for stopping the garbage collector.
- `mu`: A read-write mutex (`sync.RWMutex`) for synchronizing access to the LLRB tree.

#### Methods

- `newGarbageCollector(ledis *Ledis)`: Initializes a new garbage collector and starts the clean process.
- `add(key *Key)`: Adds a key to the garbage collector.
- `remove(key *Key)`: Removes a key from the garbage collector.
- `clean(ctx context.Context)`: Periodically checks for expired keys and removes them.
- `clone(ledis *Ledis)`: Creates a clone of the garbage collector for snapshot purposes.
- `stop()`: Stops the garbage collector.

### Challenges and Interesting Points

#### Challenges

1. **Thread Safety**: Ensuring thread safety was a key challenge, especially with concurrent access and modifications to the data store. This was addressed using mutexes (`sync.RWMutex`) to protect shared resources.
2. **Garbage Collection**: Implementing an efficient garbage collector that periodically removes expired keys without impacting performance was challenging. The use of an LLRB tree helped in maintaining a sorted order of keys based on their expiration times.
3. **Snapshot Functionality**: Creating deep copies of the entire Ledis instance for snapshots required careful handling of pointers and data structures to ensure that the snapshots were independent of the current state.

#### Interesting Points

1. **LLRB Tree for Garbage Collection**: The use of an LLRB tree for managing keys based on their expiration times was an interesting design choice. It ensured that the garbage collector could efficiently find and remove expired keys.
2. **Modular Design**: The modular design of Ledis made it easy to extend and maintain. Each component (data structures, key management, garbage collector, snapshots) was implemented as a separate module with well-defined interfaces.
3. **Error Handling**: Providing clear and informative error messages helped in debugging and understanding issues. This was an important aspect of the user experience.

## Conclusion

The design and implementation of Ledis focused on creating a lightweight, efficient, and thread-safe in-memory data store with basic data structures and features. The `Ledis` struct handles data management, the `Key` struct manages key metadata, and the `garbageCollector` struct ensures effective cleaning of expired keys. The use of mutexes ensured thread safety, while efficient data structures like maps and LLRB trees provided fast access and management. The garbage collector and lazy removal mechanisms ensured effective data management by automatically and promptly removing expired keys. These design choices helped achieve the objectives of the project and provided a robust foundation for future extensions.

**Warning:** Do not expose confidential and sensitive data.

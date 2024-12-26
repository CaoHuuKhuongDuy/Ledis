# Ledis Report

## Overview

Ledis is a simplified, lightweight version of Redis, designed to handle basic data structures such as Strings and Sets, along with special features like data expiration and snapshots. It also includes a simple web CLI for interacting with the data store.
You can run the demo here: https://ledis.onrender.com (Note: The instance will spin down with inactivity, please wait for it to be active again)

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

#### Garbage collection mechanism

We have 2 mechanisms to clean up expired keys:

##### 1. Lazy Removal

We will check if a key is expired when we try to access it. If it is, we remove it.

##### 2. Garbage Collector

How about the keys that are not accessed frequently? It will never be removed by the lazy removal mechanism. The idea is using a background goroutine to periodically check for expired keys and remove them every 15 seconds. But we will need to iterate through all the keys in the garbage collector, which has the time complexity of O(N) with N is the number of keys in Ledis and it will not efficient for a large number of keys??!!

Improvement: we will use a red-black tree to store the keys in order of their expiration times, so we can iterate through the keys that have the expiration time less than the current time to delete them and won't need to iterate through the keys that haven't expired yet. So the time complexity will be O(K) with K is the number of keys that have expired. 

P/s: We also can use another data structure like priority queue (heap) to store the keys in order as well.

#### Get the set intersection algorithm

The basic idea is using a hash map to store the frequency of each element in the first set and then iterate through the elements of the others sets and increment the frequency of each element in the hash map. If the frequency of an element is equal to the number of sets, we add it to the result set. So the time complexity will be O(N) with N is the total number of elements in all sets.

In the worst case this algorithm is effective but what if we have 99 sets with 10^6 elements and 1 set with 1 element. We will need to iterate through 10^6 * 99 + 1 to find the result. We can make a small improvement for the average complexity by:

1. Store the elements of the set has the least number of elements in a hash map.
2. Interate through the others sets and check if the element in the hash map exists in the set, if it does not, we remove it from the hash map. The final hash map will only contain the elements that exist in all sets.

This algorithm will have the time complexity of O(K * M) with K is the smallest number of elements in all sets and M is the number of sets. In the worst case, this complexity is equivalent O(N) in the basic idea but in the average case, it will be much faster (eg with the case above, it will be O(10^6 * 1) = O(10^6)).

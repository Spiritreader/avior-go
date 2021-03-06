# DONE!

## Avior Db structure
- clients
- jobs
- sub_exclude
- name_exclude
- log_include
- log_exclude

## Tasks

### Database:
- [x] Get job
- [x] Insert job
- [x] Edit job
- [x] Remove job
- [x] Get client
- [x] Insert client
- [x] Edit client
- [x] Remove client
- [x] Get fields
- [x] Add fields
- [x] Remove fields

### Global state:
The purpose of this module is to serve as a collection of all state data that is currently outputted by the service. This includes:
- [x] Progress for encoding
- [x] Progress for size estimation
- [x] Progress for duplicate search
- [x] Progress for file moving
- [x] Encoder output string
- [x] duplicate search current file and directory
- [x] current file that is moved
- [x] current slice that is encoded for estimation
  

### Duplicate search:
- [x] find duplicates on disk
- [x] config file entry
- [x] global state integration
- [x] moving of old files when modules result is REPL


### Modules:
If there is a duplicate file present in the system, 
a module should determine if that duplicate file should be discarded and replaced by the new one or kept.
Each module should also have a priority that determines how important its output is (aka execution order on crack)

A single module can return:
- `REPL`: allows replacement
- `NOCH`: does nothing (no change / noch nicht)
- `KEEP`: disqualifies replacement

**Module Infrastructure:**
- [x] modules callable via interface
- [x] module config uniformity
- [x] module priority management

____
**Module Implementations:**
- [x] Compare resolutions:
    - better resolution should allow replacement
    - return modes: `REPL`, `NOCH`, `KEEP`
- [x] Compare audioformat:
    - better audio format should allow replacement
    - return modes: `REPL`, `NOCH`, `KEEP`
- [x] Estimate size of new file
    - better file size should allow (percentage threshold)
    - depends on: encoder
    - return modes: `REPL`, `NOCH`, `KEEP`
- [x] Check for include/exclude terms in logfile
    - include mode: if include and exclude match, include takes priority
    - neutral mode: swiss
    - exclude mode: if include and exclude match, exclude takes priority
    - return modes: `REPL`, `NOCH`, `KEEP`
- [x] Age (whiltelist)
    - files that have been created below a certain threshold age will be kept
    - return modes: `KEEP`, `NOCH`
- [x] Length (whitelist)
    - new files whose recorded lenghts differ greater than a configurable percent threshold should not be eligible for replacement
    - return modes: `KEEP`, `NOCH`
- [x] MaximumFileSize (whitelist)
    - if a new file is larger in Gigabytes than the specified filesize then it should not be eligible for replacement
    - return modes: `KEEP`, `NOCH`
- [x] Legacy checker
    - if a file is legacy, overwrite it

### Encoder

**Configuration:**

The encoder takes a configuration file based on a current resolution.
It consists of the following options:
- Output Paths
- ffmpeg pre arguments
- ffmpeg post arguments
- parked arguments (not in use)
- old format: '#' was comment, PI was pre-argument, none was post argument
```
enc_path \\UMS\media\transcoded\HD1080\

-c:v hevc_nvenc
#-vf yadif=0:-1:0 hqdn3d=2:1:2:3
-vf yadif=0:-1:0
#-vf hqdn3d=2:1:2:3
#-filter:v hqdn3d=2:1:2:3
#-filter:v scale=1280:720
-level 4.1
#-r 25
-rc vbr_hq
-qmin 16
-qmax 23
-rc-lookahead 32
#-rc constqp
#-qp 21
-tier main
-acodec copy
PI -test
```

**Implementation:**
- [x] config file entry
- [x] ffmpeg output parser
- [x] cli call constructor
- [x] global state integation
- [x] estimate remaining time

### http API

- [x] resume service
- [x] pause service
- [x] stop service
- [x] expose database operations outwards
- [x] export statistics
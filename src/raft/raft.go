package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"

	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"6.824/labrpc"
)

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

//
// A Go object implementing a single Raft peer.
//
type OpType uint32
type RoleType uint32

const (
	GET OpType = iota
	PUT
)
const (
	FOLLOWER RoleType = iota
	CANDIDATE
	LEADER
)

type Entry struct {
	Term  int
	Index int

	Op    OpType
	Key   string
	Value string
}
type RaftLog struct {
	entries []Entry
}

func (rfLog *RaftLog) GetLastLogIndex() int {
	lastLogIndex := rfLog.entries[len(rfLog.entries)-1].Index
	return lastLogIndex
}

func (rfLog *RaftLog) GetLastLogTerm() int {
	if len(rfLog.entries) == 0 {
		return 0
	}
	lastLogTerm := rfLog.entries[len(rfLog.entries)-1].Term
	return lastLogTerm
}

type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	CurrentTerm int
	VatedFor    int
	Role        RoleType

	heartbeatTimeout          int64
	electionTimeout           int64
	randomizedElectionTimeout int64

	Log         *RaftLog
	commitIndex int
	lastApplied int

	// used when server's role is leader
	nextIndex  []int
	matchIndex []int

	heartbeatSignal chan int

	Quorum int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	term = rf.CurrentTerm
	if rf.Role == LEADER {
		isleader = true
	}
	rf.mu.Unlock()
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGRanted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	if rf.CurrentTerm > args.Term { // 一定算不上足够“新”
		reply.Term = rf.CurrentTerm
		reply.VoteGRanted = false
		return
	} else if rf.CurrentTerm == args.Term {

	} else {
		// args.Term > rf.CurrentTerm
		rf.mu.Lock()
		rf.CurrentTerm = args.Term
		rf.toFollower()
		rf.mu.Unlock()
	}
	reply.Term = args.Term

	if rf.VatedFor == -1 || rf.VatedFor == args.CandidateId { // 当前节点还未投票 or candidateId
		if args.LastLogTerm > rf.Log.GetLastLogTerm() || (args.LastLogTerm == rf.Log.GetLastLogTerm() && args.LastLogIndex >= rf.Log.GetLastLogIndex()) { // 比较最后一条日至，CANDIDATE的条目需要足够“新”
			rf.mu.Lock()
			rf.VatedFor = args.CandidateId
			rf.mu.Unlock()
			reply.VoteGRanted = true
		} else {
			reply.VoteGRanted = false
		}
	} else {
		reply.VoteGRanted = false // 已经投过票了
	}
}

type AppendEntriesArgs struct {
	Term              int
	LeaderId          int
	PrevLogIndex      int
	PrevLogTerm       int
	Entries           []Entry
	LeaderCommitIndex int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	if args.Term >= rf.CurrentTerm {
		rf.mu.Lock()
		rf.CurrentTerm = args.Term
		rf.heartbeatSignal <- args.LeaderId
		rf.mu.Unlock()
		reply.Success = true
	}
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply, ch chan *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	ch <- reply
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.

func (rf *Raft) newElection(ch chan *RequestVoteReply) {
	rf.CurrentTerm++
	rand.Seed(int64(time.Now().Nanosecond()))
	rf.randomizedElectionTimeout = rf.electionTimeout + rand.Int63n(rf.electionTimeout)

	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			args := &RequestVoteArgs{
				Term:         rf.CurrentTerm,
				CandidateId:  rf.me,
				LastLogIndex: rf.Log.GetLastLogIndex(),
				LastLogTerm:  rf.Log.GetLastLogTerm(),
			}
			reply := &RequestVoteReply{}
			go rf.sendRequestVote(i, args, reply, ch)
		}
	}
}

func (rf *Raft) sendHeartbeat() {
	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			args := &AppendEntriesArgs{
				Term:     rf.CurrentTerm,
				LeaderId: rf.me,
			}
			reply := &AppendEntriesReply{}
			go rf.sendAppendEntries(i, args, reply)
		}
	}
}

func (rf *Raft) toFollower() {
	// candidate to follower
	rf.Role = FOLLOWER
	rf.VatedFor = -1
}
func (rf *Raft) toCandidate() {
	// follower to candidate
	rf.Role = CANDIDATE
	rf.VatedFor = rf.me
}

func (rf *Raft) toLeader() {
	rf.Role = LEADER
	rf.VatedFor = -1
	rf.sendHeartbeat()
}

func (rf *Raft) ticker() {
	for !rf.killed() {

		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().
		if rf.Role == FOLLOWER {
			select {
			case <-time.After(time.Millisecond * time.Duration(rf.randomizedElectionTimeout)):
				// 选举超时，成为candidate
				log.Printf("--term: %v follower id:%v 选举超时,成为candidate--\n", rf.CurrentTerm, rf.me)
				rf.mu.Lock()
				rf.toCandidate()
				rf.mu.Unlock()
			case leaderId := <-rf.heartbeatSignal:
				// 收到有效心跳
				log.Printf("--term: %v follower id:%v 收到心跳消息(leader id:%v),重置选举超时--\n", rf.CurrentTerm, rf.me, leaderId)
				break
			}
		} else if rf.Role == LEADER {
			select {
			case <-time.After(time.Millisecond * time.Duration(rf.heartbeatTimeout)):
				log.Printf("--term: %v leader id:%v定期广播心跳消息--", rf.CurrentTerm, rf.me)
				rf.mu.Lock()
				rf.sendHeartbeat()
				rf.mu.Unlock()
			case leaderId := <-rf.heartbeatSignal:
				log.Printf("--term: %v leader id:%v 收到有效心跳消息(leader id:%v),成为follower--", rf.CurrentTerm, rf.me, leaderId)
				rf.mu.Lock()
				rf.toFollower()
				rf.mu.Unlock()
			}
		} else if rf.Role == CANDIDATE {
			// 开始选举
			ch := make(chan *RequestVoteReply, len(rf.peers)-1)
			rf.mu.Lock()
			log.Printf("--term: %v candidate id:%v 即将发起选举--\n", rf.CurrentTerm, rf.me)
			rf.newElection(ch)
			rf.mu.Unlock()
			votes := 1
			timeout := time.After(time.Millisecond * time.Duration(rf.randomizedElectionTimeout))
		LOOP:
			for {
				select {
				case <-timeout:
					// 选举超时，下一轮选举
					log.Printf("--term: %v candidate id:%v 本次选举超时--\n", rf.CurrentTerm, rf.me)
					break LOOP
				case leaderId := <-rf.heartbeatSignal:
					// 收到有效心跳，乖乖成为FOLLOWER
					log.Printf("--term: %v candidate id:%v 收到有效心跳消息(leader id:%v),成为follower--", rf.CurrentTerm, rf.me, leaderId)
					rf.mu.Lock()
					rf.toFollower()
					rf.mu.Unlock()
					break LOOP
				case reply := <-ch:
					if reply.Term > rf.CurrentTerm { // 已经产生了新的leader
						log.Printf("--term: %v candidate id:%v 请求投票收到更大的term:%v回复消息,成为follower--\n", rf.CurrentTerm, rf.me, rf.CurrentTerm)
						rf.mu.Lock()
						rf.CurrentTerm = reply.Term
						rf.toFollower()
						rf.mu.Unlock()
						break LOOP
					} else if reply.Term == rf.CurrentTerm {
						if reply.VoteGRanted {
							votes++
							log.Printf("--term: %v candidate id:%v 收到投票+1,当前票数为%v--\n", rf.CurrentTerm, rf.me, votes)
							if votes == rf.Quorum {
								// 晋升leader
								log.Printf("--term: %v candidate id:%v 总票数%v达到Quorum,成为leader--\n", rf.CurrentTerm, rf.me, votes)
								rf.mu.Lock()
								rf.toLeader()
								rf.mu.Unlock()
								break LOOP
							}
						} else {
							log.Printf("--term: %v candidate id:%v 收到拒绝投票消息--\n", rf.CurrentTerm, rf.me)
							// 不同意
						}
					} else {
						log.Printf("--term: %v candidate id:%v 收到过期投票消息--\n", rf.CurrentTerm, rf.me)
						// 过期的响应，丢弃
					}
				}
			}
		}
	}
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	rand.Seed(int64(time.Now().Nanosecond()))
	Log := &RaftLog{
		entries: []Entry{
			{
				Term:  0,
				Index: 0,
			},
		},
	}
	// Your initialization code here (2A, 2B, 2C).
	rf.CurrentTerm = 0
	rf.VatedFor = -1
	rf.Role = FOLLOWER

	rf.heartbeatTimeout = 150 // ms
	rf.electionTimeout = 750
	rf.randomizedElectionTimeout = rf.electionTimeout + rand.Int63n(rf.electionTimeout)

	rf.Log = Log
	rf.commitIndex = 0
	rf.lastApplied = 0

	rf.nextIndex = make([]int, 0)
	rf.matchIndex = make([]int, 0)

	rf.heartbeatSignal = make(chan int)

	rf.Quorum = (len(rf.peers) + 1) / 2
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}

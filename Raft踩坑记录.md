# Raft踩坑记录

## 领导者选举

- ticker使用select阻塞在heartbeat、voteMessage、选举超时等事件触发。
- 需要考虑Leader和Candidate收到大于/等于其Term时的`AppendEntrics()`与`RequestVote()`时，需要及时转变为Follower，并在ticker中重置选举时间。
- 向ticker channel发送信号后，需稍等ticker做完收到信号后触发的事件做完。；例如：一个Leader收到了大于其Term的消息或响应，需要及时转变为Follower，而Leader在发送信号给ticker后，可能会执行快于ticker，造成其还未真正变成Follower而继续向下执行了，造成奇怪的表现。
- 在我的实现中，用`nextIndex[rf.me]`代表自身下一个要插入在数组中的`index`。注意`nextIndex[rf.me]`并不能代表`rf.me`的`Log.entries`的实际长度。
- follower收到更大的Term的`RequestVote()`后，不需要重置选举时间，否则可能会导致尽管它可能是唯一正确的Candidate，但其发起选举的时间总是落后于其他follower，白白浪费时间。（在测试过程中，TestRejoin2B和TestBackup2B中会由于超过10秒还未选举出Leader导致程序被中断，继而FAIL）。
- 因为`nextIndex`数组不被持久化，所以恢复后`nextIndex[rf.me]=1`，此时通过`nextIndex[rf.me]`获得`rf.Log.entries`的长度会得到错误的结果。
- C unreliable测试数据比较强，在A和B都通过2000次不出错后仍然跑不过C的unreliable情况。debug后发现原因如下：
  - 注意，leader只能将`commitIndex`更新至具有当前Term的Log。也就是说，一个Leader不能将`commitIndex`更新至Log，该Log具有之前的Term。
  - follower在收到等于其Term的`AppendEntries()` 时，不要重置其投票`VotedFor`状态
  - 在`AppendEntries()`匹配成功后，如果存在新的Entry和follower原有的Log冲突（相同index，不同Term）的情况下，需要从冲突项开始删除其后的所有日志。之后再添加新的Entry。
- Start收到一条客户端的命令后，不要立即`sendHeartbeat()`！
- C的大规模随即数据下，会出现很多并发情况而导致的错讹，务必注意！

[student guide](https://thesquareplanet.com/blog/students-guide-to-raft/#an-aside-on-optimizations)
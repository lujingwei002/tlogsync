# tlogsync
## 主要功能

1. 将日志文件同步到数据库中
2. 根据XML描述文件自动建表，自动新建字段

## 日志文件格式

{服务名字}_tlog_{时间}.log

## xml文件格式

```xml
<?xml version="1.0" encoding="UTF-8"?>
<xml>
    <tlog name="gate_stat" version="1" comment="服务器状态" sharding="month">
        <field name="gameid"        type="int(11)" comment="游戏id"/>
        <field name="onlinenum"     type="int(11)" comment="在线人数"/>
    </tlog>

    <tlog name="user_login" version="1" comment="用户登录" sharding="month">
        <field name="gameid"        type="int(11)"      comment="游戏id"/>
        <field name="userid"        type="bigint(11)"   comment="用户id"/>
        <field name="logintime"     type="int(11)"      comment="登录时间"/>
    </tlog>

    <tlog name="user_login" version="2" comment="用户登录" sharding="month">
        <field name="gameid"        type="int(11)"      comment="游戏id"/>
        <field name="openid"        type="bigint(11)"  comment="openid"/>
        <field name="userid"        type="bigint(11)"   comment="用户id"/>
        <field name="logintime"     type="int(11)"      comment="登录时间"/>
    </tlog>
</xml>
```


<template>
  <div class="login" v-if="!joined">
    <form class="login-form" @submit.prevent="enterRoom">
      <div class="group">
        <input type="text" required v-model="name" />
        <span class="highlight"></span>
        <span class="bar"></span>
        <label>Enter your name</label>
      </div>
      <div class="group">
        <input id="room-input" type="text" v-model="room" placeholder="lobby" />
        <span class="highlight"></span>
        <span class="bar"></span>
        <label for="room-input">Room (optional, default lobby)</label>
      </div>
      <button class="button">ENTER ROOM</button>
    </form>
  </div>

  <div class="chat" v-else>
    <div class="chat-title">
      <figure class="avatar">
        <img src="./assets/imgs/default-icon2.png" alt="Room Avatar" />
      </figure>
      <h1>Chat Room - {{ room }}</h1>
      <h2>{{ name }} ({{ userId }}) · Online: {{ onlineCount }}</h2>
    </div>
    <div class="messages" ref="messages">
      <div class="messages-box" ref="messagesContent">
        <div class="messages-content">
          <!-- 消息列表 -->
          <div 
            v-for="message in messagesArray" 
            :key="`msg_${message.userId}_${message.timestamp}_${message.text.slice(0, 5)}`" 
            :class="message.className"
          >
            <figure class="avatar" v-if="message.className !== 'message message-personal new'">
              <img src="./assets/imgs/default-icon2.png" alt="User Avatar" />
            </figure>
            <span>{{ message.text || '【空消息】' }}</span>
            <div class="timestamp">
              <i>{{ message.name || '未知用户' }}/</i>
              <i> {{ message.timestamp || getTimestamp() }}</i>
            </div>
          </div>
          <!-- 无消息提示 -->
          <div class="no-message" v-if="messagesArray.length === 0">
            暂无消息，开始聊天吧～
          </div>
        </div>
      </div>
    </div>
    <div class="message-box">
      <textarea 
        v-model="newMessage" 
        class="message-input" 
        placeholder="Type message..."
        @input="emitTyping"
        @keydown.enter.prevent="sendMessage"
      ></textarea>
      <!-- <input 
        v-model="toUserId" 
        class="message-to" 
        placeholder="toUserId (optional)"
      /> -->
      <button class="message-submit" @click="sendMessage">
        Send
      </button>
    </div>
  </div>
  <div class="bg"></div>
</template>

<script setup>
import { ref, onUnmounted, watch, nextTick } from "vue";
import BScroll from '@better-scroll/core';
import MouseWheel from '@better-scroll/mouse-wheel';
import Scrollbar from '@better-scroll/scroll-bar';

// 1. 初始化 BetterScroll 插件
BScroll.use(MouseWheel);
BScroll.use(Scrollbar);

// 2. 状态定义
const joined = ref(false);
const name = ref("");
const userId = ref("");
const room = ref("lobby");
const onlineCount = ref(0);
const newMessage = ref("");
const toUserId = ref("");
const messagesArray = ref([]);
const messagesContent = ref(null);
const messages = ref(null);
const bs = ref(null);
let ws = null;
let reconnectTimeout = null;
let typingTimeout = null;

const resolveWsUrl = () => {
  const envUrl = import.meta.env.VITE_WS_URL?.trim();
  if (envUrl) {
    return envUrl.replace(/\/$/, "");
  }
  const { location } = globalThis;
  const protocol = location.protocol === "https:" ? "wss" : "ws";
  if (import.meta.env.DEV) {
    return `${protocol}://${location.hostname}:3001/ws`;
  }
  return `${protocol}://${location.host}/ws`;
};

const wsUrl = resolveWsUrl();

// 3. WebSocket 核心逻辑
const initWebSocket = () => {
  if (ws) ws.close();
  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    console.log('✅ WebSocket 连接成功');
    if (reconnectTimeout) clearTimeout(reconnectTimeout);
  };

  ws.onmessage = (event) => {
    try {
      const res = JSON.parse(event.data);
      handleBackendMessage(res);
    } catch (err) {
      console.error('❌ 解析消息失败：', err);
    }
  };

  ws.onerror = (err) => {
    console.error('❌ WebSocket 错误：', err);
    handleReconnect();
  };

  ws.onclose = (event) => {
    console.log('🔌 WebSocket 断开：', event.reason);
    if (!joined.value) handleReconnect();
  };
};

// 处理后端消息
const handleBackendMessage = (res) => {
  if (!res.type) return;
  switch (res.type) {
    case 'joinSuccess':
      joined.value = true;
      userId.value = res.data.userId;
      room.value = res.data.room || room.value;
      onlineCount.value = res.data.onlineCount ?? onlineCount.value;
      break;
    case 'historyMessages':
      formatHistoryMessages(res.data);
      break;
    case 'newMessage':
      addNewMessage(res.data);
      break;
    case 'privateMessage':
      addPrivateMessage(res.data);
      break;
    case 'typingStatus':
      handleTypingStatus(res.data);
      break;
    case 'msgFail':
      alert(`❌ 消息发送失败：${res.data.msg}`);
      break;
    case 'joinFail':
      alert(`❌ 加入失败：${res.data.msg || '未知原因'}`);
      break;
    case 'historyFail':
      alert(`❌ 拉取历史失败：${res.data.msg || '未知原因'}`);
      break;
    case 'typingFail':
      console.warn('⚠️ 打字状态上报失败：', res.data?.msg);
      break;
    case 'userJoin': {
      const ev = res.data;
      onlineCount.value = ev.onlineCount ?? onlineCount.value + 1;
      addSystemMessage(`${ev.name || ev.userId} 加入了房间`);
      break;
    }
    case 'userLeave': {
      const ev = res.data;
      onlineCount.value = Math.max(0, onlineCount.value - 1);
      addSystemMessage(`${ev.name || ev.userId} 离开了房间`);
      break;
    }
    default:
      console.warn(`⚠️ 未知消息类型：${res.type}`);
  }
};

// 重连逻辑
const handleReconnect = () => {
  if (reconnectTimeout) return;
  reconnectTimeout = setTimeout(initWebSocket, 5000);
};

// 发送消息封装
const sendWsMessage = (msg) => {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    alert('网络未连接，请稍候重试');
    return;
  }
  ws.send(JSON.stringify(msg));
};

// 4. 业务功能
// 登录
const enterRoom = () => {
  const trimName = name.value.trim();
  if (!trimName) {
    alert('请输入昵称');
    return;
  }
  const rm = (room.value || 'lobby').trim() || 'lobby';
  sendWsMessage({ type: 'join', name: trimName, data: { room: rm } });
  sendWsMessage({ type: 'findAllMessages' });
};

// 正在输入
const emitTyping = () => {
  const isTyping = newMessage.value.trim() !== '';
  sendWsMessage({
    type: 'typing',
    data: { isTyping, name: name.value.trim(), userId: userId.value }
  });
  if (typingTimeout) clearTimeout(typingTimeout);
  typingTimeout = setTimeout(() => {
    if (newMessage.value.trim() === '') {
      sendWsMessage({
        type: 'typing',
        data: { isTyping: false, name: name.value.trim(), userId: userId.value }
      });
    }
  }, 1000);
};

// 发送消息（前端预添加）
const sendMessage = () => {
  const trimMsg = newMessage.value.trim();
  if (!trimMsg) return;
  // 前端预添加消息
  const tempMsg = {
    text: trimMsg,
    name: name.value.trim(),
    userId: userId.value,
    timestamp: getTimestamp(),
    className: 'message message-personal new'
  };
  messagesArray.value.push(tempMsg);
  updateScroll();
  // 发送给后端
  const payload = { ...tempMsg };
  if (toUserId.value.trim()) {
    payload.toUserId = toUserId.value.trim();
  }
  sendWsMessage({ type: 'createMessage', data: payload });
  // 清空输入
  newMessage.value = '';
  sendWsMessage({
    type: 'typing',
    data: { isTyping: false, name: name.value.trim(), userId: userId.value }
  });
};

// 时间戳
const getTimestamp = () => {
  const d = new Date();
  const hours = String(d.getHours()).padStart(2, '0');
  const minutes = String(d.getMinutes()).padStart(2, '0');
  return `${hours}:${minutes}`;
};

// 格式化历史消息
const formatHistoryMessages = (history) => {
  const formatted = history
    .filter(msg => msg.text && msg.userId && msg.timestamp)
    .map(msg => ({
      ...msg,
      className: msg.userId === userId.value 
        ? 'message message-personal new' 
        : 'message new',
      name: msg.name || '历史用户'
    }));
  messagesArray.value = formatted;
  initScroll();
};

// 添加新消息
const addNewMessage = (msg) => {
  if (!msg.text || !msg.userId || !msg.timestamp) return;
  const isDuplicate = messagesArray.value.some(item => 
    item.userId === msg.userId && item.timestamp === msg.timestamp && item.text === msg.text
  );
  if (isDuplicate) return;
  const newMsg = {
    ...msg,
    className: msg.userId === userId.value 
      ? 'message message-personal new' 
      : 'message new',
    name: msg.name || '未知用户'
  };
  messagesArray.value.push(newMsg);
  updateScroll();
};

// 添加私聊消息（标记）
const addPrivateMessage = (msg) => {
  if (!msg || !msg.text || !msg.userId || !msg.timestamp) return;
  const newMsg = {
    ...msg,
    text: `[私聊] ${msg.text}`,
    className: msg.userId === userId.value 
      ? 'message message-personal new' 
      : 'message new',
    name: msg.name || '未知用户'
  };
  messagesArray.value.push(newMsg);
  updateScroll();
};

// 系统消息
const addSystemMessage = (text) => {
  const sysMsg = {
    userId: 'system',
    text,
    name: '系统',
    timestamp: getTimestamp(),
    className: 'message system new'
  };
  messagesArray.value.push(sysMsg);
  updateScroll();
};

// 处理正在输入提示
const handleTypingStatus = (data) => {
  if (data.userId === userId.value) return;
  const idx = messagesArray.value.findIndex(item => 
    item.text === '' && item.userId === data.userId
  );
  if (data.isTyping && idx === -1) {
    messagesArray.value.push({
      userId: data.userId,
      text: '',
      name: data.name || '未知用户',
      timestamp: getTimestamp(),
      className: 'message loading new'
    });
  } else if (!data.isTyping && idx !== -1) {
    messagesArray.value.splice(idx, 1);
  }
  updateScroll();
};

// 5. 滚动逻辑
const initializeScroll = () => {
  nextTick(() => {
    if (messagesContent.value) {
      if (bs.value) bs.value.destroy();
      bs.value = new BScroll(messagesContent.value, {
        scrollY: true,
        mouseWheel: { speed: 6, invert: false, easeTime: 800 },
        scrollbar: { interactive: true, fade: true },
        click: true,
        bounce: false
      });
      // 滚动条显隐
      const content = messagesContent.value;
      content.addEventListener('mouseenter', () => {
        const bar = content.querySelector('.bscroll-vertical-scrollbar');
        if (bar) bar.style.opacity = '1';
      });
      content.addEventListener('mouseleave', () => {
        const bar = content.querySelector('.bscroll-vertical-scrollbar');
        if (bar) bar.style.opacity = '0';
      });
    }
  });
};

const initScroll = () => {
  nextTick(() => {
    if (bs.value) {
      bs.value.refresh();
      bs.value.scrollTo(0, 0, 100);
    }
  });
};

const updateScroll = () => {
  nextTick(() => {
    if (bs.value) {
      bs.value.refresh();
      const maxY = bs.value.maxScrollY;
      if (maxY >= 0) bs.value.scrollTo(0, maxY + 10, 100);
    }
  });
};

// 6. 生命周期
nextTick(initWebSocket);
watch(joined, (isJoined) => isJoined && initializeScroll(), { immediate: true });

onUnmounted(() => {
  if (ws) ws.close(1000, '组件卸载');
  if (reconnectTimeout) clearTimeout(reconnectTimeout);
  if (typingTimeout) clearTimeout(typingTimeout);
  if (bs.value) bs.value.destroy();
  const content = messagesContent.value;
  if (content) {
    content.removeEventListener('mouseenter', () => {});
    content.removeEventListener('mouseleave', () => {});
  }
});
</script>

<style scoped lang="scss" src="./style.scss"></style>
// static/app.js
class ChatClient {
    constructor() {
        this.ws = null;
        this.isPrivate = false;
        this.targetUser = '';
        this.selfName = '';
        this.onlineUsers = [];
        this.onlineSet = new Set();
        this.expectingWho = false;
        this.whoTimer = null;
        this.pendingOwn = [];
        this.connect();
        this.bind();
    }

    requestWho() {
        if (this.ws && this.ws.readyState === 1) {
            console.log('[DEBUG] 请求在线用户列表');
            
            // 清理之前的状态
            if (this.whoTimer) {
                clearTimeout(this.whoTimer);
                this.whoTimer = null;
            }
            
            this.expectingWho = true;
            this.onlineSet.clear();
            
            // 如果知道自己的用户名，先加入集合
            if (this.selfName) {
                this.onlineSet.add(this.selfName);
                console.log('[DEBUG] 预先添加自己:', this.selfName);
            }
            
            this.ws.send('who');
            console.log('[DEBUG] 已发送who命令');
            
            // 设置超时
            this.whoTimer = setTimeout(() => {
                console.log('[DEBUG] who收集超时，强制完成');
                this.finishWhoCollect();
            }, 2000); // 增加超时时间到2秒
        } else {
            console.log('[DEBUG] WebSocket未连接，无法请求who');
        }
    }

    finishWhoCollect() {
        this.expectingWho = false;
        
        // 确保自己在列表中（防止who返回不完整）
        if (this.selfName && !this.onlineSet.has(this.selfName)) {
            this.onlineSet.add(this.selfName);
        }
        
        this.onlineUsers = Array.from(this.onlineSet);
        
        console.log('[DEBUG] 收集完成，在线用户:', this.onlineUsers);
        console.log('[DEBUG] 当前用户名:', this.selfName);
        
        // 排序：自己置顶
        this.onlineUsers.sort((a, b) => {
            if (a === this.selfName) return -1;
            if (b === this.selfName) return 1;
            return a.localeCompare(b, 'zh-CN');
        });
        
        // 检查私聊目标是否还在线
        if (this.targetUser && !this.onlineUsers.includes(this.targetUser)) {
            this.targetUser = '';
            this.updateTargetLabel();
            if (this.isPrivate) {
                this.pushSys(`私聊对象已离线，请重新选择`);
            }
        }
        
        this.renderUsers();
        this.updateUserHighlight();
    }

    connect() {
        const url = `ws://${location.host}/ws`;
        this.ws = new WebSocket(url);
        this.ws.onopen = () => {
            this.setStatus(true);
            this.pushSys('连接成功');
        };
        this.ws.onclose = () => {
            this.setStatus(false);
            this.pushSys('连接断开，3秒后重连');
            // 清空在线列表
            this.onlineUsers = [];
            this.onlineSet.clear();
            this.renderUsers();
            setTimeout(() => this.connect(), 3000);
        };
        this.ws.onmessage = e => this.handleMsg(e.data);
        this.ws.onerror = () => this.setStatus(false);
    }

    bind() {
        document.getElementById('renameBtn').onclick = () => {
            const v = document.getElementById('nameInput').value.trim();
            if (v) {
                if (v === this.selfName) {
                    this.pushSys('用户名未改变');
                    return;
                }
                this.ws.send('rename|' + v);
                document.getElementById('nameInput').value = '';
            }
        };
        
        // 回车发送重命名
        document.getElementById('nameInput').addEventListener('keydown', e => {
            if (e.key === 'Enter') {
                document.getElementById('renameBtn').click();
            }
        });
        
        document.getElementById('sendBtn').onclick = () => this.sendCurrent();
        document.getElementById('msgInput').addEventListener('keydown', e => {
            if (e.key === 'Enter') this.sendCurrent();
        });
        
        document.getElementById('publicBtn').onclick = () => this.switchPublic();
        document.getElementById('privateBtn').onclick = () => this.switchPrivate();
        document.getElementById('refreshUsersBtn').onclick = () => {
            this.pushSys('正在刷新在线用户列表...');
            this.requestWho();
        };
        
        // 用户列表点击事件
        document.getElementById('users').onclick = (e) => {
            let item = e.target.closest('.user-item');
            if (!item) return;
            
            const name = item.dataset.name;
            
            if (!this.isPrivate) {
                // 公聊模式下点击用户自动切换到私聊
                this.switchPrivate();
                if (name !== this.selfName) {
                    this.targetUser = name;
                    this.updateUserHighlight();
                    this.updateTargetLabel();
                    this.pushSys(`已选择与 ${name} 私聊`);
                }
                return;
            }
            
            // 私聊模式
            if (name === this.selfName) {
                this.pushSys('不能与自己私聊');
                return;
            }
            
            this.targetUser = name;
            this.updateUserHighlight();
            this.updateTargetLabel();
            this.pushSys(`已选择与 ${name} 私聊`);
        };
    }

    setStatus(ok) {
        const el = document.getElementById('status');
        el.textContent = ok ? '已连接' : '断开';
        el.className = 'badge ' + (ok ? 'ok' : 'no');
    }

    sendCurrent() {
        const inp = document.getElementById('msgInput');
        const txt = inp.value.trim();
        if (!txt) return;

        if (this.isPrivate) {
            if (!this.targetUser) {
                this.pushSys('请先选择私聊对象');
                return;
            }
            if (!this.onlineUsers.includes(this.targetUser)) {
                this.pushSys('私聊对象不存在，正在刷新列表...');
                this.requestWho();
                return;
            }
            this.sendRaw(`to|${this.targetUser}|${txt}`);
            this.pushMsg(`-> ${this.targetUser}: ${txt}`, true, true);
        } else {
            this.pushMsg(txt, true, false, this.selfName || '我');
            this.pendingOwn.push({ content: txt, t: Date.now() });
            this.sendRaw(txt);
        }
        inp.value = '';
    }

    sendRaw(s) {
        if (this.ws && this.ws.readyState === 1) this.ws.send(s);
    }

    handleMsg(m) {
        m = m.replace(/[\r\n]+/g, '').trim();
        if (!m) return;
        
        console.log('[DEBUG] 收到原始消息:', JSON.stringify(m));

        // 分配用户名
        if (m.startsWith('您已分配用户名:')) {
            this.selfName = m.split(':')[1].trim();
            this.pushSys('系统分配用户名: ' + this.selfName);
            // 立即将自己加入列表并显示
            this.onlineSet.add(this.selfName);
            this.onlineUsers = [this.selfName];
            this.renderUsers();
            // 然后请求完整列表
            setTimeout(() => this.requestWho(), 200);
            return;
        }

        // 更新用户名
        if (m.startsWith('您已更新用户名:')) {
            const oldName = this.selfName;
            this.selfName = m.split(':')[1].trim();
            this.pushSys(`用户名已更新: ${oldName} -> ${this.selfName}`);
            // 立即更新本地列表
            if (oldName && this.onlineUsers.includes(oldName)) {
                const index = this.onlineUsers.indexOf(oldName);
                this.onlineUsers[index] = this.selfName;
                this.renderUsers();
            }
            setTimeout(() => this.requestWho(), 200);
            return;
        }

        // 其他人改名广播
        if (m.includes('改名为:')) {
            this.pushSys('有用户更改了昵称，正在刷新列表...');
            setTimeout(() => this.requestWho(), 100);
            return;
        }

        // 上下线通知
        if (m.includes('已上线') || m.includes('已下线')) {
            this.pushSys(m);
            setTimeout(() => this.requestWho(), 200);
            return;
        }

        // 解析who返回的在线用户信息 - 使用更宽松的匹配
        if (this.expectingWho && (m.includes('在线') || m.includes('online'))) {
            console.log('[DEBUG] 尝试解析在线用户消息:', JSON.stringify(m));
            
            // 尝试多种格式匹配
            let userName = null;
            
            // 格式1: [IP]用户名:在线...
            let match = m.match(/\[([^\]]*)\]([^:]+):在线/);
            if (match) {
                userName = match[2].trim();
                console.log('[DEBUG] 格式1匹配成功:', userName);
            } else {
                // 格式2: 用户名:在线...
                match = m.match(/^([^:\[]+):在线/);
                if (match) {
                    userName = match[1].trim();
                    console.log('[DEBUG] 格式2匹配成功:', userName);
                } else {
                    // 格式3: 更宽松的匹配 - 包含"在线"的任何消息
                    match = m.match(/([^\[\]:]+)(?::|\s+)在线/);
                    if (match) {
                        userName = match[1].trim();
                        console.log('[DEBUG] 格式3匹配成功:', userName);
                    }
                }
            }
            
            if (userName && userName.length > 0) {
                console.log('[DEBUG] 添加在线用户:', JSON.stringify(userName));
                this.onlineSet.add(userName);
                
                // 重置计时器
                if (this.whoTimer) clearTimeout(this.whoTimer);
                this.whoTimer = setTimeout(() => {
                    this.finishWhoCollect();
                }, 500); // 增加等待时间
                return;
            } else {
                console.log('[DEBUG] 无法解析用户名，原始消息:', JSON.stringify(m));
            }
        }

        // 私聊消息
        if (m.includes('对您说:')) {
            const match = m.match(/^(.+?)对您说:(.+)$/);
            if (match) {
                const fromUser = match[1];
                const content = match[2];
                this.pushMsg(`${fromUser}: ${content}`, false, true);
            } else {
                this.pushMsg(m, false, true);
            }
            return;
        }

        // 公聊消息 - 格式: [IP]用户名:消息内容
        if (m.startsWith('[') && m.indexOf(']:') !== -1) {
            const cut = m.indexOf(']');
            const rest = m.slice(cut + 1);
            const nameEnd = rest.indexOf(':');
            let name = rest.slice(0, nameEnd).trim();
            const content = rest.slice(nameEnd + 1);
            
            // 去重自己的消息（本地已显示）
            if (name === this.selfName && this.selfName !== '') {
                this.pendingOwn = this.pendingOwn.filter(p => {
                    if (p.matched) return false;
                    if (p.content === content && (Date.now() - p.t) < 8000) {
                        p.matched = true;
                        return false;
                    }
                    return true;
                });
                return;
            }
            
            const own = (name === this.selfName && this.selfName !== '');
            this.pushMsg(content, own, false, name);
            return;
        }

        // 错误消息或系统消息
        this.pushSys(m);
    }

    pushSys(t) {
        this.appendDom(t, 'sys');
    }

    pushMsg(content, own = false, isPrivate = false, fromName = '') {
        let label = '';
        if (isPrivate) {
            label = own ? '' : '';  // 私聊消息已在content中包含发送者
        } else {
            label = (fromName && !own) ? `${fromName}: ` : '';
        }
        this.appendDom(label + content, own ? 'own msg' : (isPrivate ? 'private msg' : 'msg'));
    }

    appendDom(text, cls) {
        const box = document.getElementById('messages');
        const div = document.createElement('div');
        div.className = cls.includes('msg') ? cls : 'msg ' + cls;
        if (cls.includes('sys')) div.className = 'msg sys';
        div.textContent = text;
        div.title = new Date().toLocaleTimeString(); // 添加时间提示
        box.appendChild(div);
        box.scrollTop = box.scrollHeight;
        
        // 限制消息数量，防止内存溢出
        while (box.children.length > 500) {
            box.removeChild(box.firstChild);
        }
    }

    renderUsers() {
        const wrap = document.getElementById('users');
        wrap.innerHTML = '';
        
        console.log('[DEBUG] 渲染用户列表:', this.onlineUsers);
        
        this.onlineUsers.forEach(n => {
            const d = document.createElement('div');
            d.className = 'user-item';
            d.dataset.name = n;
            d.textContent = (n === this.selfName) ? n + ' (我)' : n;
            d.title = `点击${n === this.selfName ? '查看' : '与 ' + n + ' 私聊'}`;
            if (this.isPrivate && this.targetUser === n) d.classList.add('active');
            wrap.appendChild(d);
        });
        
        document.getElementById('userCount').textContent = this.onlineUsers.length;
    }

    updateUserHighlight() {
        document.querySelectorAll('.user-item').forEach(el => {
            el.classList.toggle('active', el.dataset.name === this.targetUser && this.isPrivate);
        });
    }

    switchPublic() {
        this.isPrivate = false;
        this.targetUser = '';
        document.getElementById('modeLabel').textContent = '公共聊天室';
        document.getElementById('currentTarget').style.display = 'none';
        document.getElementById('publicBtn').style.backgroundColor = '#007bff';
        document.getElementById('publicBtn').style.color = 'white';
        document.getElementById('privateBtn').style.backgroundColor = '';
        document.getElementById('privateBtn').style.color = '';
        this.updateUserHighlight();
    }

    switchPrivate() {
        this.isPrivate = true;
        document.getElementById('modeLabel').textContent = '私聊模式';
        document.getElementById('privateBtn').style.backgroundColor = '#007bff';
        document.getElementById('privateBtn').style.color = 'white';
        document.getElementById('publicBtn').style.backgroundColor = '';
        document.getElementById('publicBtn').style.color = '';
        this.updateTargetLabel();
        // 刷新用户列表确保最新
        this.requestWho();
    }

    updateTargetLabel() {
        const el = document.getElementById('currentTarget');
        if (this.isPrivate) {
            el.style.display = 'inline';
            if (this.targetUser) {
                el.textContent = '私聊对象: ' + this.targetUser;
                el.style.color = '#007bff';
            } else {
                el.textContent = '请点击左侧用户选择私聊对象';
                el.style.color = '#dc3545';
            }
        } else {
            el.style.display = 'none';
        }
    }
}

document.addEventListener('DOMContentLoaded', () => new ChatClient());
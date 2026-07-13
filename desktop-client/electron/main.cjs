const fs = require('node:fs');
const path = require('node:path');
const { app, BrowserWindow, dialog, ipcMain, powerSaveBlocker, shell } = require('electron');
const { downloadSong } = require('./downloader.cjs');
const {
  checkQrLogin,
  getLyric,
  getPlaylist,
  getProfile,
  getSongUrl,
  searchSongs,
  startQrLogin
} = require('./netease.cjs');

let mainWindow;
let settingsPath;
let queuePath;
let powerSaveBlockerId;
let exitConfirmed = false;
let playbackActive = false;
let queue = [];
let activeTaskId = '';
let activeControl = null;
const logs = [];

function defaultSettings() {
  const musicDir = app.getPath('music');
  return {
    sourceDir: path.join(musicDir, '163MUSIC', 'Source'),
    mp3Dir: path.join(musicDir, '163MUSIC', 'MP3'),
    musicU: '',
    mergeMetadata: true,
    preventSleep: false,
    savedPlaylists: []
  };
}

function readSettings() {
  try {
    return { ...defaultSettings(), ...JSON.parse(fs.readFileSync(settingsPath, 'utf8')) };
  } catch {
    return defaultSettings();
  }
}

function writeSettings(settings) {
  const next = { ...readSettings(), ...settings };
  fs.mkdirSync(path.dirname(settingsPath), { recursive: true });
  fs.writeFileSync(settingsPath, JSON.stringify(next, null, 2));
  return next;
}

function applyPreventSleep(enabled) {
  if (enabled && powerSaveBlockerId === undefined) {
    powerSaveBlockerId = powerSaveBlocker.start('prevent-app-suspension');
    addLog('已开启忽略系统休眠');
  }
  if (!enabled && powerSaveBlockerId !== undefined) {
    powerSaveBlocker.stop(powerSaveBlockerId);
    powerSaveBlockerId = undefined;
    addLog('已关闭忽略系统休眠');
  }
}

function addLog(message) {
  const entry = {
    time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
    message
  };
  logs.push(entry);
  if (logs.length > 600) logs.shift();
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('log-entry', entry);
  }
}

function unfinishedTasks() {
  return queue.filter((task) => !['done', 'stopped'].includes(task.state));
}

function persistQueue() {
  if (!queuePath) return;
  fs.mkdirSync(path.dirname(queuePath), { recursive: true });
  fs.writeFileSync(queuePath, JSON.stringify(queue, null, 2));
}

function taskPayload(task) {
  return {
    taskId: task.taskId,
    songName: task.song.name,
    state: task.state,
    percent: task.percent || 0,
    error: task.error || '',
    createdAt: task.createdAt,
    batchId: task.batchId,
    batchIndex: task.batchIndex,
    batchTotal: task.batchTotal
  };
}

function broadcastTask(task) {
  if (mainWindow && !mainWindow.isDestroyed()) mainWindow.webContents.send('download-progress', taskPayload(task));
}

function updateTask(task, patch) {
  Object.assign(task, patch);
  persistQueue();
  broadcastTask(task);
}

function pauseAllTasks() {
  queue.forEach((task) => {
    if (!['done', 'stopped', 'paused'].includes(task.state)) task.state = 'paused';
  });
  if (activeControl) {
    activeControl.reason = 'paused';
    activeControl.cancel?.();
  }
  persistQueue();
}

function closeRequestPayload() {
  return {
    activeDownloads: unfinishedTasks().length,
    playbackActive
  };
}

async function processQueue() {
  if (activeTaskId) return;
  const task = queue.find((item) => item.state === 'queued');
  if (!task) return;

  activeTaskId = task.taskId;
  activeControl = { reason: '', cancel: null };
  updateTask(task, { state: 'downloading', error: '' });
  try {
    await downloadSong({
      song: task.song,
      format: task.format,
      settings: readSettings(),
      webContents: {
        send: (_channel, payload) => updateTask(task, {
          state: payload.state || task.state,
          percent: payload.percent ?? task.percent,
          error: payload.error || '',
          path: payload.path || task.path || ''
        })
      },
      taskId: task.taskId,
      logFn: addLog,
      control: activeControl
    });
    updateTask(task, { state: 'done', percent: 100 });
  } catch (error) {
    if (error.code === 'PAUSED') updateTask(task, { state: 'paused' });
    else if (error.code === 'STOPPED') {
      queue = queue.filter((item) => item.taskId !== task.taskId);
      persistQueue();
    } else updateTask(task, { state: 'failed', error: error.message || String(error) });
  } finally {
    activeTaskId = '';
    activeControl = null;
    processQueue();
  }
}

function loadQueue() {
  try {
    queue = JSON.parse(fs.readFileSync(queuePath, 'utf8'));
  } catch {
    queue = [];
  }
  queue = queue.map((task) => ['downloading', 'queued'].includes(task.state) ? { ...task, state: 'paused' } : task);
  persistQueue();
}

function createWindow() {
  const windowOptions = {
    width: 1320,
    height: 860,
    minWidth: 1080,
    minHeight: 720,
    title: '163MUSIC',
    icon: path.join(__dirname, '..', 'assets', 'icon.png'),
    backgroundColor: '#f7f8fb',
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      contextIsolation: true,
      nodeIntegration: false
    }
  };
  if (process.platform === 'darwin') windowOptions.titleBarStyle = 'hiddenInset';
  mainWindow = new BrowserWindow(windowOptions);

  if (process.env.VITE_DEV_SERVER_URL) {
    mainWindow.loadURL(process.env.VITE_DEV_SERVER_URL);
  } else {
    mainWindow.loadFile(path.join(__dirname, '..', 'dist', 'index.html'));
  }

  mainWindow.on('close', (event) => {
    if ((unfinishedTasks().length > 0 || playbackActive) && !exitConfirmed) {
      event.preventDefault();
      mainWindow.webContents.send('app-close-requested', closeRequestPayload());
    }
  });
}

app.whenReady().then(() => {
  if (process.platform === 'win32') app.setAppUserModelId('top.91cz.163music');
  settingsPath = path.join(app.getPath('userData'), 'settings.json');
  queuePath = path.join(app.getPath('userData'), 'download-queue.json');
  const settings = writeSettings(readSettings());
  applyPreventSleep(Boolean(settings.preventSleep));
  loadQueue();
  createWindow();
  addLog('163MUSIC 已启动');

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});

app.on('before-quit', (event) => {
  if ((unfinishedTasks().length > 0 || playbackActive) && !exitConfirmed) {
    event.preventDefault();
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.show();
      mainWindow.webContents.send('app-close-requested', closeRequestPayload());
    }
  }
});

ipcMain.handle('settings:get', () => ({ ...readSettings(), logs }));

ipcMain.handle('settings:save', (_event, settings) => {
  const next = writeSettings(settings);
  applyPreventSleep(Boolean(next.preventSleep));
  addLog('设置已保存');
  return next;
});

ipcMain.handle('dialog:choose-directory', async () => {
  const result = await dialog.showOpenDialog(mainWindow, {
    properties: ['openDirectory', 'createDirectory']
  });
  return result.canceled ? '' : result.filePaths[0];
});

ipcMain.handle('netease:profile', async () => getProfile(readSettings()));

ipcMain.handle('app:playback-state', (_event, active) => {
  playbackActive = Boolean(active);
});

ipcMain.handle('queue:get', () => queue.map(taskPayload));

ipcMain.handle('queue:enqueue', (_event, songs, format) => {
  const batchId = `batch-${Date.now()}`;
  const list = Array.isArray(songs) ? songs : [];
  const created = list.map((song, index) => ({
    taskId: `${song.id}-${Date.now()}-${index}`,
    song,
    format,
    state: 'queued',
    percent: 0,
    error: '',
    createdAt: Date.now() + index,
    batchId,
    batchIndex: index + 1,
    batchTotal: list.length
  }));
  queue.push(...created);
  persistQueue();
  created.forEach(broadcastTask);
  processQueue();
  return { batchId, total: created.length };
});

ipcMain.handle('queue:pause', (_event, taskId) => {
  const task = queue.find((item) => item.taskId === taskId);
  if (!task) return;
  if (task.taskId === activeTaskId && activeControl) {
    updateTask(task, { state: 'paused' });
    activeControl.reason = 'paused';
    activeControl.cancel?.();
  } else if (task.state === 'queued' || task.state === 'failed') updateTask(task, { state: 'paused' });
});

ipcMain.handle('queue:resume', (_event, taskId) => {
  const task = queue.find((item) => item.taskId === taskId);
  if (!task || ['done', 'stopped'].includes(task.state)) return;
  updateTask(task, { state: 'queued', error: '' });
  processQueue();
});

ipcMain.handle('queue:stop', (_event, taskId) => {
  const task = queue.find((item) => item.taskId === taskId);
  if (!task) return;
  if (task.taskId === activeTaskId && activeControl) {
    activeControl.reason = 'stopped';
    activeControl.cancel?.();
  } else {
    queue = queue.filter((item) => item.taskId !== taskId);
    persistQueue();
  }
});

ipcMain.handle('auth:qr-start', async () => startQrLogin());

ipcMain.handle('auth:qr-check', async (_event, key) => checkQrLogin(key));

ipcMain.handle('netease:search', async (_event, keyword) => {
  if (!String(keyword || '').trim()) return [];
  addLog(`搜索单曲：${keyword}`);
  return searchSongs(keyword, readSettings());
});

ipcMain.handle('netease:playlist', async (_event, input) => {
  addLog(`读取歌单：${input}`);
  return getPlaylist(input, readSettings());
});

ipcMain.handle('netease:preview', async (_event, song) => {
  const settings = readSettings();
  const [stream, lyric] = await Promise.all([
    getSongUrl(song.id, settings),
    getLyric(song.id, settings).catch(() => ({ lyric: '', translated: '' }))
  ]);
  addLog(`试听：${song.name}`);
  return { ...stream, lyric: lyric.lyric, translated: lyric.translated };
});

ipcMain.handle('app:close-response', (_event, response) => {
  if (response === 'minimize') {
    mainWindow?.minimize();
    return;
  }
  if (response !== 'exit') return;
  exitConfirmed = true;
  if (unfinishedTasks().length > 0) pauseAllTasks();
  app.quit();
  setTimeout(() => app.exit(0), 200);
});

ipcMain.handle('shell:open-path', async (_event, targetPath) => {
  if (!targetPath) return;
  shell.showItemInFolder(targetPath);
});

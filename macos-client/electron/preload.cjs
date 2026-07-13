const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('music163', {
  getSettings: () => ipcRenderer.invoke('settings:get'),
  saveSettings: (settings) => ipcRenderer.invoke('settings:save', settings),
  chooseDirectory: () => ipcRenderer.invoke('dialog:choose-directory'),
  getProfile: () => ipcRenderer.invoke('netease:profile'),
  startQrLogin: () => ipcRenderer.invoke('auth:qr-start'),
  checkQrLogin: (key) => ipcRenderer.invoke('auth:qr-check', key),
  searchSongs: (keyword) => ipcRenderer.invoke('netease:search', keyword),
  getPlaylist: (input) => ipcRenderer.invoke('netease:playlist', input),
  getPreview: (song) => ipcRenderer.invoke('netease:preview', song),
  getQueue: () => ipcRenderer.invoke('queue:get'),
  enqueueDownloads: (songs, format) => ipcRenderer.invoke('queue:enqueue', songs, format),
  pauseDownload: (taskId) => ipcRenderer.invoke('queue:pause', taskId),
  resumeDownload: (taskId) => ipcRenderer.invoke('queue:resume', taskId),
  stopDownload: (taskId) => ipcRenderer.invoke('queue:stop', taskId),
  setPlaybackActive: (active) => ipcRenderer.invoke('app:playback-state', active),
  openPath: (path) => ipcRenderer.invoke('shell:open-path', path),
  onProgress: (handler) => {
    const listener = (_event, payload) => handler(payload);
    ipcRenderer.on('download-progress', listener);
    return () => ipcRenderer.removeListener('download-progress', listener);
  },
  onLog: (handler) => {
    const listener = (_event, payload) => handler(payload);
    ipcRenderer.on('log-entry', listener);
    return () => ipcRenderer.removeListener('log-entry', listener);
  },
  respondToClose: (response) => ipcRenderer.invoke('app:close-response', response),
  onCloseRequested: (handler) => {
    const listener = (_event, payload) => handler(payload);
    ipcRenderer.on('app-close-requested', listener);
    return () => ipcRenderer.removeListener('app-close-requested', listener);
  }
});

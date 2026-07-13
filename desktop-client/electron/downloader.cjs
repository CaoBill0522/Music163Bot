const fs = require('node:fs');
const path = require('node:path');
const http = require('node:http');
const https = require('node:https');
const { spawn } = require('node:child_process');
const ffmpegPath = require('ffmpeg-static');
const NodeID3 = require('node-id3');
const { cookieFromSettings, getLyric, getSongDetail, getSongUrl } = require('./netease.cjs');

function sanitizeName(value) {
  return String(value || 'unknown')
    .replace(/[<>:"/\\|?*\x00-\x1f]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 160);
}

function extensionFromUrl(url, fallback = 'mp3') {
  try {
    const ext = path.extname(new URL(url).pathname).replace('.', '').toLowerCase();
    if (ext && ext.length <= 5) return ext;
  } catch {}
  return fallback;
}

function ensureDir(dir) {
  fs.mkdirSync(dir, { recursive: true });
}

function sendProgress(webContents, payload) {
  webContents.send('download-progress', payload);
}

function log(logFn, message) {
  if (typeof logFn === 'function') logFn(message);
}

function cancelError(reason) {
  const error = new Error(reason === 'stopped' ? '任务已停止' : '下载已暂停');
  error.code = reason === 'stopped' ? 'STOPPED' : 'PAUSED';
  return error;
}

function downloadFile({ url, dest, settings, songName, webContents, taskId, logFn, control, redirectCount = 0 }) {
  return new Promise((resolve, reject) => {
    if (redirectCount > 5) {
      reject(new Error('下载重定向过多'));
      return;
    }

    if (control?.reason) return reject(cancelError(control.reason));
    const client = url.startsWith('https:') ? https : http;
    const request = client.get(url, {
      headers: {
        Cookie: cookieFromSettings(settings),
        Referer: 'https://music.163.com/',
        'User-Agent': `Mozilla/5.0 163MUSIC Desktop (${process.platform})`
      }
    }, (response) => {
      if ([301, 302, 303, 307, 308].includes(response.statusCode)) {
        const location = response.headers.location;
        if (!location) {
          reject(new Error('下载重定向缺少 Location'));
          return;
        }
        const nextUrl = new URL(location, url).toString();
        resolve(downloadFile({ url: nextUrl, dest, settings, songName, webContents, taskId, logFn, control, redirectCount: redirectCount + 1 }));
        return;
      }

      if (response.statusCode < 200 || response.statusCode >= 300) {
        reject(new Error(`下载失败：HTTP ${response.statusCode}`));
        return;
      }

      const total = Number(response.headers['content-length'] || 0);
      let transferred = 0;
      const file = fs.createWriteStream(dest);
      const cancel = () => {
        const error = cancelError(control?.reason);
        response.destroy(error);
        file.destroy(error);
        request.destroy(error);
      };
      if (control) control.cancel = cancel;
      response.on('data', (chunk) => {
        transferred += chunk.length;
        const percent = total ? Math.round((transferred / total) * 100) : 0;
        sendProgress(webContents, {
          taskId,
          songName,
          state: 'downloading',
          percent,
          transferred,
          total
        });
      });
      response.pipe(file);
      file.on('finish', () => {
        if (control) control.cancel = null;
        file.close(() => resolve(dest));
      });
      file.on('error', reject);
    });

    request.on('error', reject);
    request.setTimeout(45000, () => {
      request.destroy(new Error('下载连接超时'));
    });
  }).catch((error) => {
    log(logFn, `下载失败：${songName} - ${error.message}`);
    throw error;
  });
}

function runFfmpeg(inputPath, outputPath, webContents, taskId, songName, control) {
  return new Promise((resolve, reject) => {
    sendProgress(webContents, { taskId, songName, state: 'converting', percent: 100 });
    const proc = spawn(ffmpegPath, [
      '-y',
      '-i',
      inputPath,
      '-vn',
      '-codec:a',
      'libmp3lame',
      '-b:a',
      '320k',
      outputPath
    ]);
    if (control) control.cancel = () => proc.kill('SIGTERM');
    let stderr = '';
    proc.stderr.on('data', (chunk) => {
      stderr += chunk.toString();
    });
    proc.on('error', reject);
    proc.on('close', (code) => {
      if (control?.reason) {
        reject(cancelError(control.reason));
        return;
      }
      if (control) control.cancel = null;
      if (code === 0) resolve(outputPath);
      else reject(new Error(stderr.split('\n').slice(-4).join('\n') || `ffmpeg 退出码 ${code}`));
    });
  });
}

function fetchBuffer(url) {
  return new Promise((resolve, reject) => {
    if (!url) {
      resolve(null);
      return;
    }
    const client = url.startsWith('https:') ? https : http;
    const req = client.get(url, (res) => {
      if ([301, 302, 303, 307, 308].includes(res.statusCode) && res.headers.location) {
        resolve(fetchBuffer(new URL(res.headers.location, url).toString()));
        return;
      }
      const chunks = [];
      res.on('data', (chunk) => chunks.push(chunk));
      res.on('end', () => resolve(Buffer.concat(chunks)));
    });
    req.on('error', reject);
    req.setTimeout(20000, () => req.destroy(new Error('封面下载超时')));
  });
}

async function writeMp3Tags(filePath, song, lyricText, logFn) {
  const cover = await fetchBuffer(song.cover).catch((error) => {
    log(logFn, `封面下载失败：${error.message}`);
    return null;
  });
  const tags = {
    title: song.name,
    artist: song.artists,
    album: song.album,
    unsynchronisedLyrics: {
      language: 'zho',
      text: lyricText || ''
    }
  };
  if (cover) {
    tags.image = {
      mime: 'image/jpeg',
      type: { id: 3, name: 'front cover' },
      imageBuffer: cover
    };
  }
  NodeID3.write(tags, filePath);
}

async function downloadSong({ song, format, settings, webContents, taskId, logFn, control }) {
  let resolvedSong = song;
  try {
    resolvedSong = song.cover && song.album ? song : await getSongDetail(song.id, settings);
    const stream = await getSongUrl(resolvedSong.id, settings, format === 'source' ? 'lossless' : 'exhigh');
    const ext = extensionFromUrl(stream.url, stream.type || 'mp3');
    const baseName = sanitizeName(`${resolvedSong.artists} - ${resolvedSong.name}`);
    const sourceDir = settings.sourceDir;
    const mp3Dir = settings.mp3Dir;
    ensureDir(sourceDir);
    ensureDir(mp3Dir);

    const sourcePath = path.join(sourceDir, `${baseName}.${ext}`);
    log(logFn, `开始下载：${resolvedSong.name}`);
    await downloadFile({
      url: stream.url,
      dest: sourcePath,
      settings,
      songName: resolvedSong.name,
      webContents,
      taskId,
      logFn,
      control
    });

    let finalPath = sourcePath;
    if (format === 'mp3') {
      finalPath = path.join(mp3Dir, `${baseName}.mp3`);
      await runFfmpeg(sourcePath, finalPath, webContents, taskId, resolvedSong.name, control);
    }

    if (settings.mergeMetadata && path.extname(finalPath).toLowerCase() === '.mp3') {
      sendProgress(webContents, { taskId, songName: resolvedSong.name, state: 'tagging', percent: 100 });
      const lyric = await getLyric(resolvedSong.id, settings).catch(() => ({ lyric: '' }));
      await writeMp3Tags(finalPath, resolvedSong, lyric.lyric, logFn);
    }

    sendProgress(webContents, {
      taskId,
      songName: resolvedSong.name,
      state: 'done',
      percent: 100,
      path: finalPath
    });
    log(logFn, `完成：${resolvedSong.name} -> ${finalPath}`);
    return { path: finalPath, song: resolvedSong };
  } catch (error) {
    if (error.code === 'PAUSED' || error.code === 'STOPPED') {
      sendProgress(webContents, {
        taskId,
        songName: resolvedSong.name || song.name || '未知歌曲',
        state: error.code === 'PAUSED' ? 'paused' : 'stopped',
        percent: 0
      });
      throw error;
    }
    sendProgress(webContents, {
      taskId,
      songName: resolvedSong.name || song.name || '未知歌曲',
      state: 'failed',
      percent: 0,
      error: error.message || String(error)
    });
    log(logFn, `下载失败：${resolvedSong.name || song.name} - ${error.message || error}`);
    throw error;
  }
}

module.exports = {
  downloadSong,
  sanitizeName
};

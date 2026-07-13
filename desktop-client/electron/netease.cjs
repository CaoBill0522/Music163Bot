const api = require('NeteaseCloudMusicApi');

function cookieFromSettings(settings = {}) {
  const raw = (settings.musicU || '').trim();
  if (!raw) return '';
  if (raw.includes('=')) return raw;
  return `MUSIC_U=${raw}`;
}

function bodyOf(result) {
  return result && result.body ? result.body : result;
}

function extractMusicU(cookie) {
  const text = Array.isArray(cookie) ? cookie.join(';') : String(cookie || '');
  const matched = text.match(/(?:^|;\s*)MUSIC_U=([^;]+)/);
  return matched ? matched[1] : '';
}

async function call(name, params = {}, settings = {}) {
  const fn = api[name];
  if (typeof fn !== 'function') {
    throw new Error(`Netease API ${name} is not available`);
  }
  const cookie = cookieFromSettings(settings);
  const body = bodyOf(await fn({ ...params, cookie }));
  if (body && body.code && body.code !== 200 && body.code !== 800) {
    throw new Error(body.message || body.msg || `Netease API ${name} failed: ${body.code}`);
  }
  return body;
}

function parsePlaylistID(value = '') {
  const text = String(value).trim();
  if (/^\d+$/.test(text)) return text;
  try {
    const url = new URL(text);
    const direct = url.searchParams.get('id');
    if (direct) return direct;
    const hash = url.hash.startsWith('#') ? url.hash.slice(1) : url.hash;
    if (hash) {
      const hashQuery = hash.includes('?') ? hash.slice(hash.indexOf('?') + 1) : '';
      const id = new URLSearchParams(hashQuery).get('id');
      if (id) return id;
    }
    const matched = url.pathname.match(/playlist\/(\d+)/);
    if (matched) return matched[1];
  } catch {
    const matched = text.match(/(?:playlist\/|id=)(\d+)/);
    if (matched) return matched[1];
  }
  return '';
}

function mapSong(song = {}) {
  const artists = song.ar || song.artists || [];
  const album = song.al || song.album || {};
  return {
    id: song.id,
    name: song.name || '未知歌曲',
    artists: artists.map((item) => item.name).filter(Boolean).join(' / ') || '未知歌手',
    album: album.name || '',
    duration: song.dt || song.duration || 0,
    cover: album.picUrl || album.artist?.picUrl || '',
    fee: song.fee,
    privilege: song.privilege || null
  };
}

async function searchSongs(keyword, settings) {
  const body = await call('cloudsearch', {
    keywords: keyword,
    type: 1,
    limit: 30,
    offset: 0
  }, settings);
  return (body.result?.songs || []).map(mapSong);
}

async function getPlaylist(value, settings) {
  const id = parsePlaylistID(value);
  if (!id) throw new Error('无法识别歌单链接或 ID');
  const body = await call('playlist_detail', { id }, settings);
  const playlist = body.playlist || {};
  return {
    id,
    name: playlist.name || `歌单 ${id}`,
    cover: playlist.coverImgUrl || '',
    creator: playlist.creator?.nickname || '',
    songs: (playlist.tracks || []).map(mapSong)
  };
}

async function getSongUrl(id, settings, level = 'exhigh') {
  let body = await call('song_url_v1', {
    id: String(id),
    level
  }, settings);
  let data = body.data?.[0];
  if (!data?.url) {
    body = await call('song_url', { id: String(id), br: 320000 }, settings);
    data = body.data?.[0];
  }
  if (!data?.url) throw new Error('未获取到可播放或下载的歌曲地址，可能需要有效 MUSIC_U');
  return {
    url: data.url,
    type: data.type || '',
    size: data.size || 0,
    br: data.br || data.level || ''
  };
}

async function getSongDetail(id, settings) {
  const body = await call('song_detail', {
    ids: String(id)
  }, settings);
  return mapSong(body.songs?.[0] || {});
}

async function getLyric(id, settings) {
  const body = await call('lyric', {
    id: String(id),
    lv: -1,
    kv: -1,
    tv: -1
  }, settings);
  return {
    lyric: body.lrc?.lyric || '',
    translated: body.tlyric?.lyric || ''
  };
}

async function getProfile(settings) {
  const cookie = cookieFromSettings(settings);
  if (!cookie) {
    return {
      nickname: '未登录',
      avatarUrl: '',
      vipLabel: '未设置 MUSIC_U',
      expireAt: ''
    };
  }

  const account = await call('user_account', {}, settings).catch(() => null);
  const vip = await call('vip_info', {}, settings).catch(() => null);
  const profile = account?.profile || {};
  const userId = profile.userId || account?.account?.id;
  const userPlaylists = userId
    ? await call('user_playlist', { uid: userId, limit: 1000, offset: 0 }, settings).catch(() => null)
    : null;
  const likedPlaylist = (userPlaylists?.playlist || []).find((item) => item.specialType === 5)
    || (userPlaylists?.playlist || []).find((item) => /喜欢的音乐/.test(item.name || ''));
  const expireTime = vip?.data?.redVipDynamicIconUrl ? '' : vip?.data?.redVipExpireTime;
  const expireAt = expireTime ? new Date(expireTime).toLocaleDateString('zh-CN') : '';

  return {
    nickname: profile.nickname || '网易云用户',
    avatarUrl: profile.avatarUrl || '',
    vipLabel: vip?.data?.redVipLevel ? `黑胶 VIP Lv.${vip.data.redVipLevel}` : 'VIP 状态未知',
    expireAt,
    likedPlaylist: likedPlaylist ? {
      id: String(likedPlaylist.id),
      name: likedPlaylist.name || '我喜欢的音乐',
      cover: likedPlaylist.coverImgUrl || '',
      creator: likedPlaylist.creator?.nickname || profile.nickname || '',
      trackCount: likedPlaylist.trackCount || 0
    } : null
  };
}

async function startQrLogin() {
  const keyBody = bodyOf(await api.login_qr_key({}));
  const key = keyBody.data?.unikey;
  if (!key) throw new Error('无法生成扫码登录密钥');
  const qrBody = bodyOf(await api.login_qr_create({ key, qrimg: true }));
  if (!qrBody.data?.qrimg) throw new Error('无法生成扫码登录二维码');
  return { key, qrimg: qrBody.data.qrimg };
}

async function checkQrLogin(key) {
  if (!key) throw new Error('扫码登录已失效，请重新生成二维码');
  const body = bodyOf(await api.login_qr_check({ key }));
  return {
    code: body.code || 0,
    message: body.message || body.msg || '',
    musicU: body.code === 803 ? extractMusicU(body.cookie) : ''
  };
}

module.exports = {
  cookieFromSettings,
  checkQrLogin,
  getLyric,
  getPlaylist,
  getProfile,
  getSongDetail,
  getSongUrl,
  parsePlaylistID,
  searchSongs,
  startQrLogin
};

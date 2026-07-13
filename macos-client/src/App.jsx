import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Avatar,
  Box,
  Button,
  Checkbox,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControlLabel,
  IconButton,
  InputAdornment,
  LinearProgress,
  List,
  ListItem,
  ListItemAvatar,
  ListItemButton,
  ListItemText,
  Paper,
  Radio,
  RadioGroup,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography
} from '@mui/material';
import AlbumIcon from '@mui/icons-material/Album';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CloudDownloadIcon from '@mui/icons-material/CloudDownload';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import FavoriteIcon from '@mui/icons-material/Favorite';
import FormatListNumberedIcon from '@mui/icons-material/FormatListNumbered';
import FolderOpenIcon from '@mui/icons-material/FolderOpen';
import HeadphonesIcon from '@mui/icons-material/Headphones';
import LibraryMusicIcon from '@mui/icons-material/LibraryMusic';
import MusicNoteIcon from '@mui/icons-material/MusicNote';
import PauseCircleIcon from '@mui/icons-material/PauseCircle';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import PlaylistAddIcon from '@mui/icons-material/PlaylistAdd';
import PlaylistPlayIcon from '@mui/icons-material/PlaylistPlay';
import SearchIcon from '@mui/icons-material/Search';
import SettingsIcon from '@mui/icons-material/Settings';
import ShuffleIcon from '@mui/icons-material/Shuffle';
import StopCircleIcon from '@mui/icons-material/StopCircle';

const api = window.music163;

function formatDuration(ms = 0) {
  const total = Math.max(0, Math.floor(ms / 1000));
  const minutes = Math.floor(total / 60);
  const seconds = String(total % 60).padStart(2, '0');
  return `${minutes}:${seconds}`;
}

function formatSeconds(value = 0) {
  return formatDuration(Number(value || 0) * 1000);
}

function parseLrc(lrc = '') {
  return lrc
    .split('\n')
    .flatMap((line) => {
      const text = line.replace(/\[[^\]]+\]/g, '').trim();
      const matches = [...line.matchAll(/\[(\d{1,2}):(\d{1,2})(?:\.(\d{1,3}))?\]/g)];
      return matches.map((match) => ({
        time: Number(match[1]) * 60 + Number(match[2]) + Number(`0.${match[3] || 0}`),
        text
      }));
    })
    .filter((line) => line.text)
    .sort((a, b) => a.time - b.time);
}

function stateText(state) {
  return {
    downloading: '下载中',
    converting: '转换 MP3',
    tagging: '合并元信息',
    done: '已完成',
    failed: '下载失败',
    paused: '已暂停',
    queued: '等待中',
    stopped: '已停止'
  }[state] || '等待中';
}

function LoginPanel({ open, onMusicU }) {
  const [qr, setQr] = useState(null);
  const [qrStatus, setQrStatus] = useState('生成二维码后，用网易云音乐 App 扫码确认。');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open || !qr?.key || qr.success) return undefined;
    let cancelled = false;
    const poll = async () => {
      try {
        const result = await api.checkQrLogin(qr.key);
        if (cancelled) return;
        if (result.code === 803 && result.musicU) {
          setQr((current) => ({ ...current, success: true }));
          setQrStatus('登录成功，MUSIC_U 已填入设置。');
          onMusicU(result.musicU);
        } else if (result.code === 800) {
          setQrStatus('二维码已过期，请重新生成。');
        } else if (result.code === 802) {
          setQrStatus('已扫码，请在手机上确认。');
        } else {
          setQrStatus('等待扫码。');
        }
      } catch (error) {
        if (!cancelled) setQrStatus(error.message || '扫码状态检查失败');
      }
    };
    poll();
    const timer = setInterval(poll, 2000);
    return () => { cancelled = true; clearInterval(timer); };
  }, [onMusicU, open, qr?.key, qr?.success]);

  const createQr = async () => {
    setBusy(true);
    try {
      setQr(await api.startQrLogin());
      setQrStatus('等待扫码。');
    } catch (error) {
      setQrStatus(error.message || '二维码生成失败');
    } finally { setBusy(false); }
  };

  return (
    <Box className="login-panel">
      <Typography variant="subtitle1" fontWeight={800}>扫码登录网易云音乐</Typography>
      <Stack spacing={1.25} alignItems="center" className="qr-login">
        {qr?.qrimg ? <img src={qr.qrimg} alt="网易云音乐登录二维码" className="qr-image" /> : <Button variant="outlined" onClick={createQr} disabled={busy}>生成二维码</Button>}
        {qr?.qrimg && <Button size="small" onClick={createQr} disabled={busy}>刷新二维码</Button>}
        <Typography variant="body2" color="text.secondary" textAlign="center">{qrStatus}</Typography>
      </Stack>
    </Box>
  );
}

function SettingsDialog({ open, settings, logs, onClose, onSave, onOpenLogin }) {
  const [draft, setDraft] = useState(settings);

  useEffect(() => setDraft(settings), [settings, open]);

  const choose = async (key) => {
    const dir = await api.chooseDirectory();
    if (dir) setDraft((current) => ({ ...current, [key]: dir }));
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>设置</DialogTitle>
      <DialogContent dividers>
        <Stack spacing={2.5}>
          <TextField
            label="音乐源文件存储目录"
            value={draft.sourceDir || ''}
            onChange={(event) => setDraft({ ...draft, sourceDir: event.target.value })}
            InputProps={{ endAdornment: <InputAdornment position="end"><Tooltip title="选择目录"><IconButton onClick={() => choose('sourceDir')}><FolderOpenIcon /></IconButton></Tooltip></InputAdornment> }}
            fullWidth
          />
          <TextField
            label="MP3 存储目录"
            value={draft.mp3Dir || ''}
            onChange={(event) => setDraft({ ...draft, mp3Dir: event.target.value })}
            InputProps={{ endAdornment: <InputAdornment position="end"><Tooltip title="选择目录"><IconButton onClick={() => choose('mp3Dir')}><FolderOpenIcon /></IconButton></Tooltip></InputAdornment> }}
            fullWidth
          />
          <TextField label="MUSIC_U" value={draft.musicU || ''} onChange={(event) => setDraft({ ...draft, musicU: event.target.value })} type="password" InputProps={{ endAdornment: <InputAdornment position="end"><Tooltip title="登录并获取 MUSIC_U"><IconButton onClick={onOpenLogin}><HeadphonesIcon /></IconButton></Tooltip></InputAdornment> }} fullWidth />
          <FormControlLabel control={<Switch checked={Boolean(draft.mergeMetadata)} onChange={(event) => setDraft({ ...draft, mergeMetadata: event.target.checked })} />} label="合并歌曲元信息" />
          <Paper variant="outlined" className="log-window">
            {logs.length === 0 ? <Typography color="text.secondary">暂无日志</Typography> : logs.map((entry, index) => (
              <Typography key={`${entry.time}-${index}`} className="log-line"><span>{entry.time}</span>{entry.message}</Typography>
            ))}
          </Paper>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>取消</Button>
        <Button variant="contained" onClick={() => onSave(draft)}>保存</Button>
      </DialogActions>
    </Dialog>
  );
}

function LoginDialog({ open, onClose, onMusicU }) {
  return <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth><DialogTitle>获取 MUSIC_U</DialogTitle><DialogContent dividers><LoginPanel open={open} onMusicU={onMusicU} /></DialogContent><DialogActions><Button onClick={onClose}>完成</Button></DialogActions></Dialog>;
}

function DownloadChoiceDialog({ open, onClose }) {
  const [format, setFormat] = useState('source');

  useEffect(() => { if (open) setFormat('source'); }, [open]);

  return (
    <Dialog open={open} onClose={() => onClose(null)} maxWidth="xs" fullWidth>
      <DialogTitle>下载格式</DialogTitle>
      <DialogContent>
        <RadioGroup value={format} onChange={(event) => setFormat(event.target.value)} className="format-choice">
          <Paper variant="outlined" className={format === 'source' ? 'format-option selected' : 'format-option'}>
            <FormControlLabel value="source" control={<Radio />} label={<Box><Typography fontWeight={800}>原格式</Typography><Typography variant="body2" color="text.secondary">保留网易云提供的源文件格式</Typography></Box>} />
          </Paper>
          <Paper variant="outlined" className={format === 'mp3' ? 'format-option selected' : 'format-option'}>
            <FormControlLabel value="mp3" control={<Radio />} label={<Box><Typography fontWeight={800}>转换 MP3</Typography><Typography variant="body2" color="text.secondary">转换完成后再合并封面、歌词和歌曲信息</Typography></Box>} />
          </Paper>
        </RadioGroup>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => onClose(null)}>取消</Button>
        <Button variant="contained" startIcon={<CloudDownloadIcon />} onClick={() => onClose(format)}>下载</Button>
      </DialogActions>
    </Dialog>
  );
}

function UserPanel({ profile }) {
  return (
    <Paper className="user-panel" variant="outlined">
      <Avatar src={profile.avatarUrl} sx={{ width: 64, height: 64 }}><HeadphonesIcon /></Avatar>
      <Box minWidth={0}>
        <Typography variant="h6" noWrap>{profile.nickname || '未登录'}</Typography>
        <Stack direction="row" spacing={0.75} alignItems="center" flexWrap="wrap">
          <Chip size="small" color="primary" label={profile.vipLabel || 'VIP 状态未知'} />
          {profile.expireAt && <Chip size="small" variant="outlined" label={`到期 ${profile.expireAt}`} />}
        </Stack>
      </Box>
    </Paper>
  );
}

function LibraryPanel({ profile, savedPlaylists, onOpen }) {
  const liked = profile.likedPlaylist;
  return (
    <Paper className="library-panel" variant="outlined">
      <Typography variant="subtitle2" fontWeight={800} color="text.secondary">我的歌单</Typography>
      <List dense disablePadding className="library-list">
        {liked && <ListItemButton onClick={() => onOpen(liked)} className="library-row">
          <ListItemAvatar><Avatar variant="rounded" src={liked.cover}><FavoriteIcon /></Avatar></ListItemAvatar>
          <ListItemText primary={liked.name} secondary={`${liked.trackCount || 0} 首`} primaryTypographyProps={{ noWrap: true }} />
        </ListItemButton>}
        {savedPlaylists.map((item) => <ListItemButton key={item.id} onClick={() => onOpen(item)} className="library-row">
          <ListItemAvatar><Avatar variant="rounded" src={item.cover}><LibraryMusicIcon /></Avatar></ListItemAvatar>
          <ListItemText primary={item.name} secondary={`${item.trackCount || 0} 首`} primaryTypographyProps={{ noWrap: true }} />
        </ListItemButton>)}
        {!liked && savedPlaylists.length === 0 && <Typography variant="body2" color="text.secondary" className="library-empty">搜索歌单后可保存到这里。</Typography>}
      </List>
    </Paper>
  );
}

function SongList({ songs, onPreview, onDownload }) {
  return (
    <List className="song-list">
      {songs.map((song) => <ListItem key={song.id} divider className="song-row" secondaryAction={<Stack direction="row" spacing={0.5} className="song-actions"><Tooltip title="试听"><IconButton onClick={() => onPreview(song)}><PlayArrowIcon /></IconButton></Tooltip><Tooltip title="下载"><IconButton color="primary" onClick={() => onDownload([song])}><CloudDownloadIcon /></IconButton></Tooltip></Stack>}>
        <ListItemAvatar><Avatar variant="rounded" src={song.cover}><MusicNoteIcon /></Avatar></ListItemAvatar>
        <ListItemText primary={song.name} secondary={`${song.artists} · ${song.album || '未知专辑'} · ${formatDuration(song.duration)}`} primaryTypographyProps={{ noWrap: true }} secondaryTypographyProps={{ noWrap: true }} />
      </ListItem>)}
    </List>
  );
}

function ScrollingCell({ value }) {
  const text = String(value || '');
  if (text.length < 18) return <Box className="cell-static">{text}</Box>;
  return <Box className="cell-marquee"><Box className="marquee-track"><span>{text}</span><span aria-hidden="true">{text}</span></Box></Box>;
}

function PlaylistTable({ playlist, selected, setSelected, onPreview, onDownload, onSave, onPlayPlaylist }) {
  const songs = playlist?.songs || [];
  const allChecked = songs.length > 0 && selected.size === songs.length;
  const toggleAll = (checked) => setSelected(checked ? new Set(songs.map((song) => song.id)) : new Set());
  const toggleOne = (id) => {
    const next = new Set(selected);
    next.has(id) ? next.delete(id) : next.add(id);
    setSelected(next);
  };

  if (!playlist) return <Box className="result-empty"><PlaylistPlayIcon sx={{ fontSize: 42 }} /><Typography color="text.secondary">输入网易云歌单链接以查看歌曲</Typography></Box>;

  return (
    <Box className="playlist-box">
      <Stack direction="row" alignItems="center" justifyContent="space-between" spacing={1.5} className="playlist-summary">
        <Stack direction="row" spacing={1.5} alignItems="center" minWidth={0}>
          <Avatar variant="rounded" src={playlist.cover}><PlaylistPlayIcon /></Avatar>
          <Box minWidth={0}><Typography variant="subtitle1" fontWeight={800} noWrap>{playlist.name}</Typography><Typography variant="body2" color="text.secondary" noWrap>{songs.length} 首 · {playlist.creator}</Typography></Box>
        </Stack>
        <Stack direction="row" spacing={0.75} className="playlist-actions">
          <Tooltip title="保存歌单"><IconButton onClick={() => onSave(playlist)}><PlaylistAddIcon /></IconButton></Tooltip>
          <Button variant="outlined" startIcon={<PlaylistPlayIcon />} disabled={!songs.length} onClick={() => onPlayPlaylist(songs)}>播放歌单</Button>
          <Button variant="contained" startIcon={<CloudDownloadIcon />} disabled={selected.size === 0} onClick={() => onDownload(songs.filter((song) => selected.has(song.id)))}>下载已选</Button>
        </Stack>
      </Stack>
      <Box className="playlist-table-scroll">
        <Table size="small" className="playlist-table">
          <TableHead><TableRow><TableCell padding="checkbox" className="selection-column"><Checkbox checked={allChecked} indeterminate={selected.size > 0 && !allChecked} onChange={(event) => toggleAll(event.target.checked)} /></TableCell><TableCell>歌曲</TableCell><TableCell>歌手</TableCell><TableCell>专辑</TableCell><TableCell align="right" className="action-column">操作</TableCell></TableRow></TableHead>
          <TableBody>{songs.map((song) => <TableRow key={song.id} hover selected={selected.has(song.id)}><TableCell padding="checkbox" className="selection-column"><Checkbox checked={selected.has(song.id)} onChange={() => toggleOne(song.id)} /></TableCell><TableCell className="truncate-cell"><ScrollingCell value={song.name} /></TableCell><TableCell className="truncate-cell"><ScrollingCell value={song.artists} /></TableCell><TableCell className="truncate-cell"><ScrollingCell value={song.album} /></TableCell><TableCell align="right" className="action-column"><Stack direction="row" spacing={0.25} justifyContent="flex-end" className="table-actions"><Tooltip title="试听"><IconButton size="small" onClick={() => onPreview(song)}><PlayArrowIcon /></IconButton></Tooltip><Tooltip title="下载"><IconButton size="small" color="primary" onClick={() => onDownload([song])}><CloudDownloadIcon /></IconButton></Tooltip></Stack></TableCell></TableRow>)}</TableBody>
        </Table>
      </Box>
    </Box>
  );
}

function Player({ current, playQueue, onAdvance, onModeChange, onPlaybackState }) {
  const audioRef = useRef(null);
  const activeRef = useRef(null);
  const [playing, setPlaying] = useState(false);
  const [time, setTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const lyrics = useMemo(() => parseLrc(current?.lyric || ''), [current]);
  const activeIndex = lyrics.findIndex((line, index) => time >= line.time && (!lyrics[index + 1] || time < lyrics[index + 1].time));

  useEffect(() => { if (activeRef.current) activeRef.current.scrollIntoView({ block: 'center', behavior: 'smooth' }); }, [activeIndex]);
  useEffect(() => { setTime(0); setPlaying(false); setDuration((current?.song?.duration || 0) / 1000); }, [current?.url, current?.song?.duration]);

  if (!current) return <Paper className="player empty" variant="outlined"><AlbumIcon sx={{ fontSize: 56 }} /><Typography color="text.secondary">选择一首歌开始试听</Typography></Paper>;

  const togglePlayback = async () => {
    const audio = audioRef.current;
    if (!audio) return;
    if (audio.paused) await audio.play().catch(() => {});
    else audio.pause();
  };
  const seek = (event) => {
    const nextTime = Number(event.target.value);
    if (audioRef.current) audioRef.current.currentTime = nextTime;
    setTime(nextTime);
  };
  const total = Number.isFinite(duration) && duration > 0 ? duration : Math.max(time, 1);

  return (
    <Paper className="player" variant="outlined">
      <Box className={playing ? 'disc spinning' : 'disc'}><img src={current.song.cover || ''} alt="" /></Box>
      <Stack spacing={0.25} alignItems="center" className="now-playing"><Typography variant="h6" textAlign="center" noWrap>{current.song.name}</Typography><Typography color="text.secondary" textAlign="center" noWrap>{current.song.artists}</Typography></Stack>
      <audio ref={audioRef} key={current.url} src={current.url} autoPlay onPlay={() => { setPlaying(true); onPlaybackState(true); }} onPause={() => { setPlaying(false); onPlaybackState(false); }} onEnded={() => { setPlaying(false); onPlaybackState(false); onAdvance?.(); }} onLoadedMetadata={(event) => setDuration(event.currentTarget.duration)} onTimeUpdate={(event) => setTime(event.currentTarget.currentTime)} />
      <Box className="player-controls"><Stack direction="row" alignItems="center" spacing={0.75}><Tooltip title={playing ? '暂停' : '播放'}><IconButton color="primary" onClick={togglePlayback}>{playing ? <PauseCircleIcon /> : <PlayArrowIcon />}</IconButton></Tooltip>{playQueue && <Stack direction="row" className="play-mode-toggle"><Tooltip title="顺序播放"><IconButton size="small" color={playQueue.mode === 'sequence' ? 'primary' : 'default'} onClick={() => onModeChange('sequence')}><FormatListNumberedIcon fontSize="small" /></IconButton></Tooltip><Tooltip title="随机播放"><IconButton size="small" color={playQueue.mode === 'shuffle' ? 'primary' : 'default'} onClick={() => onModeChange('shuffle')}><ShuffleIcon fontSize="small" /></IconButton></Tooltip></Stack>}<Typography variant="caption" className="time-label">{formatSeconds(time)}</Typography><input className="audio-range" type="range" min="0" max={total} step="0.1" value={Math.min(time, total)} onChange={seek} aria-label="播放进度" /><Typography variant="caption" className="time-label">{formatSeconds(duration)}</Typography></Stack></Box>
      <Box className="lyrics">{lyrics.length === 0 ? <Typography color="text.secondary">暂无歌词</Typography> : lyrics.map((line, index) => <Typography key={`${line.time}-${index}`} ref={index === activeIndex ? activeRef : null} className={index === activeIndex ? 'lyric active' : 'lyric'}>{line.text}</Typography>)}</Box>
    </Paper>
  );
}

function ProgressPanel({ progress, batch, onPause, onResume, onStop }) {
  const items = Object.values(progress).sort((a, b) => b.createdAt - a.createdAt);
  return (
    <Paper className="progress-panel" variant="outlined">
      <Stack direction="row" alignItems="center" justifyContent="space-between" className="progress-header"><Typography variant="subtitle1" fontWeight={800}>下载进度</Typography><Stack direction="row" spacing={1} alignItems="center">{batch && <Chip size="small" color="primary" label={`${batch.current}/${batch.total}`} />}<CloudDownloadIcon color="primary" /></Stack></Stack>
      <Box className="progress-scroll"><Stack spacing={1.2}>{items.length === 0 ? <Typography color="text.secondary">暂无下载任务</Typography> : items.map((item) => {
        const canControl = !['done', 'stopped'].includes(item.state);
        const isRunning = ['downloading', 'converting', 'tagging', 'queued'].includes(item.state);
        return <Box key={item.taskId} className={`progress-item ${item.state === 'failed' ? 'failed' : ''}`}><Stack direction="row" justifyContent="space-between" spacing={1}><Typography fontWeight={700} noWrap>{item.songName}</Typography><Stack direction="row" alignItems="center" spacing={0.5} className="progress-status">{item.state === 'done' ? <CheckCircleIcon color="success" fontSize="small" /> : item.state === 'failed' ? <ErrorOutlineIcon color="error" fontSize="small" /> : null}<Typography variant="body2" color={item.state === 'failed' ? 'error' : 'text.secondary'} whiteSpace="nowrap">{stateText(item.state)}</Typography></Stack></Stack><LinearProgress color={item.state === 'failed' ? 'error' : 'primary'} variant="determinate" value={item.state === 'failed' ? 100 : item.percent || 0} />{item.error && <Typography variant="caption" color="error" className="download-error">{item.error}</Typography>}{canControl && <Stack direction="row" spacing={0.25} justifyContent="flex-end" className="queue-controls"><Tooltip title={isRunning ? '暂停' : '开始'}><IconButton size="small" onClick={() => isRunning ? onPause(item.taskId) : onResume(item.taskId)}>{isRunning ? <PauseCircleIcon fontSize="small" /> : <PlayArrowIcon fontSize="small" />}</IconButton></Tooltip><Tooltip title="停止"><IconButton size="small" color="error" onClick={() => onStop(item.taskId)}><StopCircleIcon fontSize="small" /></IconButton></Tooltip></Stack>}</Box>;
      })}</Stack></Box>
    </Paper>
  );
}

function ExitDialog({ request, onStay, onMinimize, onExit }) {
  const downloads = request?.activeDownloads || 0;
  const playing = Boolean(request?.playbackActive);
  return (
    <Dialog open={Boolean(request)} onClose={onStay} maxWidth="xs" fullWidth>
      <DialogTitle>确认退出</DialogTitle>
      <DialogContent><Typography>{downloads > 0 && `当前有 ${downloads} 个下载任务正在运行，退出会先暂停这些任务。`}{downloads > 0 && playing && ' '}{playing && '音乐正在播放，可以最小化应用以继续播放。'}</Typography></DialogContent>
      <DialogActions><Button onClick={onStay}>暂不退出</Button>{playing && <Button onClick={onMinimize}>最小化</Button>}<Button color={downloads > 0 ? 'warning' : 'error'} variant="contained" onClick={onExit}>{downloads > 0 ? '暂停任务后退出' : '退出'}</Button></DialogActions>
    </Dialog>
  );
}

export default function App() {
  const [settings, setSettings] = useState({ savedPlaylists: [] });
  const [profile, setProfile] = useState({});
  const [logs, setLogs] = useState([]);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [loginOpen, setLoginOpen] = useState(false);
  const [input, setInput] = useState('');
  const [songs, setSongs] = useState([]);
  const [playlist, setPlaylist] = useState(null);
  const [selected, setSelected] = useState(new Set());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [current, setCurrent] = useState(null);
  const [playQueue, setPlayQueue] = useState(null);
  const [progress, setProgress] = useState({});
  const [choiceOpen, setChoiceOpen] = useState(false);
  const [batch, setBatch] = useState(null);
  const [closeRequest, setCloseRequest] = useState(null);
  const choiceResolver = useRef(null);

  useEffect(() => {
    api.getSettings().then((value) => { setSettings({ ...value, savedPlaylists: value.savedPlaylists || [] }); setLogs(value.logs || []); });
    api.getProfile().then(setProfile).catch(() => {});
    api.getQueue().then((items) => setProgress(Object.fromEntries(items.map((item) => [item.taskId, item])))).catch(() => {});
    const offProgress = api.onProgress((payload) => {
      setProgress((currentProgress) => ({ ...currentProgress, [payload.taskId]: { ...currentProgress[payload.taskId], ...payload, createdAt: currentProgress[payload.taskId]?.createdAt || Date.now() } }));
      if (payload.batchId) setBatch((currentBatch) => currentBatch?.batchId === payload.batchId ? { ...currentBatch, current: payload.batchIndex || currentBatch.current } : currentBatch);
    });
    const offLog = api.onLog((entry) => setLogs((currentLogs) => [...currentLogs.slice(-599), entry]));
    const offCloseRequested = api.onCloseRequested(setCloseRequest);
    return () => { offProgress(); offLog(); offCloseRequested(); };
  }, []);

  const saveSettings = async (draft, close = true) => {
    const next = await api.saveSettings(draft);
    setSettings({ ...next, savedPlaylists: next.savedPlaylists || [] });
    if (close) setSettingsOpen(false);
    api.getProfile().then(setProfile).catch(() => {});
  };
  const chooseFormat = () => new Promise((resolve) => { choiceResolver.current = resolve; setChoiceOpen(true); });
  const closeChoice = (format) => { setChoiceOpen(false); choiceResolver.current?.(format); choiceResolver.current = null; };
  const loadPlaylist = async (reference) => {
    setError(''); setLoading(true);
    try {
      const result = await api.getPlaylist(reference.id || reference);
      setPlaylist(result); setSongs([]); setSelected(new Set(result.songs.map((song) => song.id)));
    } catch (err) { setError(err.message || String(err)); }
    finally { setLoading(false); }
  };
  const handleSearch = async () => {
    const value = input.trim();
    if (!value) return;
    setError(''); setLoading(true);
    try {
      if (/^https:\/\//i.test(value)) await loadPlaylist(value);
      else { setSongs(await api.searchSongs(value)); setPlaylist(null); setSelected(new Set()); }
    } catch (err) { setError(err.message || String(err)); }
    finally { setLoading(false); }
  };
  const handlePreview = async (song) => {
    setError('');
    try { setCurrent({ song, ...(await api.getPreview(song)) }); }
    catch (err) { setError(err.message || String(err)); }
  };
  const previewSingle = async (song) => {
    setPlayQueue(null);
    await handlePreview(song);
  };
  const playPlaylist = async (songs) => {
    if (!songs.length) return;
    setPlayQueue({ songs, index: 0, mode: 'sequence' });
    await handlePreview(songs[0]);
  };
  const advancePlaylist = async () => {
    if (!playQueue?.songs?.length) return;
    const { songs, index, mode } = playQueue;
    const nextIndex = mode === 'shuffle' && songs.length > 1
      ? (() => { let next = index; while (next === index) next = Math.floor(Math.random() * songs.length); return next; })()
      : (index + 1) % songs.length;
    setPlayQueue({ ...playQueue, index: nextIndex });
    await handlePreview(songs[nextIndex]);
  };
  const handleDownload = async (selectedSongs) => {
    if (selectedSongs.length === 0) return;
    const format = await chooseFormat();
    if (!format) return;
    const created = await api.enqueueDownloads(selectedSongs, format);
    setBatch({ batchId: created.batchId, current: 0, total: created.total });
  };
  const savePlaylist = async (item) => {
    const saved = settings.savedPlaylists || [];
    const nextSaved = [...saved.filter((savedItem) => String(savedItem.id) !== String(item.id)), { id: String(item.id), name: item.name, cover: item.cover, creator: item.creator, trackCount: item.songs?.length || item.trackCount || 0 }];
    await saveSettings({ ...settings, savedPlaylists: nextSaved }, false);
  };
  const togglePreventSleep = (event) => saveSettings({ ...settings, preventSleep: event.target.checked }, false);
  const isPlaylist = Boolean(playlist);

  return (
    <Box className="app-shell">
      <Box className="top-bar"><Stack direction="row" spacing={1.2} alignItems="center"><Box className="brand-mark"><MusicNoteIcon /></Box><Typography variant="h5" fontWeight={900}>163MUSIC</Typography></Stack><Tooltip title="设置"><IconButton onClick={() => setSettingsOpen(true)}><SettingsIcon /></IconButton></Tooltip></Box>
      <Box className="main-grid">
        <Box className="left-pane"><UserPanel profile={profile} /><LibraryPanel profile={profile} savedPlaylists={settings.savedPlaylists || []} onOpen={loadPlaylist} /></Box>
        <Box className="content-pane">
          <Paper className="search-panel" variant="outlined"><Stack direction="row" spacing={1.5}><TextField value={input} onChange={(event) => setInput(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter') handleSearch(); }} placeholder="输入歌曲名称或网易云歌单链接" InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment> }} fullWidth /><Button variant="contained" startIcon={<SearchIcon />} onClick={handleSearch} disabled={loading}>搜索</Button></Stack>{loading && <LinearProgress sx={{ mt: 1.5 }} />}{error && <Alert severity="error" sx={{ mt: 1.5 }}>{error}</Alert>}</Paper>
          <Paper className="result-panel" variant="outlined"><Stack direction="row" alignItems="center" justifyContent="space-between" mb={1.5}><Typography variant="subtitle1" fontWeight={800}>{isPlaylist ? '歌单歌曲' : '搜索结果'}</Typography>{isPlaylist ? <Button startIcon={<CloudDownloadIcon />} variant="outlined" disabled={!playlist?.songs?.length} onClick={() => handleDownload(playlist.songs)}>全部下载</Button> : <Chip icon={<MusicNoteIcon />} label={`${songs.length} 首`} />}</Stack><Divider />{isPlaylist ? <PlaylistTable playlist={playlist} selected={selected} setSelected={setSelected} onPreview={previewSingle} onDownload={handleDownload} onSave={savePlaylist} onPlayPlaylist={playPlaylist} /> : <SongList songs={songs} onPreview={previewSingle} onDownload={handleDownload} />}</Paper>
        </Box>
        <Box className="right-pane"><ProgressPanel progress={progress} batch={batch} onPause={api.pauseDownload} onResume={api.resumeDownload} onStop={api.stopDownload} /><Player current={current} playQueue={playQueue} onAdvance={advancePlaylist} onModeChange={(mode) => setPlayQueue((queue) => queue ? { ...queue, mode } : queue)} onPlaybackState={api.setPlaybackActive} /></Box>
      </Box>
      <Box component="footer" className="app-footer"><FormControlLabel control={<Checkbox size="small" checked={Boolean(settings.preventSleep)} onChange={togglePreventSleep} />} label="忽略系统休眠" /></Box>
      <SettingsDialog open={settingsOpen} settings={settings} logs={logs} onClose={() => setSettingsOpen(false)} onSave={saveSettings} onOpenLogin={() => setLoginOpen(true)} />
      <LoginDialog open={loginOpen} onClose={() => setLoginOpen(false)} onMusicU={async (musicU) => { await saveSettings({ ...settings, musicU }, false); }} />
      <DownloadChoiceDialog open={choiceOpen} onClose={closeChoice} />
      <ExitDialog request={closeRequest} onStay={() => { setCloseRequest(null); api.respondToClose('stay'); }} onMinimize={() => { setCloseRequest(null); api.respondToClose('minimize'); }} onExit={() => { setCloseRequest(null); api.respondToClose('exit'); }} />
    </Box>
  );
}

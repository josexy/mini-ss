import {
    Box, Button, Checkbox,
    FormControl, FormControlLabel,
    FormGroup, Grid, IconButton,
    InputLabel, MenuItem, Select,
    Stack, Switch, TextField, Tooltip
} from "@mui/material";
import {
    GetAllInterfaceName, SaveLocalConfig, SpeedTest,
    StartServer, StopServer, StopSpeedTest
} from "../../wailsjs/go/main/App"
import { PlayCircle, Save, Speed, Stop } from '@mui/icons-material';
import { useEffect, useState } from "react";
import { config } from '../../wailsjs/go/models';

interface Props {
    cfgLocal: config.LocalJsonConfig | undefined
    cfgIfaceName: string | undefined
    cfgAutoDetectIface: boolean | undefined
    showToast: (message: string, type: string) => void
}

export default function CHome({ cfgLocal, cfgIfaceName, cfgAutoDetectIface, showToast }: Props) {
    const [isStart, setStart] = useState(false)

    const [enableSocks, setEnableSocks] = useState(true)
    const [enableHttp, setEnableHttp] = useState(true)
    const [enableMixed, setEnableMixed] = useState(false)

    const [socksProxy, setSocksProxy] = useState('127.0.0.1:10086')
    const [socksAuth, setSocksAuth] = useState('')
    const [httpProxy, setHttpProxy] = useState('127.0.0.1:10087')
    const [httpAuth, setHttpAuth] = useState('')
    const [mixedProxy, setMixedProxy] = useState('127.0.0.1:10088')

    const [isSystemProxy, setSystemProxy] = useState(false)
    const [isTunMode, setTunMode] = useState(false)
    const [tunName, setTunName] = useState('utun3')
    const [tunCIDR, setTunCIDR] = useState('198.18.0.1/16')
    const [tunMTU, setTunMTU] = useState(1350)
    const [fakeDnsListen, setFakeDnsListen] = useState(':53')
    const [nameservers, setNameservers] = useState('8.8.8.8')

    const [ifaceName, setIfaceName] = useState('')
    const [autoDetectIface, setAutoDetectIface] = useState(false)
    const [ifaceList, setIfaceList] = useState<string[]>([])

    // speed test 
    const [isStartSpeedTest, SetStartSpeedTest] = useState(false)
    const [speedTestRate, setSpeedTestRate] = useState('')

    const changeMixedProxy = (checked: boolean) => {
        setEnableMixed(checked)
        changeHttpProxy(!checked)
        changeSocksProxy(!checked)
    }

    const changeSocksProxy = (checked: boolean) => { setEnableSocks(checked) }
    const changeHttpProxy = (checked: boolean) => { setEnableHttp(checked) }

    const startProxy = () => {
        // when start local server, save the json config
        _saveLocalConfig().then(() => {
            StartServer().then(() => {
                setStart(true)
                showToast('??????????????????????????????', 'success')
            }).catch(err => {
                showToast(`??????: ${err}`, 'error')
            })
        })
    }

    const stopProxy = () => {
        setStart(false)

        // stop the speed test if started when stop local proxy server
        StopSpeedTest()
        setSpeedTestRate('')

        StopServer()
            .then(() => showToast('??????????????????????????????', 'success'))
            .catch(err => {
                showToast(`??????: ${err}`, 'error')
            })
    }

    const _saveLocalConfig = () => {
        let localCfg = config.LocalJsonConfig.createFrom({
            socks_addr: socksProxy,
            http_addr: httpProxy,
            socks_auth: socksAuth,
            http_auth: httpAuth,
            mixed_addr: mixedProxy,
            system_proxy: isSystemProxy,
            enable_tun: isTunMode,
            tun: config.TunOption.createFrom({
                name: tunName,
                cidr: tunCIDR,
                mtu: tunMTU,
            }),
            fake_dns: config.FakeDnsOption.createFrom({
                listen: fakeDnsListen,
                nameservers: nameservers.split(","),
            })
        })

        if (!enableSocks) {
            localCfg.socks_addr = ""
            localCfg.socks_auth = ""
        }

        if (!enableHttp) {
            localCfg.http_addr = ""
            localCfg.http_auth = ""
        }
        if (!enableMixed) {
            localCfg.mixed_addr = ""
        }
        return SaveLocalConfig(localCfg, ifaceName, autoDetectIface)
    }

    const saveLocalConfig = () => {
        _saveLocalConfig().then(() => showToast('?????????????????????', 'success'))
    }

    const speedTest = () => {
        if (!isStart) {
            showToast('??????????????????????????????', 'error')
            return
        }

        // stop 
        if (isStartSpeedTest) {
            StopSpeedTest()
            SetStartSpeedTest(false)
            return
        }

        // start
        SetStartSpeedTest(true)
        setSpeedTestRate('')

        SpeedTest().then(rate => {
            showToast('???????????????', 'success')
            SetStartSpeedTest(false)
            setSpeedTestRate(rate)
        }).catch(err => {
            showToast(`??????: ${err}`, 'error')
            SetStartSpeedTest(false)
        })
    }

    useEffect(() => {
        GetAllInterfaceName().then(list => setIfaceList(list))
    }, [])

    useEffect(() => {
        // init local options
        if (cfgLocal) {
            setSocksProxy(cfgLocal.socks_addr)
            setSocksAuth(cfgLocal.socks_auth)
            setHttpProxy(cfgLocal.http_addr)
            setHttpAuth(cfgLocal.http_auth)
            setMixedProxy(cfgLocal.mixed_addr)
            setSystemProxy(cfgLocal.system_proxy)
            setTunMode(cfgLocal.enable_tun)
            if (cfgLocal.tun) {
                setTunName(cfgLocal.tun.name)
                setTunCIDR(cfgLocal.tun.cidr)
                setTunMTU(cfgLocal.tun.mtu)
            }
            if (cfgLocal.fake_dns) {
                setFakeDnsListen(cfgLocal.fake_dns.listen)
                setNameservers(cfgLocal.fake_dns.nameservers.join(','))
            }
            if (cfgLocal.http_addr != "") {
                changeHttpProxy(true)
            }
            if (cfgLocal.socks_addr != "") {
                changeSocksProxy(true)
            }
            if (cfgLocal.mixed_addr != "") {
                changeMixedProxy(true)
            }
        }
        if (cfgIfaceName) {
            setIfaceName(cfgIfaceName)
        }
        if (cfgAutoDetectIface) {
            setAutoDetectIface(cfgAutoDetectIface)
        }
    }, [cfgLocal, cfgIfaceName, cfgAutoDetectIface])

    return (
        <Stack
            spacing={1}
            justifyContent="center"
            alignItems="center">
            <Box>
                <FormGroup row >
                    <Checkbox size="small" checked={enableSocks}
                        onChange={(_, checked: boolean) => changeSocksProxy(checked)} />
                    <TextField variant="standard" label="SOCKS??????"
                        disabled={!enableSocks}
                        size="small"
                        value={socksProxy}
                        onChange={(e: any) => setSocksProxy(e.target.value)}
                    />
                    <TextField variant="standard" label="SOCKS????????????"
                        disabled={!enableSocks}
                        size="small"
                        value={socksAuth}
                        onChange={(e: any) => setSocksAuth(e.target.value)}
                    />
                </FormGroup>
                <FormGroup row >
                    <Checkbox size="small" checked={enableHttp}
                        onChange={(_, checked: boolean) => changeHttpProxy(checked)} />
                    <TextField variant="standard" label="HTTP??????"
                        disabled={!enableHttp}
                        size="small"
                        value={httpProxy}
                        onChange={(e: any) => setHttpProxy(e.target.value)}
                    />
                    <TextField variant="standard" label="HTTP????????????"
                        disabled={!enableHttp}
                        size="small"
                        value={httpAuth}
                        onChange={(e: any) => setHttpAuth(e.target.value)}
                    />
                </FormGroup>
                <FormGroup row >
                    <Checkbox size="small" checked={enableMixed}
                        onChange={(_, checked: boolean) => changeMixedProxy(checked)} />
                    <TextField variant="standard" label="????????????"
                        disabled={!enableMixed}
                        size="small"
                        value={mixedProxy}
                        onChange={(e: any) => setMixedProxy(e.target.value)}
                    />
                </FormGroup>
            </Box>
            <Grid container spacing={0.5} alignItems="center" justifyContent={"center"}>
                <Grid item>
                    <FormGroup>
                        <FormControlLabel control={
                            <Switch checked={isSystemProxy}
                                onChange={(_, checked: boolean) => { setSystemProxy(checked) }}
                            />}
                            label="????????????" />
                    </FormGroup>
                </Grid>
                <Grid item>
                    <FormGroup>
                        <FormControlLabel control={
                            <Switch checked={isTunMode}
                                onChange={(_, checked: boolean) => { setTunMode(checked) }}
                            />}
                            label="Tun??????" />
                    </FormGroup>
                </Grid>
                <Grid item>
                    <FormControl size='small' margin='dense'>
                        <InputLabel id="interface">????????????</InputLabel>
                        <Select
                            labelId="interface"
                            value={ifaceName}
                            onChange={(e: any) => setIfaceName(e.target.value)}
                            label="????????????"
                        >
                            {
                                ifaceList.map((name, index) => <MenuItem key={index} value={name}>{name}</MenuItem>)
                            }
                        </Select>
                    </FormControl>
                </Grid>
                <Grid item>
                    <FormGroup row>
                        <FormControlLabel control={<Checkbox
                            checked={autoDetectIface}
                            onChange={(_, checked: boolean) => setAutoDetectIface(checked)} />} label="??????" />
                    </FormGroup>
                </Grid>
            </Grid>

            <Stack
                sx={{ display: isTunMode ? 'flex' : 'none' }}
                direction={"row"}>
                <Box sx={{ margin: 1 }}>
                    <Stack padding={1}>
                        <TextField variant="standard" label="????????????"
                            size="small"
                            value={tunName}
                            onChange={(e: any) => setTunName(e.target.value)}
                        />
                        <TextField variant="standard" label="CIDR"
                            size="small"
                            value={tunCIDR}
                            onChange={(e: any) => setTunCIDR(e.target.value)}
                        />
                        <TextField variant="standard" label="MTU"
                            size="small"
                            type={"number"}
                            value={tunMTU}
                            onChange={(e: any) => setTunMTU(e.target.value)}
                        />
                    </Stack>
                </Box>
                <Box sx={{ margin: 1 }}>
                    <Stack padding={1}>
                        <TextField variant="standard" label="FakeDNS"
                            size="small"
                            value={fakeDnsListen}
                            onChange={(e: any) => setFakeDnsListen(e.target.value)}
                        />
                        <TextField variant="standard" label="Nameservers"
                            size="small"
                            value={nameservers}
                            onChange={(e: any) => setNameservers(e.target.value)}
                        />
                    </Stack>
                </Box>
            </Stack>
            <Stack direction="row" justifyContent={"center"} alignItems={"center"}>
                <Tooltip title='??????????????????'>
                    <IconButton
                        size='large'
                        color={isStartSpeedTest ? 'success' : 'inherit'}
                        onClick={() => speedTest()}
                    >
                        <Speed sx={{ fontSize: '50px' }} />
                    </IconButton>
                </Tooltip>
                <Box sx={{ padding: 1 }}>
                    {isStartSpeedTest ? "????????????..." : speedTestRate}
                </Box>
            </Stack>
            <Stack spacing={1} direction={"row"}>
                <Button startIcon={<Save />}
                    variant="outlined"
                    onClick={() => saveLocalConfig()}>??????</Button>
                <Button startIcon={<PlayCircle />}
                    disabled={isStart}
                    variant="outlined" color="success"
                    onClick={() => startProxy()}>??????</Button >
                <Button startIcon={<Stop />}
                    disabled={!isStart}
                    variant="outlined" color="error"
                    onClick={() => stopProxy()}>??????</Button>
            </Stack>
        </Stack >
    )
}
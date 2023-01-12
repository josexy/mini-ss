import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import Box from '@mui/material/Box';
import {
    Computer,
    ContentPaste,
    Home, List,
    Person, Rule,
    TextSnippet
} from '@mui/icons-material';
import { useState } from 'react';
import { Alert, AlertColor, Snackbar } from '@mui/material';
import CHome from './CHome';
import CServers from './CServers';
import CRules from './CRules';
import CAbout from './CAbout';
import CConfig from './CConfig';
import CConnections from './Connections';
import { config } from '../../wailsjs/go/models';

interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
}

function TabPanel(props: TabPanelProps) {
    const { children, value, index, ...other } = props;
    return (
        <div
            role="tabpanel"
            hidden={value !== index}
            id={`vertical-tabpanel-${index}`}
            aria-labelledby={`vertical-tab-${index}`}
            {...other}
        >
            <Box sx={{ padding: 2 }}>
                {children}
            </Box>
        </div>
    )
}

function a11yProps(index: number) {
    return {
        id: `vertical-tab-${index}`,
        'aria-controls': `vertical-tabpanel-${index}`,
    };
}


export default function CMainTabs() {
    const [tabIndex, setTabIndex] = useState(0)
    const [cfgJson, setCfgJson] = useState<config.JsonConfig>()

    // alert message show/hide
    const [alertOpen, setAlertOpen] = useState(false)
    const [alertMessage, setAlertMessage] = useState('')
    const [alertType, setAlertType] = useState<AlertColor>('success')

    const handleChange = (e: any, newValue: number) => {
        setTabIndex(newValue);
    }

    const handleAlertClose = (e: any, reason?: string) => {
        if (reason === 'clickaway') {
            return
        }
        setAlertOpen(false);
    };

    const showToast = (message: string, type: string) => {
        setAlertType(type as AlertColor)
        setAlertMessage(message)
        setAlertOpen(true)
    }

    return (
        <Box sx={{ width: '100%' }}>
            <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                <Tabs
                    centered
                    value={tabIndex}
                    onChange={handleChange}
                >
                    <Tab icon={<Home />} label="主页" {...a11yProps(0)} />
                    <Tab icon={<List />} label="节点" {...a11yProps(1)} />
                    <Tab icon={<Rule />} label="规则" {...a11yProps(2)} />
                    <Tab icon={<Computer />} label="连接" {...a11yProps(3)} />
                    <Tab icon={<TextSnippet />} label="配置" {...a11yProps(4)} />
                    <Tab icon={<Person />} label="关于" {...a11yProps(5)} />
                </Tabs>
            </Box>
            <Box>
                <TabPanel value={tabIndex} index={0}>
                    <CHome
                        cfgLocal={cfgJson?.local}
                        cfgIfaceName={cfgJson?.iface}
                        cfgAutoDetectIface={cfgJson?.auto_detect_iface}
                        showToast={showToast} />
                </TabPanel>
                <TabPanel value={tabIndex} index={1}>
                    <CServers cfgServer={cfgJson?.server} showToast={showToast} />
                </TabPanel>
                <TabPanel value={tabIndex} index={2}>
                    <CRules cfgRules={cfgJson?.rules} />
                </TabPanel>
                <TabPanel value={tabIndex} index={3}>
                    <CConnections />
                </TabPanel>
                <TabPanel value={tabIndex} index={4}>
                    <CConfig setCfgJson={setCfgJson} showToast={showToast} />
                </TabPanel>
                <TabPanel value={tabIndex} index={5}>
                    <CAbout />
                </TabPanel>
            </Box>
            <Snackbar open={alertOpen} autoHideDuration={2000}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
                onClose={handleAlertClose}>
                <Alert onClose={handleAlertClose} severity={alertType} sx={{ width: '100%' }}>
                    {alertMessage}
                </Alert>
            </Snackbar>
        </Box>
    );
}

import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';
import './App.css';

import { Container } from '@mui/material';
import CMainTabs from './components/CMainTabs';

export default function App() {
    return (
        <Container>
            <CMainTabs />
        </Container>
    )
}

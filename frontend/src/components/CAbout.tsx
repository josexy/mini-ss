import { GitHub } from "@mui/icons-material";
import { Box, IconButton, Stack, Tooltip } from "@mui/material";
import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";

export default function CAbout() {

    const openGithub = () => {
        BrowserOpenURL("https://github.com/josexy")
    }

    return (
        <Stack justifyContent="center" alignItems={"center"}>
            <Box>
                <strong>mini-ss</strong>
            </Box>
            <Box>
                version: 1.0
            </Box>
            <Tooltip title="打开Github主页">
                <IconButton onClick={() => openGithub()}>
                    <GitHub />
                </IconButton>
            </Tooltip>
        </Stack>
    )
}
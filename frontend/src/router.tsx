import { createHashRouter } from "react-router";
import { Layout } from "@/wechat";
import { ContactPage } from '@/wechat/contact';
import { ChatRoomPage } from "@/wechat/chatroom";
import { SessionPage } from "@/wechat/session";
import NotFoundPage from "./NotFoundPage";

const router = createHashRouter([
    {
        path: '/',
        element: <Layout />,
        children: [
            { index: true, element: <ContactPage /> },
            { path: 'contact', element: <ContactPage /> },
            { path: 'chatroom', element: <ChatRoomPage /> },
            { path: 'session', element: <SessionPage /> },
            { path: '*', element: <NotFoundPage /> },
        ],
    }
]);

export default router;

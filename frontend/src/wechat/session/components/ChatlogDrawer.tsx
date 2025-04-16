import React from 'react';
import {
  makeStyles,
  Button,
  DrawerBody,
  DrawerHeader,
  DrawerHeaderTitle,
  OverlayDrawer,
  useRestoreFocusSource,
} from '@fluentui/react-components';
import type { DialogOpenChangeEvent, DialogOpenChangeData } from '@fluentui/react-components';
import { Dismiss24Regular } from '@fluentui/react-icons';
import type { ChatlogItem, ChatSessionItem, GetDataParams } from '@/wechat/typing';
import { getChatlog } from '@/wechat/WeChatService';
import { useRequest } from '@/hooks/useRequest';

const useStyles = makeStyles({
  root: {
    width: '500px',
  },
});

type ChatlogDrawerProps = {
  isOpen: boolean;
  setIsOpen: (isOpen: boolean) => void;
  currentChatSessionItem: ChatSessionItem;
}

export function ChatlogDrawer(props: ChatlogDrawerProps) {
  const { isOpen, setIsOpen, currentChatSessionItem } = props;

  const styles = useStyles();
  const restoreFocusSourceAttributes = useRestoreFocusSource();
  const [chatlogs, setChatlogs] = React.useState<ChatlogItem[]>([]);
  const { run } = useRequest<GetDataParams, ChatlogItem[]>(params => getChatlog(params!));

  React.useEffect(() => {
    if (isOpen) {
      const params = {
        limit: 10,
        offset: 0,
        talker: currentChatSessionItem.userName,
        time: currentChatSessionItem.nTime,
      };
      run(params).then((res) => {
        setChatlogs(res);
      });
    }
    return () => {};
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen]);

  const handleOnOpenChange = (_: DialogOpenChangeEvent, { open }: DialogOpenChangeData) => {
    setIsOpen(open);
  };

  return (
    <OverlayDrawer
      position="end"
      {...restoreFocusSourceAttributes}
      open={isOpen}
      onOpenChange={handleOnOpenChange}
      className={styles.root}
    >
      <DrawerHeader>
        <DrawerHeaderTitle
          action={
            <Button
              appearance="subtle"
              aria-label="Close"
              icon={<Dismiss24Regular />}
              onClick={() => setIsOpen(false)}
            />
          }
        >
          Chat Session Details
        </DrawerHeaderTitle>
      </DrawerHeader>

      <DrawerBody>
        <ul>
          <li>{currentChatSessionItem?.userName}</li>
          <li>{currentChatSessionItem?.nickName}</li>
          <li>{currentChatSessionItem?.nOrder}</li>
          <li>{currentChatSessionItem?.nTime}</li>
          <li>{currentChatSessionItem?.content}</li>
        </ul>
        <ul>
          {chatlogs.map((log) => (
            <li key={log.seq}>
              <strong>{log.talkerName}:</strong> {log.content}
            </li>
          ))}
        </ul>
      </DrawerBody>
    </OverlayDrawer>
  );
}
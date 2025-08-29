import React from 'react';
import {
  makeStyles,
  Button,
  DrawerBody,
  DrawerHeader,
  DrawerHeaderTitle,
  OverlayDrawer,
  useRestoreFocusSource,
  Spinner,
  DrawerFooter,
} from '@fluentui/react-components';
import type { DialogOpenChangeEvent, DialogOpenChangeData } from '@fluentui/react-components';
import { CalendarStrings, DatePicker, defaultDatePickerStrings } from '@fluentui/react-datepicker-compat';
import dayjs from 'dayjs';
import { Dismiss24Regular } from '@fluentui/react-icons';
import type { ChatlogItem, ChatSessionItem, GetDataParams } from '@/wechat/typing';
import { getChatlog } from '@/wechat/WeChatService';
import { useRequest } from '@/hooks/useRequest';

const localizedStrings: CalendarStrings = {
  ...defaultDatePickerStrings,
  days: [
    "X",
    "Lunes",
    "Martes",
    "Miercoles",
    "Jueves",
    "Viernes",
    "Sabado",
  ],
  shortDays: ["日", "一", "二", "三", "四", "五", "六"],
  months: [
    "1月",
    "2月",
    "3月",
    "4月",
    "5月",
    "6月",
    "7月",
    "8月",
    "9月",
    "10月",
    "11月",
    "12月",
  ],

  shortMonths: [
    "1月",
    "2月",
    "3月",
    "4月",
    "5月",
    "6月",
    "7月",
    "8月",
    "9月",
    "10月",
    "11月",
    "12月",
  ],
  goToToday: "今天",
};

const useStyles = makeStyles({
  root: {
    width: '600px',
  },
  loading: {
    marginLeft: '10px',
    display: 'inline-block',
    verticalAlign: 'middle',
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
  const [limit, setLimit] = React.useState(10);
  const [offset, setOffset] = React.useState(0);
  const [time, setTime] = React.useState('');
  const [chatlogs, setChatlogs] = React.useState<ChatlogItem[]>([]);
  const { loading, run } = useRequest<GetDataParams, ChatlogItem[]>(params => getChatlog(params!));

  React.useEffect(() => {
    console.log('ChatlogDrawer mounted -> ' + Date.now());
    if (isOpen) {
      setLimit(10);
      setOffset(0);
      setChatlogs([]);
      setTime(currentChatSessionItem?.nTime ? dayjs(currentChatSessionItem.nTime).format('YYYY-MM-DD') : dayjs().format('YYYY-MM-DD'));
    }

    return () => {
      console.log('ChatlogDrawer unmounted -> ' + Date.now());
      setTime('');
      setChatlogs([]);
    };
  }, [isOpen, currentChatSessionItem]);

  React.useEffect(() => {
    if (currentChatSessionItem?.userName) {
      loadData();
    }

    return () => { };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [limit, offset, time, currentChatSessionItem]);

  const loadData = async () => {
    if (!time) {
      return;
    }

    run({ limit, offset, talker: currentChatSessionItem.userName, time })
      .then(data => {
        setChatlogs([...chatlogs, ...data]);
      })
      .catch(error => {
        console.error(error);
      })
      .finally(() => {
      });
  };

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
          <span>Chat Session Details({currentChatSessionItem?.userName})</span>
          <span className={styles.loading}>
            {loading && <Spinner size="tiny" aria-live="polite" />}
          </span>
        </DrawerHeaderTitle>
      </DrawerHeader>

      <DrawerBody>
        <div>
          <DatePicker
            strings={localizedStrings}
            value={time ? dayjs(time).toDate() : null}
            placeholder="Select a date..."
            formatDate={date => dayjs(date).format('YYYY-MM-DD')}
            onSelectDate={(date) => {
              const selectedDate = dayjs(date).format('YYYY-MM-DD');
              setTime(selectedDate);
              // loadData(currentChatSessionItem.userName, selectedDate);
            }}
          />
        </div>
        <ul>
          {chatlogs.map((log) => (
            <li key={log.seq}>
              <strong>{log.talkerName}:</strong> {log.content}
            </li>
          ))}
        </ul>
      </DrawerBody>

      <DrawerFooter>
        <Button appearance="secondary" onClick={() => setIsOpen(false)}>
          Close
        </Button>
        <Button appearance="primary" onClick={() => setOffset(offset + limit)}>
          Load More
        </Button>
      </DrawerFooter>
    </OverlayDrawer>
  );
}
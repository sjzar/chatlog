import React from 'react';
import {
  makeStyles,
  tokens,
  TableBody,
  TableCell,
  TableRow,
  Table,
  TableHeader,
  TableHeaderCell,
  TableCellLayout,
  Button,
  useId,
  Option,
  Dropdown,
  Link,
} from "@fluentui/react-components";
import type {
  SelectionEvents,
  OptionOnSelectData,
} from '@fluentui/react-components';
import { useRequest } from '@/hooks/useRequest';
import { getChatSessions } from '../WeChatService';
import { ChatSessionData, ChatSessionItem, GetDataParams } from '../typing';
import toast, { Toaster } from 'react-hot-toast';
import { ChatlogDrawer } from './components/ChatlogDrawer';

const columns = [
  { columnKey: "userName", label: "UserName" },
  { columnKey: "nickName", label: "NickName" },
  { columnKey: "nOrder", label: "Order" },
  { columnKey: "nTime", label: "Time" },
  { columnKey: "content", label: "Content" },
];

const useStyles = makeStyles({
  root: {
    height: `calc(100% - ${tokens.spacingHorizontalL})`,
  },

  pagination: {
    marginTop: tokens.spacingHorizontalL,
    display: 'flex',
    gap: tokens.spacingHorizontalL
  },
  dropdown: {
    width: '100px',
    minWidth: '100px',
  },
});

export function SessionPage() {
  const [limit, setLimit] = React.useState(10);
    const [offset, setOffset] = React.useState(0);
    const { run } = useRequest<GetDataParams, ChatSessionData>(params => getChatSessions(params!));
    const [items, setItems] = React.useState<ChatSessionItem[]>([]);
    const styles = useStyles();
    const dropdownId = useId("dropdown-default");
    const options = [
      '10',
      '20',
      '30',
      '50',
      '100',
      '200',
      '500',
    ];
    const [isOpen, setIsOpen] = React.useState(false);
    const [currentChatSessionItem, setCurrentChatSessionItem] = React.useState<ChatSessionItem>();
  
    React.useEffect(() => {
      loadData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [limit, offset]);
  
    const loadData = async () => {
      const loadingToastId = toast.loading("Loading contact...");
      run({ limit, offset }).then(data => {
        setItems(data.items);
      }).catch(error => {
        console.error(error);
        toast.error("Failed to load contact data.");
      })
        .finally(() => {
          toast.dismiss(loadingToastId);
        });
    };
  
    const handleOnOptionSelect = (_: SelectionEvents, data: OptionOnSelectData) => {
      setLimit(Number(data.optionValue));
    }
  
    return (
      <div className={styles.root}>
        <Toaster position="top-center" />
        <Table arial-label="Default table" style={{ minWidth: "510px" }}>
          <TableHeader>
            <TableRow>
              {columns.map((column) => (
                <TableHeaderCell key={column.columnKey}>
                  {column.label}
                </TableHeaderCell>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((item) => (
              <TableRow key={item.userName}>
                <TableCell>
                  <TableCellLayout>
                    {item.userName}
                  </TableCellLayout>
                </TableCell>
                <TableCell>
                  <TableCellLayout>
                    {item.nickName}
                  </TableCellLayout>
                </TableCell>
                <TableCell>
                  <TableCellLayout>
                    {item.nOrder}
                  </TableCellLayout>
                </TableCell>
                <TableCell>
                  <TableCellLayout>
                    {item.nTime}
                  </TableCellLayout>
                </TableCell>
                <TableCell>
                  <TableCellLayout>
                    {item.content}
                  </TableCellLayout>
                </TableCell>
                <TableCell>
                  <TableCellLayout>
                    <Link onClick={() => {
                      setCurrentChatSessionItem(item);
                      setIsOpen(true);
                    }}>View Details</Link>
                  </TableCellLayout>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <div className={styles.pagination}>
          <Button appearance="primary" onClick={() => setOffset(offset - limit)} disabled={offset === 0}>Previous</Button>
          <Button appearance="primary" onClick={() => setOffset(offset + limit)}>Next</Button>
          <Dropdown id={dropdownId} className={styles.dropdown}
            placeholder="limit"
            onOptionSelect={handleOnOptionSelect}
            defaultSelectedOptions={[`${limit}`]}
            defaultValue={`${limit}`}
          >
            {options.map((option) => (
              <Option key={option} value={option}>
                {option}
              </Option>
            ))}
          </Dropdown>
        </div>

        <ChatlogDrawer
          isOpen={isOpen}
          setIsOpen={setIsOpen}
          currentChatSessionItem={currentChatSessionItem!}
        />
      </div>
    );
}
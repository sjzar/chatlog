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
import { getChatRoom } from '../WeChatService';
import { ChatRoomData, ChatRoomItem, ChatRoomUser, GetDataParams } from '../typing';
import toast, { Toaster } from 'react-hot-toast';
import { UserDrawer } from './components/UserDrawer';

const columns = [
  { columnKey: "name", label: "ChatRoomName" },
  { columnKey: "nickName", label: "NickName" },
  { columnKey: "owner", label: "Owner" },
  { columnKey: "users", label: "Users" },
  { columnKey: "remark", label: "Remark" },
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

export function ChatRoomPage() {
  const [limit, setLimit] = React.useState(10);
  const [offset, setOffset] = React.useState(0);
  const { run } = useRequest<GetDataParams, ChatRoomData>(params => getChatRoom(params!));
  const [items, setItems] = React.useState<ChatRoomItem[]>([]);
  const [users, setUsers] = React.useState<ChatRoomUser[]>([]);
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

  React.useEffect(() => {
    loadData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [limit, offset]);

  const loadData = async () => {
    const loadingToastId = toast.loading("Loading chat room...");
    run({ limit, offset }).then(data => {
      setItems(data.items);
    }).catch(error => {
      console.error(error);
      toast.error("Failed to load chat room data.");
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
            <TableRow key={item.name}>
              <TableCell>
                <TableCellLayout>
                  {item.name}
                </TableCellLayout>
              </TableCell>
              <TableCell>
                <TableCellLayout>
                  {item.nickName}
                </TableCellLayout>
              </TableCell>
              <TableCell>
                <TableCellLayout>
                  {item.owner}
                </TableCellLayout>
              </TableCell>
              <TableCell>
                <TableCellLayout>
                  <Link onClick={() => {
                    setUsers(item.users);
                    setIsOpen(true);
                  }}>View Details</Link>
                </TableCellLayout>
              </TableCell>
              <TableCell>
                <TableCellLayout>
                  {item.remark}
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

      <UserDrawer isOpen={isOpen} setIsOpen={setIsOpen} users={users} />
    </div>
  );
}
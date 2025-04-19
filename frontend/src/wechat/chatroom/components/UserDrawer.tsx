import { ChatRoomUser } from '@/wechat/typing';
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

const useStyles = makeStyles({
  root: {
    width: '500px',
  },
});

type UserDrawerProps = {
  isOpen: boolean;
  setIsOpen: (isOpen: boolean) => void;
  users: ChatRoomUser[];
}

export function UserDrawer(props: UserDrawerProps) {
  const { isOpen, setIsOpen, users } = props;

  const styles = useStyles();
  const restoreFocusSourceAttributes = useRestoreFocusSource();

  const handleOnOpenChange = (_: DialogOpenChangeEvent, { open }: DialogOpenChangeData) => {
    setIsOpen(open);
  };

  return (
    <OverlayDrawer
      modalType="non-modal"
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
          Chat Room Users
        </DrawerHeaderTitle>
      </DrawerHeader>

      <DrawerBody>
        <ul>
          {users.map(user => (
            <li key={user.userName}>{user.userName}({user.displayName.length === 0 ? '无' : user.displayName})</li>
          ))}
        </ul>
      </DrawerBody>
    </OverlayDrawer>
  );
}
import React from "react";
import {
  makeStyles,
  tokens,
  SelectTabEventHandler,
  Tab,
  TabList
} from "@fluentui/react-components";
import {
  ContactCardGroupFilled,
  ChatFilled,
  ChatMultipleFilled,
} from "@fluentui/react-icons";
import { useLocation, useNavigate } from "react-router";

const useStyles = makeStyles({
  header: {
    backgroundColor: tokens.colorBrandBackground2,
  },
});

export function Header() {
  const [selectedValue, setSelectedValue] = React.useState("contact");
  const navigate = useNavigate();
  const styles = useStyles();
  const location = useLocation();

  React.useEffect(() => {
    const path = location.pathname.split("/")[1];
    if (path) {
      setSelectedValue(path);
    }
  }, [location]);

  const handleOnTabSelect: SelectTabEventHandler = (_, data) => {
    navigate(`/${data.value}`);
  }

  return (
    <header className={styles.header}>
      <TabList selectedValue={selectedValue} size="large" onTabSelect={handleOnTabSelect}>
        <Tab icon={<ContactCardGroupFilled />} value="contact">
          Contact
        </Tab>
        <Tab icon={<ChatFilled />} value="chatroom">
          Chat Room
        </Tab>
        <Tab icon={<ChatMultipleFilled />} value="session">
          Sessions
        </Tab>
      </TabList>
    </header>
  );
}

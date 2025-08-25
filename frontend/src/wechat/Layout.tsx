import { useOutlet } from "react-router";
import { Header } from '@/wechat/components';

export default function Layout() {
  const outlet = useOutlet();

  return (
    <div>
      <Header />

      <main>
        {outlet}
      </main>
    </div>
  );
}

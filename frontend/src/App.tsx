import { RouterProvider } from 'react-router';
import { RecoilRoot } from 'recoil';
import { FluentProvider, teamsLightTheme } from '@fluentui/react-components';
import router from './router';
import './App.css';

function App() {
  return (
    <RecoilRoot>
      <FluentProvider theme={teamsLightTheme} className="app">
        <RouterProvider router={router} />
      </FluentProvider>
    </RecoilRoot>
  );
}

export default App

import {render, waitFor} from 'test-utils';
import Header from '../Header';

it('Header', async () => {
  const {getByTestId} = render(<Header />);
  await waitFor(() => getByTestId('menu-link'));
});

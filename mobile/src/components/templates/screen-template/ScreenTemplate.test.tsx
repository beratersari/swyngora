import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Text } from '@/components/atoms/text';
import { ScreenTemplate } from './ScreenTemplate';

describe('ScreenTemplate', () => {
  it('renders title children and footer', () => {
    render(
      <ScreenTemplate title="Markets" footer={<Text>Footer</Text>}>
        <Text>Body</Text>
      </ScreenTemplate>,
    );
    expect(screen.getByText('Markets')).toBeTruthy();
    expect(screen.getByText('Body')).toBeTruthy();
    expect(screen.getByText('Footer')).toBeTruthy();
  });
});

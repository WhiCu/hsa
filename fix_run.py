import os

directory = 'internal/application/'
for filename in os.listdir(directory):
    if filename.endswith('_test.go'):
        filepath = os.path.join(directory, filename)
        with open(filepath, 'r') as f:
            content = f.read()

        # Replace transactor.EXPECT().RunInTransaction(...) Return(...)
        # with Run(...) for function callbacks, which is how mockery with EXPECT handles function args,
        # or just change back to .On("RunInTransaction").Return(...) just for this.
        content = content.replace('.EXPECT().RunInTransaction(', '.On("RunInTransaction", ')

        with open(filepath, 'w') as f:
            f.write(content)
import os

filepath = 'internal/application/finish_invite_registration_test.go'
with open(filepath, 'r') as f:
    lines = f.readlines()

new_lines = []
for line in lines:
    if 'It("Initial SignCount Preservation on Registration"' in line:
        pass # we will manually insert it
    new_lines.append(line)

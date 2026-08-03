#!/bin/bash
for f in internal/application/*_test.go; do
  sed -i 's/\.EXPECT()\./\.EXPECT()./g' $f
done

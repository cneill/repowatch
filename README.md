# repowatch

This tool lets you visualize who is committing to git repos over time. It collects all the pairs of name + email
associated with commits, assigns a unique identifier + color, and prints the symbol for the owner of each commit in the
repository.

You can also use the `-committer` flag to see who _committed_ the change.

## Example

The infamous `xz` repo with the `-committer` flag set.

![Example of repowatch output in the xz repo](./example.png)

## Usage

```
./repowatch [-committer] [repo path]
```

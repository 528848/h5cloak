use strict;
use warnings;
use Digest::SHA qw(sha256_hex);
binmode STDIN;
binmode STDOUT;
$| = 1;
my $D = $ENV{CLOAK_PARTS} || '/opt/cloak-releases/parts';
system('rm', '-rf', $D);
mkdir $D;

sub readn {
    my ($n) = @_;
    my $buf = '';
    while (length($buf) < $n) {
        my $got = sysread(STDIN, my $chunk, $n - length($buf));
        die "eof\n" unless $got;
        $buf .= $chunk;
    }
    return $buf;
}

sub readline_raw {
    my $buf = '';
    while (1) {
        my $got = sysread(STDIN, my $c, 1);
        return $buf if !$got || $c eq "\n";
        $buf .= $c if $c ne "\r";
    }
}

print "READY\n";
while (1) {
    my $line = readline_raw();
    $line =~ s/^\xef\xbb\xbf//;
    next if $line eq '';
    my ($op, $a, $b) = split / /, $line, 3;
    if ($op eq 'DONE') { print "BYE\n"; exit 0; }
    if ($op eq 'PUT') {
        my $data = readn($b + 0);
        open my $fh, '>:raw', "$D/$a" or die $!;
        print {$fh} $data;
        close $fh;
        print "HASH $a ", sha256_hex($data), "\n";
        next;
    }
    if ($op eq 'RUN') {
        my $data = readn($a + 0);
        open my $fh, '>:raw', '/tmp/cloak-install.sh' or die $!;
        print {$fh} $data;
        close $fh;
        exit(system('bash', '/tmp/cloak-install.sh') >> 8);
    }
    print "ERR $line\n";
}

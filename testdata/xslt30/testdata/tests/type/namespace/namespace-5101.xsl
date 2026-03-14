<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:foo="http://aaa/">

<?spec xslt#lre-namespaces?>
    <!-- Purpose: Simple case of creating LRE with nested namespace declarations. -->

<xsl:template match="/">
    <foo:stuff xmlns:foo="http://bbb/">
      <foo:stuff xmlns:foo="http://ccc/"/>
    </foo:stuff>
</xsl:template>

</xsl:stylesheet>



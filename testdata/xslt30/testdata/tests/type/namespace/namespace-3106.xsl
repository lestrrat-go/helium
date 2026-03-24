<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
<?error ?>
    <!-- Purpose: Put an unprefixed xmlns declaration in xsl:element where requested name has prefix. -->
  <!-- Should see one warning about namespace "abc" unresolvable.
       Recovery: put yyy directly inside higher element (out). -->

<xsl:template match = "/">
  <xsl:param name="prefix" select="'abc'"/>
  <out>
    <xsl:element name="{$prefix}c:foo" xmlns="">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>
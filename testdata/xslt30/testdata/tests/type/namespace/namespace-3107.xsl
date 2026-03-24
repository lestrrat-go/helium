<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
<?error ?>
    <!-- Purpose: Have a non-empty but unprefixed xmlns declaration while specifying prefixed name. -->
  <!-- Should see one warning about namespace "wxyz" unresolvable.
       Recovery: put yyy directly inside higher element (out). -->

<xsl:template match = "/">
  <xsl:param name="prefix" select="'wxyz'"/>
  <out>
    <xsl:element name="{$prefix}:foo" xmlns="http://test.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>
<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
<?error ?>
  <!-- Discretionary: name="element-name-not-QName" choice="pass-through" -->
    <!-- Purpose: Test for resetting of a prefixed namespace by a LRE. -->
  <!-- Should see one warning about namespace "none" unresolvable.
       Recovery: put yyy directly inside higher element (out). -->

<xsl:template match = "/">
  <xsl:param name="prefix" select="'none'"/>
  <out>
    <xsl:element name="{$prefix}:foo">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>
<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns="http://testguys.com/">

<?spec xslt#xsl-element?>
<?error ?>
    <!-- Purpose: Test for resetting of a prefixed namespace by a LRE; stylesheet default NS set. -->
  <!-- Should see one warning about namespace "none" unresolvable.
       Recovery: put yyy directly inside higher element (out). -->
  <!-- MHK: changed to make it a dynamic error -->

<xsl:template match = "/">
  <xsl:param name="prefix" select="'none'"/>
  <out>
    <xsl:element name="{concat($prefix, ':foo')}">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>
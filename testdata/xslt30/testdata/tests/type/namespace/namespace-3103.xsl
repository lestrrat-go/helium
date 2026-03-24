<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
<?error ?>
  <!-- Discretionary: name="element-name-not-QName" choice="pass-through" -->
    <!-- Purpose: Test for non-empty illegal element name. -->
  <!-- Should see a warning about illegal element name. Recovery: put yyy directly in out. -->
  <!-- MHK: changed to make it a dynamic error, not a static one -->
<xsl:template match = "/">
<xsl:param name="n" select="'this is a string'"/>
  <out>
    <xsl:element name="{$n}">
      <yyy xmlns=""/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>
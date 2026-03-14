<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
  <!-- Purpose: Tests apply-templates starting with a attribute axis followed
       by additional relative-location-path steps. -->
  
<xsl:template match="/">
  <out>
    <xsl:for-each select="//center">
      <xsl:apply-templates select="@*/following::text()"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="text()">
  <xsl:text> #text</xsl:text>
</xsl:template>

</xsl:stylesheet>
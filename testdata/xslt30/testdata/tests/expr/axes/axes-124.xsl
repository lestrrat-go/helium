<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
  <!-- Purpose: Tests apply-templates from a comment using parent axis. -->
  
<xsl:template match="/">
  <out>
    <xsl:for-each select="//center/comment()">
      <xsl:apply-templates select=".."/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="comment()">
  <xsl:value-of select="."/>
</xsl:template>

<xsl:template match="text()"/><!-- suppress whitespace-only nodes -->

</xsl:stylesheet>
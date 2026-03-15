<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#global-variables?>
  <!-- Purpose: Because root node is current when top-level param is set, can get top-level comment. -->
  

<xsl:param name="z" select="comment()"/>

<xsl:template match="doc">
  <out>
    <xsl:text>$z contains</xsl:text>
    <xsl:value-of select="$z"/>
  </out>
</xsl:template>

</xsl:stylesheet>

<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#global-variables?>
  <!-- Purpose: Show that there is always a current node -->
  

<xsl:param name="children" select="count(*)"/>

<xsl:template match="docs">
  <out>
    <xsl:text>$children contains </xsl:text>
    <xsl:value-of select="$children"/>
  </out>
</xsl:template>

</xsl:stylesheet>

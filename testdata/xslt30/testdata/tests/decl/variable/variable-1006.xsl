<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Set variable to null string explicitly and implicitly, compare. -->
 
<xsl:variable name="empty" />

<xsl:template match="doc">
  <xsl:param name="x" select="substring-before('a','z')" />
  <out>
    <xsl:value-of select="$x=$empty"/>
  </out>
</xsl:template>

</xsl:stylesheet>

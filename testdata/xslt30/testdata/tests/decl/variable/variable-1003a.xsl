<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: var11imp -->
  <!-- Purpose: to be imported by variable11 -->

<xsl:param name="test" select="'subsheet, should be overridden by main sheet'"/>

<xsl:template match="/">
  <out>
    <xsl:value-of select="$test"/>
  </out>
</xsl:template>

</xsl:stylesheet>
<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on general comparisons -->

  <xsl:template name="main">
    <xsl:variable name="date"><xsl:value-of select="current-date()"/></xsl:variable>
    <out><xsl:value-of select="(1 to 5) = ('apple', 'banana', 'cherry', 'damson', '5.00e0')"/></out>
  </xsl:template>


</xsl:stylesheet>

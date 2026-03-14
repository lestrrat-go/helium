<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Set top-level xsl:param to a string -->
  
<xsl:param name="ExpressionTest" select="'ABC'"/>

<xsl:template match="doc">
   <out>
      <xsl:value-of select="$ExpressionTest"/>
   </out>
</xsl:template>

</xsl:stylesheet>
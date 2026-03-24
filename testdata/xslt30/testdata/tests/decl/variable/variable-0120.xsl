<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Within an overriding global variable, attempt to reference the overridden variable -->
  
  <xsl:import href="variable-0120a.xsl"/>

  <xsl:variable name="v" select="$v+1"/>
  
  <xsl:template match="/">
    <out><xsl:value-of select="$v"/></out>
  </xsl:template>

</xsl:stylesheet>

<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: A processor that does not support backwards compatibility mode raises no error
    if no 1.0 code is executed. -->

  <xsl:template name="main" version="2.0">
    <out>
      <xsl:value-of select="23"/>
    </out>
  </xsl:template>
  
  <xsl:template name="other" version="1.0">
    <out>
      <xsl:value-of select="25"/>
    </out>
  </xsl:template>

</xsl:stylesheet>

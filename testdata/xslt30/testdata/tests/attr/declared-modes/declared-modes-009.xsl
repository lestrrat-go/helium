<?xml version="1.0"?> 
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
declared-modes='false'>

  <!-- PURPOSE:  Use of undeclared mode on xsl:template is not an error (declared-modes = false). -->

  <xsl:template match="text()" mode="a">
    <xsl:text>mode-a:</xsl:text>
    <xsl:value-of select="."/>
  </xsl:template>

  <xsl:template name="main" visibility="public">
    <a/>
  </xsl:template>

</xsl:package>

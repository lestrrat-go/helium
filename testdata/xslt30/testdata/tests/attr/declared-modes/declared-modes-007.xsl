<?xml version="1.0"?> 
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
declared-modes='1'>

  <!-- PURPOSE:  Use of unnamed mode on xsl:template is an error (declared-modes = true). -->

  <xsl:template name="main" match="/" visibility="public">
    <xsl:text>no-mode:</xsl:text>
    <xsl:value-of select="24"/>
  </xsl:template>

</xsl:package>

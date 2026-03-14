<?xml version="1.0"?> 
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- PURPOSE:  Use of unnamed mode on xsl:apply-templates is an error (declared-modes defaults to true). -->

  <xsl:template name="main" visibility="public">
    <xsl:text>no-mode:</xsl:text>
    <xsl:apply-templates select="17"/>
    <xsl:value-of select="24"/>
  </xsl:template>

</xsl:package>

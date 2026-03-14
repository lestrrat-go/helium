<?xml version="1.0"?> 
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
declared-modes=' no '>

  <!-- PURPOSE:  Use of undeclared mode on xsl:apply-templates is not an error (declared-modes = false). -->

  <xsl:template name="main" visibility="public">
    <xsl:apply-templates select="()" mode="absent"/>
    <a/>
  </xsl:template>

</xsl:package>

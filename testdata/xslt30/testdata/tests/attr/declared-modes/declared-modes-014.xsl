<?xml version="1.0"?> 
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
declared-modes=' yes '>

  <!-- PURPOSE:  Use of undeclared mode on xsl:apply-templates is an error (declared-modes = true). -->

  <xsl:template name="main" visibility="public">
    <xsl:apply-templates select="()" mode="#unnamed"/>
    <a/>
  </xsl:template>

</xsl:package>

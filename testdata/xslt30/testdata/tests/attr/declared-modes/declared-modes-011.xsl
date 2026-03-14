<?xml version="1.0"?> 
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
declared-modes='0'>

  <!-- PURPOSE:  Use of unnamed mode on xsl:template is not an error (declared-modes = false). -->

  <xsl:template name="main" match="/" visibility="public">
    <a/>
  </xsl:template>

</xsl:package>

<?xml version="1.0"?> 

<!-- try006 -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:f="http://localhost/functions" exclude-result-prefixes=" f">

  <!-- Test xsl:try  -->

  <xsl:template name="main">
    <out>
      <xsl:try select="1 div 0">
        <xsl:catch errors="*" select="'Infinity'"/>
      </xsl:try>
    </out>
  </xsl:template>

</xsl:stylesheet>

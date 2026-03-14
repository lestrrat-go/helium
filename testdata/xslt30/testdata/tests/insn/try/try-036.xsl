<?xml version="1.0"?> 

<!-- try007 -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:err="http://www.w3.org/2005/xqt-errors" exclude-result-prefixes=" err">

  <!-- Show that xpath-default-namespace does not apply -->


  <xsl:template name="main" xpath-default-namespace="http://www.w3.org/2005/xqt-errors">
    <out>
      <xsl:try select="1 div 0">
        <xsl:catch errors="FOAR0001" select="'Should not be caught'"/>
        <xsl:catch errors="*" select="'OK'"/>
      </xsl:try>
    </out>
  </xsl:template>

</xsl:stylesheet>

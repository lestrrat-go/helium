<?xml version="1.0"?> 

<!-- try008 -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:err="http://www.w3.org/2005/xqt-errors" exclude-result-prefixes="err">

  <!-- Catch multiple errors -->


  <xsl:template name="main">
    <out>
      <xsl:try select="1 div 0">
        <xsl:catch errors="err:FOAR9876 err:FOAR0001" select="'Infinity'"/>
        <xsl:catch errors="*" select="'Unexpected error'"/>
      </xsl:try>
    </out>
  </xsl:template>

</xsl:stylesheet>

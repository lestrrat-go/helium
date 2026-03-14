<?xml version="1.0"?> 

<!-- try008 -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:err="http://www.w3.org/2005/xqt-errors" exclude-result-prefixes="err">

  <!-- Catch multiple errors, with sequence constructor -->

  <xsl:template name="main">
    <out>
      <xsl:try>
        <xsl:attribute name="test" select="1 div 0"/>
        <xsl:catch errors="err:FOAR9876 err:FOAR0001">
          <xsl:attribute name="test" select="'Infinity'"/>
        </xsl:catch>
        <xsl:catch errors="*" select="'Unexpected error'"/>
      </xsl:try>
    </out>
  </xsl:template>

</xsl:stylesheet>
